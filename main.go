package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"grading-system/handler"
	"grading-system/repository"
	"grading-system/service"
)

func main() {
	dataDir := filepath.Join(".", "data")
	exportDir := filepath.Join(".", "data", "exports")

	repo := repository.GetRepository(dataDir)
	if err := repo.LoadData(); err != nil {
		log.Printf("Warning: 加载数据失败: %v", err)
	}

	submissionService := service.NewSubmissionService(repo)
	exportService := service.NewExportService(repo, exportDir)
	viewService := service.NewViewService()

	submissions := repo.ListSubmissions()
	for _, sub := range submissions {
		if sub.CurrentStage == "initial" && len(sub.ScoreRecords) == 0 {
			err := submissionService.RunInitialGrading(sub.ID)
			if err != nil {
				log.Printf("初评失败 %s: %v", sub.ID, err)
			} else {
				log.Printf("初评完成: %s", sub.ID)
			}
		}
	}

	h := handler.NewHandler(submissionService, exportService, viewService)

	r := gin.Default()

	r.LoadHTMLGlob("templates/*.html")
	r.Static("/static", "./static")

	r.GET("/", h.Index)
	r.GET("/paper/:id", h.PaperDetail)
	r.GET("/submission/:id", h.SubmissionDetail)
	r.GET("/jump", h.JumpToSubmission)

	api := r.Group("/api")
	{
		api.GET("/papers", h.ListPapersAPI)
		api.GET("/papers/:id", h.GetPaperAPI)
		api.GET("/submissions/:id", h.GetSubmissionAPI)
		api.POST("/submissions/:id/grade/initial", h.RunInitialGrading)
		api.POST("/submissions/:id/grade/:stage", h.SubmitReview)
		api.POST("/submissions/:id/lock", h.LockSubmission)
		api.POST("/submissions/:id/unlock", h.UnlockSubmission)
		api.POST("/export", h.CreateExport)
		api.GET("/export/:id", h.GetExportStatus)
		api.GET("/export/:id/download", h.DownloadExport)
	}

	port := ":8080"
	fmt.Printf("服务器启动中... http://localhost%s\n", port)
	fmt.Printf("首页: http://localhost%s/\n", port)
	if err := r.Run(port); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
