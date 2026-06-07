package model

import "time"

type QuestionType string

const (
	TypeSingleChoice   QuestionType = "single_choice"
	TypeMultipleChoice QuestionType = "multiple_choice"
	TypeFillBlank      QuestionType = "fill_blank"
	TypeTrueFalse      QuestionType = "true_false"
	TypeShortAnswer    QuestionType = "short_answer"
	TypeEssay          QuestionType = "essay"
	TypeMaterialAnalysis QuestionType = "material_analysis"
)

type GradingStage string

const (
	StageInitial  GradingStage = "initial"
	StageReview   GradingStage = "review"
	StageFinal    GradingStage = "final"
	StageFinished GradingStage = "finished"
)

type ScoreLevel string

const (
	LevelExcellent ScoreLevel = "优秀"
	LevelGood      ScoreLevel = "良好"
	LevelPass      ScoreLevel = "合格"
	LevelFail      ScoreLevel = "不合格"
)

type Question struct {
	ID              string      `json:"id"`
	Type            QuestionType `json:"type"`
	Title           string      `json:"title"`
	FullScore       float64     `json:"full_score"`
	Options         []string    `json:"options,omitempty"`
	CorrectAnswer   interface{} `json:"correct_answer"`
	Keywords        []string    `json:"keywords,omitempty"`
	ReferenceAnswer string      `json:"reference_answer,omitempty"`
	Material        string      `json:"material,omitempty"`
	MinLength       int         `json:"min_length,omitempty"`
}

type Paper struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Subject     string     `json:"subject"`
	TotalScore  float64    `json:"total_score"`
	Questions   []Question `json:"questions"`
}

type Student struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Class    string `json:"class"`
	Grade    string `json:"grade"`
}

type Answer struct {
	QuestionID string      `json:"question_id"`
	Content    interface{} `json:"content"`
}

type QuestionScore struct {
	QuestionID string  `json:"question_id"`
	Score      float64 `json:"score"`
	FullScore  float64 `json:"full_score"`
	Comment    string  `json:"comment,omitempty"`
}

type ScoreRecord struct {
	ID         string         `json:"id"`
	Stage      GradingStage   `json:"stage"`
	Grader     string         `json:"grader"`
	TotalScore float64        `json:"total_score"`
	QuestionScores []QuestionScore `json:"question_scores"`
	Comment    string         `json:"comment"`
	Timestamp  time.Time      `json:"timestamp"`
}

type PaperSubmission struct {
	ID            string        `json:"id"`
	PaperID       string        `json:"paper_id"`
	StudentID     string        `json:"student_id"`
	Answers       []Answer      `json:"answers"`
	CurrentStage  GradingStage  `json:"current_stage"`
	ScoreRecords  []ScoreRecord `json:"score_records"`
	ScoreLevel    ScoreLevel    `json:"score_level,omitempty"`
	FinalScore    float64       `json:"final_score,omitempty"`
	LockedBy      string        `json:"locked_by,omitempty"`
	LockedAt      *time.Time    `json:"locked_at,omitempty"`
	SubmittedAt   time.Time     `json:"submitted_at"`
}

type ExportTask struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Status    string      `json:"status"`
	FileName  string      `json:"file_name,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

type GradeInfo struct {
	Subject  string
	Score    float64
	FullScore float64
	Level    ScoreLevel
}
