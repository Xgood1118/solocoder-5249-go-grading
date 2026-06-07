package service

import (
	"fmt"
	"time"

	"grading-system/algorithm"
	"grading-system/model"
	"grading-system/repository"
)

type SubmissionService struct {
	repo *repository.Repository
}

func NewSubmissionService(repo *repository.Repository) *SubmissionService {
	return &SubmissionService{repo: repo}
}

func (s *SubmissionService) GetRepo() *repository.Repository {
	return s.repo
}

func (s *SubmissionService) RunInitialGrading(subID string) error {
	sub, ok := s.repo.GetSubmission(subID)
	if !ok {
		return fmt.Errorf("答卷不存在: %s", subID)
	}

	paper, ok := s.repo.GetPaper(sub.PaperID)
	if !ok {
		return fmt.Errorf("试卷不存在: %s", sub.PaperID)
	}

	questionScores, totalScore := GradePaper(paper, sub.Answers)

	record := model.ScoreRecord{
		ID:             fmt.Sprintf("score_%s_initial", subID),
		Stage:          model.StageInitial,
		Grader:         "system",
		TotalScore:     totalScore,
		QuestionScores: questionScores,
		Comment:        "系统自动初评",
		Timestamp:      time.Now(),
	}

	sub.ScoreRecords = append(sub.ScoreRecords, record)
	sub.CurrentStage = model.StageReview
	s.repo.SaveSubmission(sub)

	return nil
}

func (s *SubmissionService) GetSubmissionsByStage(paperID string, stage model.GradingStage) []model.PaperSubmission {
	all := s.repo.ListSubmissionsByPaper(paperID)
	var result []model.PaperSubmission
	for _, sub := range all {
		if sub.CurrentStage == stage {
			result = append(result, sub)
		}
	}
	return result
}

func (s *SubmissionService) GetGradingProgress(paperID string) map[string]int {
	all := s.repo.ListSubmissionsByPaper(paperID)
	result := map[string]int{
		"total":    len(all),
		"initial":  0,
		"review":   0,
		"final":    0,
		"finished": 0,
	}
	for _, sub := range all {
		switch sub.CurrentStage {
		case model.StageInitial:
			result["initial"]++
		case model.StageReview:
			result["review"]++
		case model.StageFinal:
			result["final"]++
		case model.StageFinished:
			result["finished"]++
		}
	}
	return result
}

func (s *SubmissionService) LockSubmission(subID, teacher string) bool {
	return s.repo.LockSubmission(subID, teacher)
}

func (s *SubmissionService) UnlockSubmission(subID, teacher string) bool {
	return s.repo.UnlockSubmission(subID, teacher)
}

func (s *SubmissionService) SubmitReview(subID, teacher string, questionScores []model.QuestionScore, comment string, stage model.GradingStage) error {
	sub, ok := s.repo.GetSubmission(subID)
	if !ok {
		return fmt.Errorf("答卷不存在")
	}

	if sub.LockedBy == "" {
		s.repo.LockSubmission(subID, teacher)
	} else if sub.LockedBy != teacher {
		return fmt.Errorf("答卷已被其他老师锁定")
	}

	if sub.CurrentStage != stage {
		return fmt.Errorf("当前阶段不匹配")
	}

	totalScore := 0.0
	for _, qs := range questionScores {
		totalScore += qs.Score
	}

	recordID := fmt.Sprintf("score_%s_%s_%d", subID, stage, len(sub.ScoreRecords))
	record := model.ScoreRecord{
		ID:             recordID,
		Stage:          stage,
		Grader:         teacher,
		TotalScore:     totalScore,
		QuestionScores: questionScores,
		Comment:        comment,
		Timestamp:      time.Now(),
	}

	sub.ScoreRecords = append(sub.ScoreRecords, record)

	switch stage {
	case model.StageReview:
		sub.CurrentStage = model.StageFinal
	case model.StageFinal:
		sub.CurrentStage = model.StageFinished
		paper, _ := s.repo.GetPaper(sub.PaperID)
		sub.FinalScore = totalScore
		sub.ScoreLevel = algorithm.CalculateScoreLevel(totalScore, paper.TotalScore)
		s.repo.SaveSnapshot(sub)
	}

	sub.LockedBy = ""
	sub.LockedAt = nil

	s.repo.UpdateSubmission(sub)
	return nil
}

func (s *SubmissionService) GetSubmissionDetail(subID string) (map[string]interface{}, error) {
	sub, ok := s.repo.GetSubmission(subID)
	if !ok {
		return nil, fmt.Errorf("答卷不存在")
	}

	paper, ok := s.repo.GetPaper(sub.PaperID)
	if !ok {
		return nil, fmt.Errorf("试卷不存在")
	}

	student, ok := s.repo.GetStudent(sub.StudentID)
	if !ok {
		return nil, fmt.Errorf("学生不存在")
	}

	var latestScoreRecord *model.ScoreRecord
	if len(sub.ScoreRecords) > 0 {
		latest := sub.ScoreRecords[len(sub.ScoreRecords)-1]
		latestScoreRecord = &latest
	}

	result := map[string]interface{}{
		"submission": sub,
		"paper":      paper,
		"student":    student,
	}

	if latestScoreRecord != nil {
		result["latest_score"] = latestScoreRecord

		qsMap := make(map[string]model.QuestionScore)
		for _, qs := range latestScoreRecord.QuestionScores {
			qsMap[qs.QuestionID] = qs
		}
		result["question_scores"] = qsMap

		ansMap := make(map[string]interface{})
		for _, a := range sub.Answers {
			ansMap[a.QuestionID] = a.Content
		}
		result["answers"] = ansMap
	}

	return result, nil
}

func (s *SubmissionService) ListAllPapers() []model.Paper {
	return s.repo.ListPapers()
}

func (s *SubmissionService) GetPaperWithProgress(paperID string) (map[string]interface{}, error) {
	paper, ok := s.repo.GetPaper(paperID)
	if !ok {
		return nil, fmt.Errorf("试卷不存在")
	}

	progress := s.GetGradingProgress(paperID)
	submissions := s.repo.ListSubmissionsByPaper(paperID)

	students := s.repo.ListStudents()
	studentMap := make(map[string]model.Student)
	for _, st := range students {
		studentMap[st.ID] = st
	}

	subList := make([]map[string]interface{}, 0, len(submissions))
	for _, sub := range submissions {
		item := map[string]interface{}{
			"submission": sub,
			"student":    studentMap[sub.StudentID],
		}
		subList = append(subList, item)
	}

	return map[string]interface{}{
		"paper":       paper,
		"progress":    progress,
		"submissions": subList,
		"students":    studentMap,
	}, nil
}
