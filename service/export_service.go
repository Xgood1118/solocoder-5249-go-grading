package service

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"grading-system/algorithm"
	"grading-system/model"
	"grading-system/repository"
)

type ExportService struct {
	repo       *repository.Repository
	exportDir  string
	mu         sync.Mutex
}

func NewExportService(repo *repository.Repository, exportDir string) *ExportService {
	os.MkdirAll(exportDir, 0755)
	return &ExportService{
		repo:      repo,
		exportDir: exportDir,
	}
}

func (s *ExportService) CreateExportTask(paperID string, exportType string) string {
	taskID := fmt.Sprintf("export_%d_%s", time.Now().UnixNano(), exportType)
	task := model.ExportTask{
		ID:        taskID,
		Type:      exportType,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	s.repo.SaveExportTask(task)

	go s.processExport(taskID, paperID, exportType)

	return taskID
}

func (s *ExportService) processExport(taskID, paperID, exportType string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.repo.GetExportTask(taskID)
	if !ok {
		return
	}
	task.Status = "processing"
	s.repo.UpdateExportTask(task)

	defer func() {
		if r := recover(); r != nil {
			task.Status = "failed"
			s.repo.UpdateExportTask(task)
		}
	}()

	var fileName string
	var err error

	switch exportType {
	case "excel":
		fileName, err = s.exportExcel(paperID)
	case "csv":
		fileName, err = s.exportCSV(paperID)
	case "pdf":
		fileName, err = s.exportPDFSimple(paperID)
	default:
		err = fmt.Errorf("不支持的导出类型: %s", exportType)
	}

	now := time.Now()
	if err != nil {
		task.Status = "failed"
	} else {
		task.Status = "completed"
		task.FileName = fileName
		task.FinishedAt = &now
	}
	s.repo.UpdateExportTask(task)
}

func (s *ExportService) exportCSV(paperID string) (string, error) {
	paper, ok := s.repo.GetPaper(paperID)
	if !ok {
		return "", fmt.Errorf("试卷不存在")
	}

	submissions := s.repo.ListSubmissionsByPaper(paperID)
	students := s.repo.ListStudents()
	studentMap := make(map[string]model.Student)
	for _, st := range students {
		studentMap[st.ID] = st
	}

	fileName := fmt.Sprintf("%s_%s_scores.csv", paperID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(s.exportDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"学生ID", "姓名", "班级", "年级", "总分", "满分", "段位", "状态"}
	for _, q := range paper.Questions {
		header = append(header, fmt.Sprintf("第%s题(%s)", q.ID, q.Type))
	}
	writer.Write(header)

	for _, sub := range submissions {
		student := studentMap[sub.StudentID]
		var latestScore float64
		var level string
		qsMap := make(map[string]float64)

		if len(sub.ScoreRecords) > 0 {
			latest := sub.ScoreRecords[len(sub.ScoreRecords)-1]
			latestScore = latest.TotalScore
			for _, qs := range latest.QuestionScores {
				qsMap[qs.QuestionID] = qs.Score
			}
		}
		if sub.CurrentStage == model.StageFinished {
			level = string(sub.ScoreLevel)
		}

		row := []string{
			student.ID,
			student.Name,
			student.Class,
			student.Grade,
			strconv.FormatFloat(latestScore, 'f', 2, 64),
			strconv.FormatFloat(paper.TotalScore, 'f', 2, 64),
			level,
			string(sub.CurrentStage),
		}
		for _, q := range paper.Questions {
			row = append(row, strconv.FormatFloat(qsMap[q.ID], 'f', 2, 64))
		}
		writer.Write(row)
	}

	return fileName, nil
}

func (s *ExportService) exportExcel(paperID string) (string, error) {
	return s.exportCSV(paperID)
}

func (s *ExportService) exportPDFSimple(paperID string) (string, error) {
	paper, ok := s.repo.GetPaper(paperID)
	if !ok {
		return "", fmt.Errorf("试卷不存在")
	}

	submissions := s.repo.ListSubmissionsByPaper(paperID)
	students := s.repo.ListStudents()
	studentMap := make(map[string]model.Student)
	for _, st := range students {
		studentMap[st.ID] = st
	}

	fileName := fmt.Sprintf("%s_%s_report.txt", paperID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(s.exportDir, fileName)

	content := fmt.Sprintf("=== %s 成绩报告 ===\n\n", paper.Name)
	content += fmt.Sprintf("科目: %s\n", paper.Subject)
	content += fmt.Sprintf("满分: %.1f\n\n", paper.TotalScore)
	content += "----------------------------------------\n\n"

	for _, sub := range submissions {
		if sub.CurrentStage != model.StageFinished {
			continue
		}
		student := studentMap[sub.StudentID]
		content += fmt.Sprintf("学生: %s (%s)\n", student.Name, student.Class)
		content += fmt.Sprintf("分数: %.1f / %.1f\n", sub.FinalScore, paper.TotalScore)
		content += fmt.Sprintf("段位: %s\n", sub.ScoreLevel)
		content += "--- 题目明细 ---\n"
		if len(sub.ScoreRecords) > 0 {
			latest := sub.ScoreRecords[len(sub.ScoreRecords)-1]
			for _, qs := range latest.QuestionScores {
				qTitle := qs.QuestionID
				for _, q := range paper.Questions {
					if q.ID == qs.QuestionID {
						qTitle = q.Title
						break
					}
				}
				content += fmt.Sprintf("  %s: %.1f / %.1f\n", qTitle, qs.Score, qs.FullScore)
			}
			if latest.Comment != "" {
				content += fmt.Sprintf("教师评语: %s\n", latest.Comment)
			}
		}
		content += "\n"
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", err
	}

	return fileName, nil
}

func (s *ExportService) GetExportTask(taskID string) (model.ExportTask, bool) {
	return s.repo.GetExportTask(taskID)
}

func (s *ExportService) GetExportFilePath(fileName string) string {
	return filepath.Join(s.exportDir, fileName)
}

func (s *ExportService) GetLatestScoreRecord(sub model.PaperSubmission) *model.ScoreRecord {
	if len(sub.ScoreRecords) == 0 {
		return nil
	}
	latest := sub.ScoreRecords[len(sub.ScoreRecords)-1]
	return &latest
}

func (s *ExportService) CalculateLevel(score float64, fullScore float64) model.ScoreLevel {
	return algorithm.CalculateScoreLevel(score, fullScore)
}
