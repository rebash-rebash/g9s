package analyzer

import (
	"fmt"

	"github.com/rebash-rebash/g9s/internal/model"
)

// UsageClass is the high-level utilization classification used by G9S.
type UsageClass string

const (
	UsageNotRunning UsageClass = "NOT RUNNING"
	UsageIdle       UsageClass = "IDLE"
	UsageUnderused  UsageClass = "UNDERUSED"
	UsageActive     UsageClass = "ACTIVE"
)

// Thresholds are intentionally conservative. G9S should identify candidates
// for review, not make destructive decisions based on a single metric.
const (
	idleCPUP95Ratio        = 0.01
	underusedCPUP95Ratio   = 0.30
	idleIOBytesPerSecond   = 1024.0
	underusedIOBytesPerSec = 102400.0
)

// VMUsageClass combines CPU, network and disk signals for a VM.
// A VM is only classified as IDLE when all observed activity signals are very
// low. Low CPU alone is never enough to call a VM idle.
func VMUsageClass(vm model.VM, cpu model.Utilization, io model.IOStats) UsageClass {
	if vm.Status != "RUNNING" {
		return UsageNotRunning
	}
	if cpu.P95 == nil {
		return UsageActive
	}

	networkIdle := below(io.NetworkInP95, idleIOBytesPerSecond) &&
		below(io.NetworkOutP95, idleIOBytesPerSecond)
	diskIdle := below(io.DiskReadP95, idleIOBytesPerSecond) &&
		below(io.DiskWriteP95, idleIOBytesPerSecond)
	if *cpu.P95 < idleCPUP95Ratio && networkIdle && diskIdle {
		return UsageIdle
	}

	networkLow := below(io.NetworkInP95, underusedIOBytesPerSec) &&
		below(io.NetworkOutP95, underusedIOBytesPerSec)
	diskLow := below(io.DiskReadP95, underusedIOBytesPerSec) &&
		below(io.DiskWriteP95, underusedIOBytesPerSec)
	if *cpu.P95 < underusedCPUP95Ratio && networkLow && diskLow {
		return UsageUnderused
	}

	return UsageActive
}

// VMUsageStatus accepts optional I/O data. With I/O available, the full
// classifier is used; without it, G9S deliberately avoids claiming IDLE.
func VMUsageStatus(vm model.VM, cpu model.Utilization, io ...model.IOStats) string {
	if vm.Status != "RUNNING" {
		return string(UsageNotRunning)
	}
	if cpu.P95 == nil {
		return "UNKNOWN"
	}
	if len(io) > 0 {
		return string(VMUsageClass(vm, cpu, io[0]))
	}
	if *cpu.P95 < idleCPUP95Ratio {
		return "VERY LOW CPU — review"
	}
	if *cpu.P95 < underusedCPUP95Ratio {
		return string(UsageUnderused) + " candidate"
	}
	return string(UsageActive)
}

// Recommendation accepts optional I/O data and remains conservative: it never
// recommends deletion or shutdown solely from utilization data.
func Recommendation(vm model.VM, cpu model.Utilization, io ...model.IOStats) string {
	if vm.Status != "RUNNING" {
		return "No runtime recommendation; VM is not running."
	}
	if cpu.P95 == nil {
		return "Collect more utilization data before recommending a change."
	}
	if len(io) > 0 {
		switch VMUsageClass(vm, cpu, io[0]) {
		case UsageIdle:
			return "Review whether this VM is required; CPU, network, and disk activity are all very low."
		case UsageUnderused:
			return fmt.Sprintf("Review machine sizing; CPU P95 is %.1f%% and I/O activity is low.", *cpu.P95*100)
		default:
			return "No utilization-based action recommended."
		}
	}
	if *cpu.P95 < idleCPUP95Ratio {
		return "Review whether this VM is required; CPU usage is extremely low."
	}
	if *cpu.P95 < underusedCPUP95Ratio {
		return fmt.Sprintf("Review machine sizing; CPU P95 is %.1f%%.", *cpu.P95*100)
	}
	return "No utilization-based action recommended."
}

func below(value *float64, threshold float64) bool {
	return value != nil && *value < threshold
}
