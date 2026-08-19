package cost

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/rebash-rebash/g9s/internal/model"
)

const hoursPerMonth = 730.0

// VMEstimate is an estimate only. It intentionally excludes discounts,
// sustained-use/committed-use discounts, guest licenses, disks, and network
// egress until those prices are sourced separately.
type VMEstimate struct {
	MachineType string
	VCPU        int
	MemoryGB    float64
	HourlyUSD   float64
	DailyUSD    float64
	MonthlyUSD  float64
	Source      string
}

// Catalog contains baseline on-demand Compute Engine rates in USD/hour.
// Keep this small and explicit until G9S integrates Cloud Billing Catalog API.
type Catalog struct {
	VCPUHourUSD   float64
	MemoryGBHourUSD float64
	Source        string
}

// NewUSCentral1Catalog returns the baseline catalog used by the first cost
// estimator. Rates are deliberately injected so the estimator can later use
// region-specific or live Billing Catalog prices without changing its API.
func NewUSCentral1Catalog() Catalog {
	return Catalog{
		// Baseline N1 US pricing; replace with live catalog data before treating
		// the estimate as an invoice-grade number.
		VCPUHourUSD:     0.031611,
		MemoryGBHourUSD: 0.004237,
		Source:          "baseline on-demand catalog; not billing data",
	}
}

func (c Catalog) EstimateVM(vm model.VM) VMEstimate {
	vCPU, memoryGB := parseMachineType(vm.MachineType)
	hourly := float64(vCPU)*c.VCPUHourUSD + memoryGB*c.MemoryGBHourUSD
	return VMEstimate{
		MachineType: vm.MachineType,
		VCPU:        vCPU,
		MemoryGB:    memoryGB,
		HourlyUSD:   hourly,
		DailyUSD:    hourly * 24,
		MonthlyUSD:  hourly * hoursPerMonth,
		Source:      c.Source,
	}
}

var machinePattern = regexp.MustCompile(`^(?:n1|e2|n2|n2d|t2d|c2|c2d|c3|c3d|m1|m2|m3|m4|t2a|a2|g2|h3|h4)-(.+)$`)

func parseMachineType(machineType string) (int, float64) {
	machineType = strings.TrimSpace(machineType)
	if machineType == "" {
		return 0, 0
	}
	parts := strings.Split(machineType, "-")
	if len(parts) < 2 {
		return 0, 0
	}
	shape := parts[len(parts)-1]
	if strings.Contains(shape, "custom") {
		return 0, 0
	}
	vCPU, err := strconv.Atoi(shape)
	if err != nil {
		return 0, 0
	}

	// N1 predefined memory ratios are the initial supported family. Other
	// families are returned with an unknown memory value rather than guessed.
	if strings.HasPrefix(machineType, "n1-") {
		memoryByCPU := map[int]float64{
			1: 3.75, 2: 7.5, 4: 15, 8: 30, 16: 60, 32: 120, 64: 240, 96: 360,
		}
		return vCPU, memoryByCPU[vCPU]
	}
	return vCPU, 0
}

func FormatEstimate(e VMEstimate) string {
	if e.HourlyUSD == 0 {
		return "Pricing unavailable"
	}
	return fmt.Sprintf("$%.4f/hr  $%.2f/day  $%.2f/month", e.HourlyUSD, e.DailyUSD, e.MonthlyUSD)
}
