package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"grading-system/model"
	"grading-system/service"
)

type Handler struct {
	submissionService *service.SubmissionService
	exportService     *service.ExportService
	viewService       *service.ViewService
}

func NewHandler(ss *service.SubmissionService, es *service.ExportService, vs *service.ViewService) *Handler {
	return &Handler{
		submissionService: ss,
		exportService:     es,
		viewService:       vs,
	}
}

func (h *Handler) Index(c *gin.Context) {
	papers := h.submissionService.ListAllPapers()
	c.HTML(http.StatusOK, "index.html", gin.H{
		"papers": papers,
	})
}

func (h *Handler) PaperDetail(c *gin.Context) {
	paperID := c.Param("id")
	repo := h.submissionService.GetRepo()
	paper, ok := repo.GetPaper(paperID)
	if !ok {
		c.String(http.StatusNotFound, "试卷不存在")
		return
	}

	submissions := repo.ListSubmissionsByPaper(paperID)
	students := make(map[string]model.Student)
	for _, s := range repo.ListStudents() {
		students[s.ID] = s
	}

	viewData := h.viewService.BuildPaperDetailView(paper, submissions, students)
	c.HTML(http.StatusOK, "paper_detail.html", viewData)
}

func (h *Handler) SubmissionDetail(c *gin.Context) {
	subID := c.Param("id")
	repo := h.submissionService.GetRepo()
	sub, ok := repo.GetSubmission(subID)
	if !ok {
		c.String(http.StatusNotFound, "答卷不存在")
		return
	}

	paper, ok := repo.GetPaper(sub.PaperID)
	if !ok {
		c.String(http.StatusNotFound, "试卷不存在")
		return
	}

	student, ok := repo.GetStudent(sub.StudentID)
	if !ok {
		c.String(http.StatusNotFound, "学生不存在")
		return
	}

	viewData := h.viewService.BuildSubmissionView(paper, sub, student)
	c.HTML(http.StatusOK, "submission_detail.html", viewData)
}

func (h *Handler) JumpToSubmission(c *gin.Context) {
	paperID := c.Query("paper_id")
	studentID := c.Query("student_id")
	if paperID == "" || studentID == "" {
		c.String(http.StatusBadRequest, "参数缺失")
		return
	}

	repo := h.submissionService.GetRepo()
	sub, ok := repo.GetSubmissionByPaperAndStudent(paperID, studentID)
	if !ok {
		c.String(http.StatusNotFound, "未找到对应答卷")
		return
	}

	c.Redirect(http.StatusFound, "/submission/"+sub.ID)
}

func (h *Handler) ListPapersAPI(c *gin.Context) {
	papers := h.submissionService.ListAllPapers()
	c.JSON(http.StatusOK, gin.H{"papers": papers})
}

func (h *Handler) GetPaperAPI(c *gin.Context) {
	paperID := c.Param("id")
	data, err := h.submissionService.GetPaperWithProgress(paperID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) GetSubmissionAPI(c *gin.Context) {
	subID := c.Param("id")
	data, err := h.submissionService.GetSubmissionDetail(subID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

type ScoreRequest struct {
	QuestionScores []model.QuestionScore `json:"question_scores"`
	Comment        string                `json:"comment"`
	Teacher        string                `json:"teacher"`
}

func (h *Handler) SubmitReview(c *gin.Context) {
	subID := c.Param("id")
	stage := model.GradingStage(c.Param("stage"))

	var req ScoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Teacher == "" {
		req.Teacher = "default_teacher"
	}

	err := h.submissionService.SubmitReview(subID, req.Teacher, req.QuestionScores, req.Comment, stage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) LockSubmission(c *gin.Context) {
	subID := c.Param("id")
	teacher := c.DefaultQuery("teacher", "default_teacher")

	ok := h.submissionService.LockSubmission(subID, teacher)
	if !ok {
		c.JSON(http.StatusConflict, gin.H{"error": "无法锁定答卷"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "locked"})
}

func (h *Handler) UnlockSubmission(c *gin.Context) {
	subID := c.Param("id")
	teacher := c.DefaultQuery("teacher", "default_teacher")

	ok := h.submissionService.UnlockSubmission(subID, teacher)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法解锁答卷"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "unlocked"})
}

type ExportRequest struct {
	PaperID   string `json:"paper_id"`
	Type      string `json:"type"`
	StudentID string `json:"student_id"`
}

func (h *Handler) CreateExport(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.PaperID = c.Query("paper_id")
		req.Type = c.DefaultQuery("type", "excel")
		req.StudentID = c.Query("student_id")
	}

	taskID := h.exportService.CreateExportTask(req.PaperID, req.Type, req.StudentID)
	c.JSON(http.StatusOK, gin.H{"task_id": taskID, "status": "pending"})
}

func (h *Handler) GetExportStatus(c *gin.Context) {
	taskID := c.Param("id")
	task, ok := h.exportService.GetExportTask(taskID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) DownloadExport(c *gin.Context) {
	taskID := c.Param("id")
	task, ok := h.exportService.GetExportTask(taskID)
	if !ok {
		c.String(http.StatusNotFound, "任务不存在")
		return
	}
	if task.Status != "completed" {
		c.String(http.StatusBadRequest, "导出任务尚未完成")
		return
	}

	filePath := h.exportService.GetExportFilePath(task.FileName)
	c.FileAttachment(filePath, task.FileName)
}

func (h *Handler) RunInitialGrading(c *gin.Context) {
	subID := c.Param("id")
	err := h.submissionService.RunInitialGrading(subID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
