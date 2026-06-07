package service

import (
	"fmt"
	"math"
	"path/filepath"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"

	"grading-system/model"
)

type PaperExportData struct {
	Paper       model.Paper
	Submissions []model.PaperSubmission
	Students    map[string]model.Student
	SchoolName  string
}

type StudentExportData struct {
	Paper      model.Paper
	Submission model.PaperSubmission
	Student    model.Student
	SchoolName string
}

func ExportExcel(data PaperExportData, exportDir string) (string, error) {
	f := excelize.NewFile()
	sheetName := "成绩表"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return "", err
	}
	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1")

	_, err = f.NewSheet("成绩明细")
	if err != nil {
		return "", err
	}

	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 18, Color: "1A365D"},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	styleHeader, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4A90D9"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "B0C4DE", Style: 1},
			{Type: "right", Color: "B0C4DE", Style: 1},
			{Type: "top", Color: "B0C4DE", Style: 1},
			{Type: "bottom", Color: "B0C4DE", Style: 1},
		},
	})

	styleCell, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "D3D3D3", Style: 1},
			{Type: "right", Color: "D3D3D3", Style: 1},
			{Type: "top", Color: "D3D3D3", Style: 1},
			{Type: "bottom", Color: "D3D3D3", Style: 1},
		},
	})

	styleExcellent, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "38A169", Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"F0FFF4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleGood, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "3182CE", Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"EBF8FF"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	stylePass, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "D69E2E", Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"FFFFF0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleFail, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "E53E3E", Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"FFF5F5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	lastCol := 5 + len(data.Paper.Questions)
	lastColLetter, _ := excelize.ColumnNumberToName(lastCol)

	f.MergeCell(sheetName, "A1", lastColLetter+"1")
	f.SetCellValue(sheetName, "A1", data.SchoolName+" - "+data.Paper.Name+" 成绩单")
	f.SetCellStyle(sheetName, "A1", lastColLetter+"1", styleTitle)
	f.SetRowHeight(sheetName, 1, 36)

	f.SetCellValue(sheetName, "A2", "导出时间："+time.Now().Format("2006-01-02 15:04:05"))
	f.MergeCell(sheetName, "A2", lastColLetter+"2")
	styleSubTitle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "718096", Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	f.SetCellStyle(sheetName, "A2", lastColLetter+"2", styleSubTitle)

	headers := []string{"序号", "学号", "姓名", "班级", "总分", "段位"}
	for i, q := range data.Paper.Questions {
		typeName := ""
		switch q.Type {
		case model.TypeSingleChoice:
			typeName = "单选"
		case model.TypeMultipleChoice:
			typeName = "多选"
		case model.TypeFillBlank:
			typeName = "填空"
		case model.TypeTrueFalse:
			typeName = "判断"
		case model.TypeShortAnswer:
			typeName = "简答"
		case model.TypeEssay:
			typeName = "论述"
		case model.TypeMaterialAnalysis:
			typeName = "材料分析"
		}
		headers = append(headers, fmt.Sprintf("第%d题(%s/%.0f分)", i+1, typeName, q.FullScore))
	}

	headerRow := 3
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, headerRow)
		f.SetCellValue(sheetName, cell, h)
	}
	f.SetCellStyle(sheetName, "A3", lastColLetter+"3", styleHeader)
	f.SetRowHeight(sheetName, 3, 32)

	f.SetColWidth(sheetName, "A", "A", 6)
	f.SetColWidth(sheetName, "B", "B", 12)
	f.SetColWidth(sheetName, "C", "C", 10)
	f.SetColWidth(sheetName, "D", "D", 14)
	f.SetColWidth(sheetName, "E", "E", 10)
	f.SetColWidth(sheetName, "F", "F", 8)
	for i := 0; i < len(data.Paper.Questions); i++ {
		col, _ := excelize.ColumnNumberToName(7 + i)
		f.SetColWidth(sheetName, col, col, 14)
	}

	for idx, sub := range data.Submissions {
		row := headerRow + 1 + idx
		stu := data.Students[sub.StudentID]

		var totalScore float64
		var level model.ScoreLevel
		qsMap := make(map[string]float64)

		if len(sub.ScoreRecords) > 0 {
			latest := sub.ScoreRecords[len(sub.ScoreRecords)-1]
			totalScore = latest.TotalScore
			for _, qs := range latest.QuestionScores {
				qsMap[qs.QuestionID] = qs.Score
			}
		}

		if sub.CurrentStage == model.StageFinished {
			level = sub.ScoreLevel
		} else {
			level = ""
		}

		rowValues := []interface{}{
			idx + 1,
			stu.ID,
			stu.Name,
			stu.Class,
			fmt.Sprintf("%.1f", totalScore),
			level,
		}
		for _, q := range data.Paper.Questions {
			rowValues = append(rowValues, fmt.Sprintf("%.1f", qsMap[q.ID]))
		}

		for i, v := range rowValues {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			f.SetCellValue(sheetName, cell, v)
		}

		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("%s%d", lastColLetter, row), styleCell)
		f.SetRowHeight(sheetName, row, 22)

		levelCell := fmt.Sprintf("F%d", row)
		switch level {
		case model.LevelExcellent:
			f.SetCellStyle(sheetName, levelCell, levelCell, styleExcellent)
		case model.LevelGood:
			f.SetCellStyle(sheetName, levelCell, levelCell, styleGood)
		case model.LevelPass:
			f.SetCellStyle(sheetName, levelCell, levelCell, stylePass)
		case model.LevelFail:
			f.SetCellStyle(sheetName, levelCell, levelCell, styleFail)
		}
	}

	summaryRow := headerRow + len(data.Submissions) + 2
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", summaryRow), "统计信息")
	styleSummaryTitle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12, Color: "2D3748"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"EDF2F7"}, Pattern: 1},
	})
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", summaryRow), fmt.Sprintf("%s%d", lastColLetter, summaryRow), styleSummaryTitle)

	stats := map[string]int{
		string(model.LevelExcellent): 0,
		string(model.LevelGood):      0,
		string(model.LevelPass):      0,
		string(model.LevelFail):      0,
	}
	totalScoreAll := 0.0
	finishedCount := 0
	for _, sub := range data.Submissions {
		if sub.CurrentStage == model.StageFinished {
			stats[string(sub.ScoreLevel)]++
			totalScoreAll += sub.FinalScore
			finishedCount++
		}
	}

	avgScore := 0.0
	if finishedCount > 0 {
		avgScore = totalScoreAll / float64(finishedCount)
	}

	statsData := [][]interface{}{
		{"已定稿人数", finishedCount},
		{"平均分", fmt.Sprintf("%.1f", avgScore)},
		{"优秀人数", stats[string(model.LevelExcellent)]},
		{"良好人数", stats[string(model.LevelGood)]},
		{"合格人数", stats[string(model.LevelPass)]},
		{"不合格人数", stats[string(model.LevelFail)]},
	}

	for i, row := range statsData {
		r := summaryRow + 1 + i
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", r), row[0])
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", r), row[1])
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", r), fmt.Sprintf("B%d", r), styleCell)
	}

	fileName := fmt.Sprintf("%s_成绩表_%s.xlsx", data.Paper.ID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(exportDir, fileName)

	if err := f.SaveAs(filePath); err != nil {
		return "", err
	}

	return fileName, nil
}

