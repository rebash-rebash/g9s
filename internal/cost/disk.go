package cost

import (
	"fmt"
	"strings"

	"github.com/rebash-rebash/g9s/internal/model"
)

const diskHoursPerMonth = 730.0

// DiskEstimate is an on-demand provisioned-capacity estimate for a zonal disk.
type DiskEstimate struct {
	Name       string
	Type       string
	SizeGB     int64
	HourlyUSD  float64
	MonthlyUSD float64
	Available  bool
	Source     string
}

// EstimateDisk uses current published us-central1 baseline rates. Actual
// billing may differ because of discounts, snapshots, regional replication,
// or other billing constructs.
func EstimateDisk(d model.Disk) DiskEstimate {
	rate, ok := diskRate(d.Type)
	if !ok || d.SizeGB <= 0 {
		return DiskEstimate{Name: d.Name, Type: d.Type, SizeGB: d.SizeGB, Source: "Google Cloud us-central1 baseline; not billing data"}
	}
	hourly := float64(d.SizeGB) * rate
	return DiskEstimate{
		Name:       d.Name,
		Type:       d.Type,
		SizeGB:     d.SizeGB,
		HourlyUSD:  hourly,
		MonthlyUSD: hourly * diskHoursPerMonth,
		Available:  true,
		Source:     "Google Cloud us-central1 on-demand disk pricing; estimate only",
	}
}

func diskRate(diskType string) (float64, bool) {
	switch strings.ToLower(diskType) {
	case "pd-standard":
		return 0.04 / diskHoursPerMonth, true
	case "pd-balanced":
		return 0.10 / diskHoursPerMonth, true
	case "pd-ssd":
		return 0.17 / diskHoursPerMonth, true
	case "pd-extreme":
		return 0.125 / diskHoursPerMonth, true
	default:
		return 0, false
	}
}

func FormatDiskEstimate(e DiskEstimate) string {
	if !e.Available {
		return "Pricing unavailable"
	}
	return fmt.Sprintf("$%.2f/month", e.MonthlyUSD)
}
