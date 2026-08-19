package cost

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/rebash-rebash/g9s/internal/model"
)

const hoursPerMonth = 730.0

type VMEstimate struct {
	MachineType string
	VCPU        int
	MemoryGB    float64
	HourlyUSD   float64
	DailyUSD    float64
	MonthlyUSD  float64
	Source      string
	Available   bool
}

type Catalog struct {
	VCPUHourUSD     float64
	MemoryGBHourUSD float64
	Source          string
}

func NewUSCentral1Catalog() Catalog {
	return Catalog{
		VCPUHourUSD:     0.031611,
		MemoryGBHourUSD: 0.004237,
		Source:          "Google Cloud us-central1 on-demand baseline; not billing data",
	}
}

func (c Catalog) EstimateVM(vm model.VM) VMEstimate {
	vCPU, memoryGB, hourly, supported := machineTypePrice(vm.MachineType)
	if !supported {
		return VMEstimate{MachineType: vm.MachineType, VCPU: vCPU, MemoryGB: memoryGB, Source: c.Source}
	}
	if vm.Status != "RUNNING" {
		return VMEstimate{MachineType: vm.MachineType, VCPU: vCPU, MemoryGB: memoryGB, Source: c.Source, Available: true}
	}
	return VMEstimate{
		MachineType: vm.MachineType,
		VCPU:        vCPU,
		MemoryGB:    memoryGB,
		HourlyUSD:   hourly,
		DailyUSD:    hourly * 24,
		MonthlyUSD:  hourly * hoursPerMonth,
		Source:      c.Source,
		Available:   true,
	}
}

var machinePattern = regexp.MustCompile(`^(n1|n2|n2d)-(standard|highmem|highcpu)-([0-9]+)$`)

// machineTypePrice covers the machine families currently present in the
// project. Standard-family prices scale linearly from the us-central1
// on-demand baseline for the family. It is still an estimate, not billing.
func machineTypePrice(machineType string) (int, float64, float64, bool) {
	machineType = strings.TrimSpace(machineType)
	match := machinePattern.FindStringSubmatch(machineType)
	if len(match) != 4 {
		return 0, 0, 0, false
	}
	family, shape, size := match[1], match[2], match[3]
	vCPU, err := strconv.Atoi(size)
	if err != nil || vCPU <= 0 {
		return 0, 0, 0, false
	}
	memoryGB, ok := machineMemory(family, shape, vCPU)
	if !ok {
		return 0, 0, 0, false
	}

	var baseVCPU int
	var baseHourly float64
	switch {
	case family == "n1" && shape == "standard":
		baseVCPU, baseHourly = 2, 0.0950
	case family == "n2" && shape == "standard":
		baseVCPU, baseHourly = 4, 0.1942
	case family == "n2d" && shape == "standard":
		baseVCPU, baseHourly = 4, 0.1680
	default:
		return 0, 0, 0, false
	}

	hourly := baseHourly * float64(vCPU) / float64(baseVCPU)
	return vCPU, memoryGB, hourly, true
}

func machineMemory(family, shape string, vCPU int) (float64, bool) {
	memoryByCPU := map[string]map[int]float64{
		"n1-standard": {1: 3.75, 2: 7.5, 4: 15, 8: 30, 16: 60, 32: 120, 64: 240, 96: 360},
		"n2-standard": {2: 8, 4: 16, 8: 32, 16: 64, 32: 128, 48: 192, 64: 256, 80: 320, 96: 384, 128: 512},
		"n2d-standard": {2: 8, 4: 16, 8: 32, 16: 64, 32: 128, 48: 192, 64: 256, 80: 320, 96: 384, 128: 512, 224: 896},
	}
	values, ok := memoryByCPU[family+"-"+shape]
	if !ok {
		return 0, false
	}
	memory, ok := values[vCPU]
	return memory, ok
}

func FormatEstimate(e VMEstimate) string {
	if !e.Available {
		return "Pricing unavailable for this machine type"
	}
	if e.HourlyUSD == 0 {
		return "$0 compute (VM not running)"
	}
	return fmt.Sprintf("$%.4f/hr  $%.2f/day  $%.2f/month", e.HourlyUSD, e.DailyUSD, e.MonthlyUSD)
}