func ExportStudentPDF(data StudentExportData, exportDir string) (string, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(26, 54, 93)
	pdf.CellFormat(0, 10, data.SchoolName, "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 12)
	pdf.SetTextColor(113, 128, 150)
	pdf.CellFormat(0, 8, "STUDENT TRANSCRIPT", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	pdf.SetDrawColor(74, 144, 217)
	pdf.SetLineWidth(0.8)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(6)

	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(45, 55, 72)
	pdf.CellFormat(0, 8, fmt.Sprintf("%s - %s", data.Paper.Subject, data.Paper.Name), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(74, 85, 104)
	pdf.CellFormat(40, 6, fmt.Sprintf("Student ID: %s", data.Student.ID), "", 0, "L", false, 0, "")
	pdf.CellFormat(40, 6, fmt.Sprintf("Name: %s", data.Student.Name), "", 0, "L", false, 0, "")
	pdf.CellFormat(50, 6, fmt.Sprintf("Class: %s", data.Student.Class), "", 0, "L", false, 0, "")
	pdf.CellFormat(40, 6, fmt.Sprintf("Grade: %s", data.Student.Grade), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	score := 0.0
	var latestRecord *model.ScoreRecord
	if len(data.Submission.ScoreRecords) > 0 {
		latest := data.Submission.ScoreRecords[len(data.Submission.ScoreRecords)-1]
		latestRecord = &latest
		score = latest.TotalScore
	}
	if data.Submission.CurrentStage == model.StageFinished {
		score = data.Submission.FinalScore
	}

	pdf.SetFillColor(240, 249, 255)
	pdf.Rect(15, pdf.GetY(), 85, 28, "F")
	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(74, 85, 104)
	pdf.SetXY(20, pdf.GetY()+4)
	pdf.CellFormat(0, 6, "Total Score", "", 1, "L", false, 0, "")
	pdf.SetX(20)
	pdf.SetFont("Arial", "B", 24)
	pdf.SetTextColor(74, 144, 217)
	pdf.CellFormat(0, 10, fmt.Sprintf("%.1f / %.1f", score, data.Paper.TotalScore), "", 1, "L", false, 0, "")

	level := data.Submission.ScoreLevel
	levelColor := []int{113, 128, 150}
	switch level {
	case model.LevelExcellent:
		levelColor = []int{56, 161, 105}
	case model.LevelGood:
		levelColor = []int{49, 130, 206}
	case model.LevelPass:
		levelColor = []int{214, 158, 46}
	case model.LevelFail:
		levelColor = []int{229, 62, 62}
	}
	pdf.SetXY(115, pdf.GetY()-26)
	pdf.SetFillColor(255, 255, 255)
	pdf.Rect(110, pdf.GetY()-30, 85, 28, "F")
	pdf.SetDrawColor(levelColor[0], levelColor[1], levelColor[2])
	pdf.SetLineWidth(1)
	pdf.Rect(110, pdf.GetY()-30, 85, 28, "D")
	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(levelColor[0], levelColor[1], levelColor[2])
	pdf.SetXY(115, pdf.GetY()-26)
	pdf.CellFormat(0, 6, "Grade Level", "", 1, "L", false, 0, "")
	pdf.SetX(115)
	pdf.SetFont("Arial", "B", 22)
	pdf.CellFormat(0, 10, string(level), "", 1, "L", false, 0, "")
	pdf.SetY(pdf.GetY() + 2)
	pdf.Ln(6)

	radarX := 105.0
	radarY := 110.0
	radarRadius := 38.0
	drawRadarChart(pdf, data, radarX, radarY, radarRadius)

	detailStartY := radarY + radarRadius + 15
	pdf.SetY(detailStartY)

	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(45, 55, 72)
	pdf.CellFormat(0, 8, "Score Details", "", 1, "L", false, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(74, 144, 217)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(10, 7, "No.", "1", 0, "C", true, 0, "")
	pdf.CellFormat(28, 7, "Type", "1", 0, "C", true, 0, "")
	pdf.CellFormat(82, 7, "Question", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 7, "Score", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 7, "Full Score", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 7, "Rate", "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(45, 55, 72)
	fill := false

	if latestRecord != nil {
		qsMap := make(map[string]model.QuestionScore)
		for _, qs := range latestRecord.QuestionScores {
			qsMap[qs.QuestionID] = qs
		}

		for i, q := range data.Paper.Questions {
			qs := qsMap[q.ID]
			rate := 0.0
			if q.FullScore > 0 {
				rate = qs.Score / q.FullScore * 100
			}

			typeName := ""
			switch q.Type {
			case model.TypeSingleChoice:
				typeName = "Single Choice"
			case model.TypeMultipleChoice:
				typeName = "Multiple Choice"
			case model.TypeFillBlank:
				typeName = "Fill Blank"
			case model.TypeTrueFalse:
				typeName = "True/False"
			case model.TypeShortAnswer:
				typeName = "Short Answer"
			case model.TypeEssay:
				typeName = "Essay"
			case model.TypeMaterialAnalysis:
				typeName = "Material Analysis"
			}

			if fill {
				pdf.SetFillColor(247, 250, 252)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}

			title := q.Title
			if len([]rune(title)) > 30 {
				title = string([]rune(title)[:28]) + "..."
			}

			pdf.CellFormat(10, 6.5, fmt.Sprintf("%d", i+1), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(28, 6.5, typeName, "1", 0, "C", fill, 0, "")
			pdf.CellFormat(82, 6.5, title, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(25, 6.5, fmt.Sprintf("%.1f", qs.Score), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(25, 6.5, fmt.Sprintf("%.1f", q.FullScore), "1", 0, "C", fill, 0, "")
			pdf.CellFormat(20, 6.5, fmt.Sprintf("%.0f%%", rate), "1", 1, "C", fill, 0, "")

			fill = !fill
		}
	}

	pdf.Ln(4)
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(45, 55, 72)
	pdf.CellFormat(0, 7, "Teacher's Comment", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(74, 85, 104)

	comment := ""
	if latestRecord != nil {
		comment = latestRecord.Comment
	}
	if comment == "" {
		comment = "No comment yet."
	}

	pdf.SetFillColor(247, 250, 252)
	pdf.SetDrawColor(226, 232, 240)
	x := pdf.GetX()
	y := pdf.GetY()
	pdf.Rect(x, y, 190, 20, "FD")
	pdf.SetXY(x+5, y+3)
	pdf.MultiCell(180, 5, comment, "", "L", false)

	pdf.SetY(-30)
	pdf.SetDrawColor(74, 144, 217)
	pdf.SetLineWidth(0.5)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(2)
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(160, 174, 192)
	pdf.CellFormat(0, 5, fmt.Sprintf("Generated on %s | Grading System v1.0", time.Now().Format("2006-01-02 15:04:05")), "", 0, "C", false, 0, "")

	fileName := fmt.Sprintf("%s_%s_%s_成绩单.pdf", data.Paper.ID, data.Student.ID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(exportDir, fileName)

	if err := pdf.OutputFileAndClose(filePath); err != nil {
		return "", err
	}

	return fileName, nil
}

func drawRadarChart(pdf *gofpdf.Fpdf, data StudentExportData, cx, cy, r float64) {
	qsMap := make(map[string]model.QuestionScore)
	if len(data.Submission.ScoreRecords) > 0 {
		latest := data.Submission.ScoreRecords[len(data.Submission.ScoreRecords)-1]
		for _, qs := range latest.QuestionScores {
			qsMap[qs.QuestionID] = qs
		}
	}

	type radarDim struct {
		score float64
		full  float64
		label string
	}

	objective := radarDim{0, 0, "Objective"}
	subjective := radarDim{0, 0, "Subjective"}
	shortAnswer := radarDim{0, 0, "Short Ans"}
	essay := radarDim{0, 0, "Essay"}
	knowledge := radarDim{0, 0, "Knowledge"}
	application := radarDim{0, 0, "Application"}

	for _, q := range data.Paper.Questions {
		qs := qsMap[q.ID]
		switch q.Type {
		case model.TypeSingleChoice, model.TypeMultipleChoice, model.TypeFillBlank, model.TypeTrueFalse:
			objective.score += qs.Score
			objective.full += q.FullScore
		default:
			subjective.score += qs.Score
			subjective.full += q.FullScore
		}
		switch q.Type {
		case model.TypeShortAnswer:
			shortAnswer.score += qs.Score
			shortAnswer.full += q.FullScore
		case model.TypeEssay, model.TypeMaterialAnalysis:
			essay.score += qs.Score
			essay.full += q.FullScore
		}
	}

	if objective.full > 0 {
		knowledge.score = objective.score * 0.7
		knowledge.full = objective.full * 0.7
		application.score = subjective.score * 0.6
		application.full = subjective.full * 0.6
	}
	if knowledge.full == 0 {
		knowledge.full = 1
	}
	if application.full == 0 {
		application.full = 1
	}
	if shortAnswer.full == 0 {
		shortAnswer.full = 1
	}
	if essay.full == 0 {
		essay.full = 1
	}
	if objective.full == 0 {
		objective.full = 1
	}
	if subjective.full == 0 {
		subjective.full = 1
	}

	dims := []radarDim{objective, subjective, shortAnswer, essay, knowledge, application}
	angles := []float64{-90, -30, 30, 90, 150, 210}

	pdf.SetDrawColor(200, 200, 200)
	pdf.SetLineWidth(0.3)

	for level := 1; level <= 5; level++ {
		rr := r * float64(level) / 5
		for i := 0; i < len(dims); i++ {
			angle1 := angles[i] * math.Pi / 180
			angle2 := angles[(i+1)%len(dims)] * math.Pi / 180
			x1 := cx + rr*math.Cos(angle1)
			y1 := cy + rr*math.Sin(angle1)
			x2 := cx + rr*math.Cos(angle2)
			y2 := cy + rr*math.Sin(angle2)
			pdf.Line(x1, y1, x2, y2)
		}
	}

	pdf.SetDrawColor(150, 150, 150)
	for i, dim := range dims {
		_ = dim
		angle := angles[i] * math.Pi / 180
		x := cx + r*math.Cos(angle)
		y := cy + r*math.Sin(angle)
		pdf.Line(cx, cy, x, y)
	}

	pdf.SetFillColor(74, 144, 217)
	pdf.SetDrawColor(30, 64, 175)
	pdf.SetLineWidth(1.0)

	for i, d := range dims {
		ratio := d.score / d.full
		if ratio > 1 {
			ratio = 1
		}
		angle := angles[i] * math.Pi / 180
		x := cx + r*ratio*math.Cos(angle)
		y := cy + r*ratio*math.Sin(angle)
		if i == 0 {
			pdf.MoveTo(x, y)
		} else {
			pdf.LineTo(x, y)
		}
	}
	pdf.ClosePath()
	pdf.SetAlpha(0.3, "Normal")
	pdf.DrawPath("FD")
	pdf.SetAlpha(1, "Normal")

	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(74, 85, 104)
	for i, d := range dims {
		angle := angles[i] * math.Pi / 180
		label := d.label
		labelR := r + 4
		x := cx + labelR*math.Cos(angle)
		y := cy + labelR*math.Sin(angle)
		pdf.SetXY(x-15, y-3)
		pdf.CellFormat(30, 6, label, "", 0, "C", false, 0, "")
	}

	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(45, 55, 72)
	pdf.SetXY(cx-30, cy-r-10)
	pdf.CellFormat(60, 6, "Performance Radar", "", 0, "C", false, 0, "")
}
