package service

import (
	"fmt"

	"grading-system/algorithm"
	"grading-system/model"
)

func GradeQuestion(q model.Question, answerContent interface{}) (float64, string) {
	switch q.Type {
	case model.TypeSingleChoice:
		ans, ok := answerContent.(string)
		if !ok {
			return 0, "答案格式错误"
		}
		correct, _ := q.CorrectAnswer.(string)
		score := algorithm.GradeSingleChoice(correct, ans, q.FullScore)
		return score, ""

	case model.TypeMultipleChoice:
		ansList, ok := toStringSlice(answerContent)
		if !ok {
			return 0, "答案格式错误"
		}
		correctList, _ := toStringSlice(q.CorrectAnswer)
		score := algorithm.GradeMultipleChoice(correctList, ansList, q.FullScore)
		return score, ""

	case model.TypeFillBlank:
		ansList, ok := toStringSlice(answerContent)
		if !ok {
			return 0, "答案格式错误"
		}
		correctList, _ := toStringSlice(q.CorrectAnswer)
		score := algorithm.GradeFillBlank(correctList, ansList, q.FullScore)
		return score, ""

	case model.TypeTrueFalse:
		ansBool, ok := answerContent.(bool)
		if !ok {
			return 0, "答案格式错误"
		}
		correctBool, _ := q.CorrectAnswer.(bool)
		score := algorithm.GradeTrueFalse(correctBool, ansBool, q.FullScore)
		return score, ""

	case model.TypeShortAnswer:
		ans, ok := answerContent.(string)
		if !ok {
			return 0, "答案格式错误"
		}
		score := algorithm.GradeShortAnswer(ans, q.Keywords, q.FullScore)
		comment := fmt.Sprintf("关键词覆盖率: %.0f%%", algorithm.KeywordCoverage(ans, q.Keywords)*100)
		return score, comment

	case model.TypeEssay, model.TypeMaterialAnalysis:
		ans, ok := answerContent.(string)
		if !ok {
			return 0, "答案格式错误"
		}
		minLen := q.MinLength
		if minLen == 0 {
			minLen = 50
		}
		score := algorithm.GradeEssayOrMaterial(ans, q.Keywords, q.ReferenceAnswer, q.FullScore, minLen)
		keywordCov := algorithm.KeywordCoverage(ans, q.Keywords)
		sim := algorithm.SimilarityRatio(ans, q.ReferenceAnswer)
		comment := fmt.Sprintf("关键词覆盖率: %.0f%%, 相似度: %.0f%%, 答案长度: %d字",
			keywordCov*100, sim*100, len([]rune(ans)))
		return score, comment

	default:
		return 0, "未知题型"
	}
}

func GradePaper(paper model.Paper, answers []model.Answer) ([]model.QuestionScore, float64) {
	answerMap := make(map[string]interface{})
	for _, a := range answers {
		answerMap[a.QuestionID] = a.Content
	}

	var questionScores []model.QuestionScore
	total := 0.0

	for _, q := range paper.Questions {
		ansContent, ok := answerMap[q.ID]
		var score float64
		var comment string
		if !ok {
			score = 0
			comment = "未作答"
		} else {
			score, comment = GradeQuestion(q, ansContent)
		}
		questionScores = append(questionScores, model.QuestionScore{
			QuestionID: q.ID,
			Score:      score,
			FullScore:  q.FullScore,
			Comment:    comment,
		})
		total += score
	}

	return questionScores, total
}

func toStringSlice(v interface{}) ([]string, bool) {
	switch val := v.(type) {
	case []string:
		return val, true
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			} else {
				return nil, false
			}
		}
		return result, true
	default:
		return nil, false
	}
}
