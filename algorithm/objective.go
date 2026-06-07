package algorithm

import (
	"strings"
)

func GradeSingleChoice(correctAnswer string, studentAnswer string, fullScore float64) float64 {
	if strings.TrimSpace(strings.ToLower(correctAnswer)) == strings.TrimSpace(strings.ToLower(studentAnswer)) {
		return fullScore
	}
	return 0
}

func GradeMultipleChoice(correctAnswers []string, studentAnswers []string, fullScore float64) float64 {
	if len(correctAnswers) == 0 {
		return 0
	}

	correctSet := make(map[string]bool)
	for _, a := range correctAnswers {
		correctSet[strings.TrimSpace(strings.ToLower(a))] = true
	}

	studentSet := make(map[string]bool)
	for _, a := range studentAnswers {
		studentSet[strings.TrimSpace(strings.ToLower(a))] = true
	}

	for ans := range studentSet {
		if !correctSet[ans] {
			return 0
		}
	}

	correctCount := 0
	for ans := range studentSet {
		if correctSet[ans] {
			correctCount++
		}
	}

	return fullScore * float64(correctCount) / float64(len(correctAnswers))
}

func GradeFillBlank(correctAnswers []string, studentAnswers []string, fullScore float64) float64 {
	if len(correctAnswers) == 0 {
		return 0
	}

	scorePerBlank := fullScore / float64(len(correctAnswers))
	total := 0.0

	for i := 0; i < len(correctAnswers) && i < len(studentAnswers); i++ {
		if strings.TrimSpace(strings.ToLower(correctAnswers[i])) == strings.TrimSpace(strings.ToLower(studentAnswers[i])) {
			total += scorePerBlank
		}
	}

	return total
}

func GradeTrueFalse(correctAnswer bool, studentAnswer bool, fullScore float64) float64 {
	if correctAnswer == studentAnswer {
		return fullScore
	}
	return 0
}
