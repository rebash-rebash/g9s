package analyzer

import "github.com/rebash-rebash/g9s/internal/model"

func ClassifyUtilization(m model.Metric) string {
	if m.Value < 10 {
		return "very-low"
	}
	if m.Value < 30 {
		return "low"
	}
	if m.Value < 70 {
		return "normal"
	}
	if m.Value < 90 {
		return "high"
	}
	return "very-high"
}

func AnalyzeWaste(f model.Finding) model.Finding { return f }

func EstimateSaving(current, recommended float64) float64 {
	if current <= recommended {
		return 0
	}
	return current - recommended
}
