package algorithm

import "grading-system/model"

func CalculateScoreLevel(score float64, fullScore float64) model.ScoreLevel {
	ratio := score / fullScore * 100
	switch {
	case ratio >= 90:
		return model.LevelExcellent
	case ratio >= 75:
		return model.LevelGood
	case ratio >= 60:
		return model.LevelPass
	default:
		return model.LevelFail
	}
}
