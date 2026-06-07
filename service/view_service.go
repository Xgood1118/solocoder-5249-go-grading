package service

import (
	"fmt"
	"strings"

	"grading-system/model"
)

type QuestionOptionView struct {
	Label     string
	Text      string
	IsCorrect bool
	IsStudent bool
}

type KeywordView struct {
	Text string
	Hit  bool
}

type QuestionItemView struct {
	ID              string
	Index           int
	Title           string
	TypeLabel       string
	FullScore       float64
	ScoreStr        string
	StudentAnswer   string
	ReferenceAnswer string
	Material        string
	Options         []QuestionOptionView
	Keywords        []KeywordView
	AutoComment     string
	TeacherComment  string
	ReadOnly        bool
}

type SubmissionViewData struct {
	Paper           model.Paper
	Submission      model.PaperSubmission
	Student         model.Student
	StudentNameInit string
	FinalScoreStr   string
	LevelClass      string
	ReadOnly        bool
	IsFinished      bool
	CanReview       bool
	CanFinal        bool
	NextStageLabel  string
	GeneralComment  string
	QuestionItems   []QuestionItemView
}

type PaperProgressView struct {
	Total          int
	Initial        int
	Review         int
	Final          int
	Finished       int
	InitialPercent float64
	ReviewPercent  float64
	FinalPercent   float64
	FinishedPercent float64
}

type PaperSubmissionItem struct {
	Submission model.PaperSubmission
	Student    model.Student
	ScoreStr   string
	IsFinished bool
}

type PaperDetailViewData struct {
	Paper       model.Paper
	Progress    PaperProgressView
	Submissions []PaperSubmissionItem
}

type ViewService struct{}

func NewViewService() *ViewService {
	return &ViewService{}
}

func (vs *ViewService) BuildSubmissionView(paper model.Paper, sub model.PaperSubmission, student model.Student) SubmissionViewData {
	data := SubmissionViewData{
		Paper:      paper,
		Submission: sub,
		Student:    student,
	}

	runes := []rune(student.Name)
	if len(runes) > 0 {
		data.StudentNameInit = string(runes[0])
	}

	var latestScore *model.ScoreRecord
	if len(sub.ScoreRecords) > 0 {
		latest := sub.ScoreRecords[len(sub.ScoreRecords)-1]
		latestScore = &latest
		data.GeneralComment = latest.Comment
	}

	if sub.CurrentStage == model.StageFinished {
		data.ReadOnly = true
		data.IsFinished = true
		data.FinalScoreStr = fmt.Sprintf("%.1f", sub.FinalScore)
	} else if latestScore != nil {
		data.FinalScoreStr = fmt.Sprintf("%.1f", latestScore.TotalScore)
	} else {
		data.FinalScoreStr = "0.0"
	}

	switch sub.ScoreLevel {
	case model.LevelExcellent:
		data.LevelClass = "excellent"
	case model.LevelGood:
		data.LevelClass = "good"
	case model.LevelPass:
		data.LevelClass = "pass"
	case model.LevelFail:
		data.LevelClass = "fail"
	default:
		data.LevelClass = "fail"
	}

	data.CanReview = sub.CurrentStage == model.StageReview
	data.CanFinal = sub.CurrentStage == model.StageFinal

	if sub.CurrentStage == model.StageReview {
		data.NextStageLabel = "复核"
	} else if sub.CurrentStage == model.StageFinal {
		data.NextStageLabel = "终评"
	}

	qsMap := make(map[string]model.QuestionScore)
	if latestScore != nil {
		for _, qs := range latestScore.QuestionScores {
			qsMap[qs.QuestionID] = qs
		}
	}

	ansMap := make(map[string]interface{})
	for _, a := range sub.Answers {
		ansMap[a.QuestionID] = a.Content
	}

	for i, q := range paper.Questions {
		item := QuestionItemView{
			ID:        q.ID,
			Index:     i + 1,
			Title:     q.Title,
			FullScore: q.FullScore,
			ReadOnly:  data.ReadOnly,
		}

		switch q.Type {
		case model.TypeSingleChoice:
			item.TypeLabel = "单选题"
		case model.TypeMultipleChoice:
			item.TypeLabel = "多选题"
		case model.TypeFillBlank:
			item.TypeLabel = "填空题"
		case model.TypeTrueFalse:
			item.TypeLabel = "判断题"
		case model.TypeShortAnswer:
			item.TypeLabel = "简答题"
		case model.TypeEssay:
			item.TypeLabel = "论述题"
		case model.TypeMaterialAnalysis:
			item.TypeLabel = "材料分析题"
		}

		if qs, ok := qsMap[q.ID]; ok {
			item.ScoreStr = fmt.Sprintf("%.1f", qs.Score)
			item.AutoComment = qs.Comment
			item.TeacherComment = qs.Comment
		} else {
			item.ScoreStr = "0.0"
		}

		if q.Type == model.TypeSingleChoice || q.Type == model.TypeMultipleChoice {
			correctAns := toStringList(q.CorrectAnswer)
			studentAns := toStringList(ansMap[q.ID])
			studentSet := make(map[string]bool)
			for _, a := range studentAns {
				studentSet[strings.TrimSpace(strings.ToUpper(a))] = true
			}
			correctSet := make(map[string]bool)
			for _, a := range correctAns {
				correctSet[strings.TrimSpace(strings.ToUpper(a))] = true
			}

			for _, opt := range q.Options {
				label := strings.SplitN(opt, ".", 2)[0]
				text := ""
				if len(strings.SplitN(opt, ".", 2)) > 1 {
					text = strings.TrimSpace(strings.SplitN(opt, ".", 2)[1])
				}
				labelUpper := strings.TrimSpace(strings.ToUpper(label))
				item.Options = append(item.Options, QuestionOptionView{
					Label:     label,
					Text:      text,
					IsCorrect: correctSet[labelUpper],
					IsStudent: studentSet[labelUpper],
				})
			}
		}

		if q.Type == model.TypeFillBlank {
			studentAns := toStringList(ansMap[q.ID])
			item.StudentAnswer = strings.Join(studentAns, " / ")
		} else if q.Type == model.TypeTrueFalse {
			if ans, ok := ansMap[q.ID].(bool); ok {
				if ans {
					item.StudentAnswer = "正确"
				} else {
					item.StudentAnswer = "错误"
				}
			}
		} else if q.Type == model.TypeShortAnswer || q.Type == model.TypeEssay || q.Type == model.TypeMaterialAnalysis {
			if ans, ok := ansMap[q.ID].(string); ok {
				item.StudentAnswer = ans
			}
			if q.ReferenceAnswer != "" {
				item.ReferenceAnswer = q.ReferenceAnswer
			}
		}

		if len(q.Keywords) > 0 {
			studentAnswerText := ""
			if ans, ok := ansMap[q.ID].(string); ok {
				studentAnswerText = strings.ToLower(ans)
			}
			for _, kw := range q.Keywords {
				hit := strings.Contains(studentAnswerText, strings.ToLower(kw))
				item.Keywords = append(item.Keywords, KeywordView{
					Text: kw,
					Hit:  hit,
				})
			}
		}

		if q.Material != "" {
			item.Material = q.Material
		}

		data.QuestionItems = append(data.QuestionItems, item)
	}

	return data
}

