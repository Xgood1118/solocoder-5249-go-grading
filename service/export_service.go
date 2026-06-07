package service

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"grading-system/model"
	"grading-system/repository"
)

const SchoolName = "启明教育培训中心"

type ExportService struct {
	repo      *repository.Repository
	exportDir string
	mu        sync.Mutex
}

func NewExportService(repo *repository.Repository, exportDir string) *ExportService {
	os.MkdirAll(exportDir, 0755)
	return &ExportService{
		repo:      repo,
		exportDir: exportDir,
	}
}

func (s *ExportService) CreateExportTask(paperID string, exportType string, studentID string) string {
	taskID := fmt.Sprintf("export_%d_%s_%s", time.Now().UnixNano(), exportType, paperID)
	task := model.ExportTask{
		ID:        taskID,
		Type:      exportType,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	s.repo.SaveExportTask(task)

	go s.processExport(taskID, paperID, exportType, studentID)

	return taskID
}

func (s *ExportService) processExport(taskID, paperID, exportType, studentID string) {
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

	paper, ok := s.repo.GetPaper(paperID)
	if !ok {
		task.Status = "failed"
		s.repo.UpdateExportTask(task)
		return
	}

	submissions := s.repo.ListSubmissionsByPaper(paperID)
	students := s.repo.ListStudents()
	studentMap := make(map[string]model.Student)
	for _, st := range students {
		studentMap[st.ID] = st
	}

	paperData := PaperExportData{
		Paper:       paper,
		Submissions: submissions,
		Students:    studentMap,
		SchoolName:  SchoolName,
	}

	switch exportType {
	case "excel":
		fileName, err = ExportExcel(paperData, s.exportDir)
	case "csv":
		fileName, err = s.exportCSV(paperData)
	case "pdf":
		if studentID != "" {
			sub, ok := s.repo.GetSubmissionByPaperAndStudent(paperID, studentID)
			if !ok {
				task.Status = "failed"
				s.repo.UpdateExportTask(task)
				return
			}
			stu := studentMap[studentID]
			stuData := StudentExportData{
				Paper:      paper,
				Submission: sub,
				Student:    stu,
				SchoolName: SchoolName,
			}
			fileName, err = ExportStudentPDF(stuData, s.exportDir)
		} else {
			fileName, err = s.exportAllPDFs(paperData)
		}
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

func (s *ExportService) exportCSV(data PaperExportData) (string, error) {
	fileName := fmt.Sprintf("%s_成绩表_%s.csv", data.Paper.ID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(s.exportDir, fileName)

	f, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// 用 UTF-8 BOM 防止 Excel 打开乱码
	f.WriteString("\xEF\xBB\xBF")

	header := "序号,学号,姓名,班级,年级,总分,满分,段位,状态"
	for _, q := range data.Paper.Questions {
		header += fmt.Sprintf(",第%s题(%s/%.0f分)", q.ID, q.Type, q.FullScore)
	}
	header += "\n"
	f.WriteString(header)

	for idx, sub := range data.Submissions {
		stu := data.Students[sub.StudentID]
		var totalScore float64
		level := ""
		qsMap := make(map[string]float64)

		if len(sub.ScoreRecords) > 0 {
			latest := sub.ScoreRecords[len(sub.ScoreRecords)-1]
			totalScore = latest.TotalScore
			for _, qs := range latest.QuestionScores {
				qsMap[qs.QuestionID] = qs.Score
			}
		}
		if sub.CurrentStage == model.StageFinished {
			level = string(sub.ScoreLevel)
		}

		row := fmt.Sprintf("%d,%s,%s,%s,%s,%.1f,%.1f,%s,%s",
			idx+1, stu.ID, stu.Name, stu.Class, stu.Grade,
			totalScore, data.Paper.TotalScore, level, sub.CurrentStage)
		for _, q := range data.Paper.Questions {
			row += fmt.Sprintf(",%.1f", qsMap[q.ID])
		}
		row += "\n"
		f.WriteString(row)
	}

	return fileName, nil
}

func (s *ExportService) exportAllPDFs(data PaperExportData) (string, error) {
	tempDir := filepath.Join(s.exportDir, fmt.Sprintf("pdf_temp_%d", time.Now().UnixNano()))
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	for _, sub := range data.Submissions {
		stu := data.Students[sub.StudentID]
		stuData := StudentExportData{
			Paper:      data.Paper,
			Submission: sub,
			Student:    stu,
			SchoolName: data.SchoolName,
		}
		_, err := ExportStudentPDF(stuData, tempDir)
		if err != nil {
			return "", err
		}
	}

	zipName := fmt.Sprintf("%s_学生成绩单_批量_%s.zip", data.Paper.ID, time.Now().Format("20060102_150405"))
	zipPath := filepath.Join(s.exportDir, zipName)

	err := zipDirectory(tempDir, zipPath)
	if err != nil {
		return "", err
	}

	return zipName, nil
}

func zipDirectory(sourceDir, zipFile string) error {
	archive, err := os.Create(zipFile)
	if err != nil {
		return err
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		srcPath := filepath.Join(sourceDir, file.Name())
		srcFile, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		w, err := zipWriter.Create(file.Name())
		if err != nil {
			return err
		}
		_, err = io.Copy(w, srcFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ExportService) GetExportTask(taskID string) (model.ExportTask, bool) {
	return s.repo.GetExportTask(taskID)
}

func (s *ExportService) GetExportFilePath(fileName string) string {
	return filepath.Join(s.exportDir, fileName)
}
