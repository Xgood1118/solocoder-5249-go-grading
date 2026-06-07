package algorithm

import (
	"strings"
)

func LCSLength(a, b string) int {
	runesA := []rune(a)
	runesB := []rune(b)
	m := len(runesA)
	n := len(runesB)

	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if runesA[i-1] == runesB[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] > dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	return dp[m][n]
}

func SimilarityRatio(studentAnswer, referenceAnswer string) float64 {
	if len(referenceAnswer) == 0 {
		return 0
	}
	lcs := LCSLength(studentAnswer, referenceAnswer)
	return float64(lcs) / float64(len([]rune(referenceAnswer)))
}

func KeywordCoverage(studentAnswer string, keywords []string) float64 {
	if len(keywords) == 0 {
		return 0
	}

	lowerAnswer := strings.ToLower(studentAnswer)
	count := 0

	for _, kw := range keywords {
		if strings.Contains(lowerAnswer, strings.ToLower(kw)) {
			count++
		}
	}

	return float64(count) / float64(len(keywords))
}

func GradeShortAnswer(studentAnswer string, keywords []string, fullScore float64) float64 {
	if len(keywords) == 0 {
		return 0
	}
	coverage := KeywordCoverage(studentAnswer, keywords)
	return fullScore * coverage
}

func GradeEssayOrMaterial(studentAnswer string, keywords []string, referenceAnswer string, fullScore float64, minLength int) float64 {
	keywordScore := KeywordCoverage(studentAnswer, keywords) * 0.5

	refLen := len([]rune(referenceAnswer))
	ansLen := len([]rune(studentAnswer))
	var lengthScore float64
	if refLen > 0 && ansLen >= refLen {
		lengthScore = 1.0
	} else if refLen > 0 {
		lengthScore = float64(ansLen) / float64(refLen)
		if lengthScore > 1.0 {
			lengthScore = 1.0
		}
	}
	lengthScore *= 0.2

	similarity := SimilarityRatio(studentAnswer, referenceAnswer) * 0.3

	totalRatio := keywordScore + lengthScore + similarity
	if totalRatio > 1.0 {
		totalRatio = 1.0
	}

	score := fullScore * totalRatio

	if minLength > 0 && ansLen < minLength {
		score = score * 0.7
	}

	if score < 0 {
		score = 0
	}
	if score > fullScore {
		score = fullScore
	}

	return score
}