func (vs *ViewService) BuildPaperDetailView(paper model.Paper, submissions []model.PaperSubmission, students map[string]model.Student) PaperDetailViewData {
	data := PaperDetailViewData{
		Paper: paper,
	}

	progress := PaperProgressView{}
	progress.Total = len(submissions)

	for _, sub := range submissions {
		switch sub.CurrentStage {
		case model.StageInitial:
			progress.Initial++
		case model.StageReview:
			progress.Review++
		case model.StageFinal:
			progress.Final++
		case model.StageFinished:
			progress.Finished++
		}

		stu := students[sub.StudentID]
		item := PaperSubmissionItem{
			Submission: sub,
			Student:    stu,
			IsFinished: sub.CurrentStage == model.StageFinished,
		}

		if sub.CurrentStage == model.StageFinished {
			item.ScoreStr = fmt.Sprintf("%.1f", sub.FinalScore)
		} else if len(sub.ScoreRecords) > 0 {
			latest := sub.ScoreRecords[len(sub.ScoreRecords)-1]
			item.ScoreStr = fmt.Sprintf("%.1f", latest.TotalScore)
		} else {
			item.ScoreStr = "--"
		}

		data.Submissions = append(data.Submissions, item)
	}

	if progress.Total > 0 {
		progress.InitialPercent = float64(progress.Initial) / float64(progress.Total) * 100
		progress.ReviewPercent = float64(progress.Review) / float64(progress.Total) * 100
		progress.FinalPercent = float64(progress.Final) / float64(progress.Total) * 100
		progress.FinishedPercent = float64(progress.Finished) / float64(progress.Total) * 100
	}

	data.Progress = progress
	return data
}

func toStringList(v interface{}) []string {
	switch val := v.(type) {
	case string:
		return []string{val}
	case []string:
		return val
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}
