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
		Source:          "baseline N1 on-demand estimate; not billing data",
	}
}

func (c Catalog) EstimateVM(vm model.VM) VMEstimate {
	vCPU, memoryGB, supported := parseMachineType(vm.MachineType)
	if !supported {
		return VMEstimate{MachineType: vm.MachineType, VCPU: vCPU, MemoryGB: memoryGB, Source: c.Source}
	}

	// Stopped/terminated VMs do not consume vCPU/memory. Persistent disks and
	// retained IPs are separate cost components and will be added later.
	if vm.Status != "RUNNING" {
		return VMEstimate{
			MachineType: vm.MachineType,
			VCPU:        vCPU,
			MemoryGB:    memoryGB,
			Source:      c.Source,
			Available:   true,
		}
	}

	hourly := float64(vCPU)*c.VCPUHourUSD + memoryGB*c.MemoryGBHourUSD
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

var n1MachinePattern = regexp.MustCompile(`^n1-(?:standard|highmem|highcpu)-([0-9]+)$`)

func parseMachineType(machineType string) (int, float64, bool) {
	machineType = strings.TrimSpace(machineType)
	match := n1MachinePattern.FindStringSubmatch(machineType)
	if len(match) != 2 {
		return 0, 0, false
	}
	vCPU, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, 0, false
	}

	var memoryByCPU map[int]float64
	switch {
	case strings.HasPrefix(machineType, "n1-standard-"):
		memoryByCPU = map[int]float64{1: 3.75, 2: 7.5, 4: 15, 8: 30, 16: 60, 32: 120, 64: 240, 96: 360}
	case strings.HasPrefix(machineType, "n1-highmem-"):
		memoryByCPU = map[int]float64{2: 13, 4: 26, 8: 52, 16: 104, 32: 208, 64: 416, 96: 624}
	case strings.HasPrefix(machineType, "n1-highcpu-"):
		memoryByCPU = map[int]float64{2: 1.8, 4: 3.6, 8: 7.2, 16: 14.4, 32: 28.8}
	default:
		return 0, 0, false
	}
	memoryGB, ok := memoryByCPU[vCPU]
	return vCPU, memoryGB, ok
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
