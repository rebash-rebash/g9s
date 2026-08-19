package analyzer

import (
	"fmt"
	"sort"

	"github.com/rebash-rebash/g9s/internal/cost"
	"github.com/rebash-rebash/g9s/internal/model"
)

// BuildFindings creates conservative, review-oriented FinOps findings.
// Findings never imply that a resource is safe to delete automatically.
func BuildFindings(vms []model.VM, disks []model.Disk, catalog cost.Catalog) []model.Finding {
	findings := make([]model.Finding, 0)

	for _, d := range disks {
		if d.Attached {
			continue
		}
		e := cost.EstimateDisk(d)
		if !e.Available || e.MonthlyUSD <= 0 {
			continue
		}
		findings = append(findings, model.Finding{
			ResourceID:     d.Name,
			ResourceType:   "Persistent Disk",
			Severity:       severityForMonthlyCost(e.MonthlyUSD),
			Reason:         fmt.Sprintf("%d GiB %s disk is unattached.", d.SizeGB, d.Type),
			MonthlyCost:    e.MonthlyUSD,
			PotentialSave:  e.MonthlyUSD,
			Recommendation: "Review whether this disk is still required; delete only after confirming it is unused.",
		})
	}

	for _, vm := range vms {
		if vm.Status == "RUNNING" {
			continue
		}
		findings = append(findings, model.Finding{
			ResourceID:     vm.Name,
			ResourceType:   "Compute Engine VM",
			Severity:       "INFO",
			Reason:         fmt.Sprintf("VM is %s; compute charges are currently $0, but attached storage may still incur cost.", vm.Status),
			Recommendation: "Review attached disks and retained networking resources before deleting the VM.",
		})
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].PotentialSave == findings[j].PotentialSave {
			return findings[i].Severity < findings[j].Severity
		}
		return findings[i].PotentialSave > findings[j].PotentialSave
	})
	return findings
}

func severityForMonthlyCost(monthly float64) string {
	switch {
	case monthly >= 50:
		return "HIGH"
	case monthly >= 10:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

func PotentialMonthlySavings(findings []model.Finding) float64 {
	var total float64
	for _, f := range findings {
		total += f.PotentialSave
	}
	return total
}
