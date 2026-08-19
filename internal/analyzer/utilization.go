package analyzer

import "github.com/rebash-rebash/g9s/internal/model"

// VMUsageStatus describes operational and CPU-utilization state without making
// a cost decision. Cost recommendations are intentionally handled separately.
func VMUsageStatus(vm model.VM, utilization model.Utilization) string {
	if vm.Status != "RUNNING" {
		return "NOT RUNNING"
	}
	if utilization.P95 == nil {
		return "UNKNOWN"
	}
	switch {
	case *utilization.P95 < 0.10:
		return "VERY LOW CPU"
	case *utilization.P95 < 0.30:
		return "UNDERUTILIZED"
	case *utilization.P95 < 0.70:
		return "NORMAL"
	default:
		return "HIGH CPU"
	}
}

// Recommendation returns a conservative next step based only on CPU data.
// It does not claim savings until billing and machine pricing are integrated.
func Recommendation(vm model.VM, utilization model.Utilization) string {
	if vm.Status != "RUNNING" {
		return "No utilization recommendation while VM is not running."
	}
	if utilization.P95 == nil {
		return "Collect more utilization data before recommending changes."
	}
	switch {
	case *utilization.P95 < 0.10:
		return "Review whether this VM is required; CPU usage is extremely low."
	case *utilization.P95 < 0.30:
		return "Review machine sizing; CPU usage is consistently low."
	case *utilization.P95 >= 0.70:
		return "Review capacity and workload pressure; CPU usage is high."
	default:
		return "No CPU-based action recommended."
	}
}
