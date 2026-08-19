package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rebash-rebash/g9s/internal/analyzer"
	"github.com/rebash-rebash/g9s/internal/cost"
	"github.com/rebash-rebash/g9s/internal/gcp"
	"github.com/rebash-rebash/g9s/internal/model"
)

type vmItem struct{ vm model.VM }

func (i vmItem) Title() string { return i.vm.Name }
func (i vmItem) Description() string {
	return fmt.Sprintf("%s  •  %s  •  %s", i.vm.Zone, i.vm.MachineType, i.vm.Status)
}
func (i vmItem) FilterValue() string { return i.vm.Name }

type computeModel struct {
	list           list.Model
	loading        bool
	err            error
	detail         *model.VM
	disks          []model.Disk
	monitoring     *gcp.MonitoringService
	utilization    model.Utilization
	ioStats        model.IOStats
	metricsLoading bool
	metricsErr     error
	catalog        cost.Catalog
}

func newComputeModel(width, height int, monitoring *gcp.MonitoringService) computeModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	l.Title = "Compute Engine — Virtual Machines"
	return computeModel{list: l, loading: true, monitoring: monitoring, catalog: cost.NewUSCentral1Catalog()}
}

func (m computeModel) Update(msg tea.Msg) (computeModel, tea.Cmd) {
	if m.detail != nil {
		switch key := msg.(type) {
		case tea.KeyMsg:
			if key.String() == "esc" {
				m.detail = nil
				m.utilization = model.Utilization{}
				m.ioStats = model.IOStats{}
				m.metricsLoading = false
				m.metricsErr = nil
			}
		case vmMetricsMsg:
			m.metricsLoading = false
			m.utilization = key.utilization
			m.ioStats = key.ioStats
			m.metricsErr = key.err
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-4)
	case vmListMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			break
		}
		m.disks = msg.disks
		items := make([]list.Item, 0, len(msg.vms))
		for _, vm := range msg.vms {
			items = append(items, vmItem{vm: vm})
		}
		m.list.SetItems(items)
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return backMsg{} }
		case "enter":
			if item, ok := m.list.SelectedItem().(vmItem); ok {
				vm := item.vm
				m.detail = &vm
				m.utilization = model.Utilization{}
				m.ioStats = model.IOStats{}
				m.metricsErr = nil
				m.metricsLoading = vm.Status == "RUNNING" && m.monitoring != nil
				if m.metricsLoading {
					return m, m.loadMetrics(vm.InstanceID)
				}
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m computeModel) loadMetrics(instanceID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		cpu, cpuErr := m.monitoring.GetCPUUtilization(ctx, instanceID)
		if cpuErr != nil {
			return vmMetricsMsg{utilization: cpu, err: cpuErr}
		}
		ioStats, ioErr := m.monitoring.GetIOStats(ctx, instanceID)
		return vmMetricsMsg{utilization: cpu, ioStats: ioStats, err: ioErr}
	}
}

func (m computeModel) View() string {
	if m.detail != nil {
		return renderVMDetails(*m.detail, m.disks, m.monitoring != nil, m.metricsLoading, m.utilization, m.ioStats, m.metricsErr, m.catalog)
	}
	if m.loading {
		return "Loading Compute Engine instances..."
	}
	if m.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: " + m.err.Error())
	}
	return m.list.View() + "\n" + lipgloss.NewStyle().Faint(true).Render("↑↓ navigate  / search  enter details  esc back") + "\n"
}

func renderVMDetails(vm model.VM, disks []model.Disk, monitoringAvailable, metricsLoading bool, utilization model.Utilization, ioStats model.IOStats, metricsErr error, catalog cost.Catalog) string {
	title := lipgloss.NewStyle().Bold(true).Render("Compute Engine — VM Details")
	label := lipgloss.NewStyle().Bold(true)
	line := func(name, value string) string {
		if value == "" {
			value = "-"
		}
		return fmt.Sprintf("%-18s %s", label.Render(name), value)
	}

	body := []string{
		title,
		"",
		line("Name", vm.Name),
		line("Status", vm.Status),
		line("Zone", vm.Zone),
		line("Machine Type", vm.MachineType),
		line("Internal IP", vm.InternalIP),
		line("External IP", vm.ExternalIP),
		line("Boot Disk", vm.BootDisk),
		line("Attached Disks", fmt.Sprintf("%d", vm.DiskCount)),
		line("Networks", fmt.Sprintf("%d", vm.NetworkCount)),
		line("Created", vm.CreationTime),
	}

	computeEstimate := catalog.EstimateVM(vm)
	diskMonthly := attachedDiskMonthly(vm, disks)
	body = append(body,
		"",
		lipgloss.NewStyle().Bold(true).Render("ESTIMATED MONTHLY COST"),
		line("Compute", formatUSD(computeEstimate.MonthlyUSD)),
		line("Persistent Disk", formatUSD(diskMonthly)),
		line("Total", formatUSD(computeEstimate.MonthlyUSD+diskMonthly)),
		line("Pricing", "Baseline estimate — not billing data"),
	)

	body = append(body, "", lipgloss.NewStyle().Bold(true).Render("CPU UTILIZATION (24h)"))
	if vm.Status != "RUNNING" {
		body = append(body, "Not running — usage metrics are not evaluated.")
	} else if !monitoringAvailable {
		body = append(body, "Cloud Monitoring is unavailable.")
	} else if metricsLoading {
		body = append(body, "Loading Cloud Monitoring metrics...")
	} else if metricsErr != nil {
		body = append(body, lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Monitoring error: "+metricsErr.Error()))
	} else if utilization.Average == nil {
		body = append(body, "No CPU metric data available.")
	} else {
		body = append(body,
			line("Current", formatRatio(utilization.Current)),
			line("Average", formatRatio(utilization.Average)),
			line("P95", formatRatio(utilization.P95)),
			"",
			line("Assessment", assessCPU(utilization)),
			"",
			lipgloss.NewStyle().Bold(true).Render("INTELLIGENCE"),
			line("Usage", analyzer.VMUsageStatus(vm, utilization, ioStats)),
			line("Recommendation", analyzer.Recommendation(vm, utilization, ioStats)),
		)
	}

	body = append(body, "", lipgloss.NewStyle().Bold(true).Render("NETWORK (24h)"))
	if vm.Status != "RUNNING" || metricsLoading || metricsErr != nil {
		body = append(body, "No network throughput data available.")
	} else {
		body = append(body,
			line("Receive avg", formatBytesPerSecond(ioStats.NetworkInAverage)),
			line("Receive P95", formatBytesPerSecond(ioStats.NetworkInP95)),
			line("Send avg", formatBytesPerSecond(ioStats.NetworkOutAverage)),
			line("Send P95", formatBytesPerSecond(ioStats.NetworkOutP95)),
		)
	}

	body = append(body, "", lipgloss.NewStyle().Bold(true).Render("DISK I/O (24h)"))
	if vm.Status != "RUNNING" || metricsLoading || metricsErr != nil {
		body = append(body, "No disk throughput data available.")
	} else {
		body = append(body,
			line("Read avg", formatBytesPerSecond(ioStats.DiskReadAverage)),
			line("Read P95", formatBytesPerSecond(ioStats.DiskReadP95)),
			line("Write avg", formatBytesPerSecond(ioStats.DiskWriteAverage)),
			line("Write P95", formatBytesPerSecond(ioStats.DiskWriteP95)),
		)
	}

	body = append(body, "", lipgloss.NewStyle().Faint(true).Render("esc back"))
	return lipgloss.JoinVertical(lipgloss.Left, body...)
}

func attachedDiskMonthly(vm model.VM, disks []model.Disk) float64 {
	var total float64
	for _, d := range disks {
		attached := strings.Contains(strings.ToLower(d.Name), strings.ToLower(vm.BootDisk)) && vm.BootDisk != ""
		if !attached {
			for _, user := range d.Users {
				if strings.Contains(user, "/instances/"+vm.Name) {
					attached = true
					break
				}
			}
		}
		if attached {
			e := cost.EstimateDisk(d)
			if e.Available {
				total += e.MonthlyUSD
			}
		}
	}
	return total
}

func formatUSD(value float64) string {
	if value <= 0 {
		return "$0.00"
	}
	return fmt.Sprintf("$%.2f", value)
}

func formatRatio(value *float64) string {
	if value == nil { return "N/A" }
	return fmt.Sprintf("%.1f%%", *value*100)
}

func formatBytesPerSecond(value *float64) string {
	if value == nil { return "N/A" }
	v := *value
	switch {
	case v >= 1024*1024*1024: return fmt.Sprintf("%.2f GiB/s", v/(1024*1024*1024))
	case v >= 1024*1024: return fmt.Sprintf("%.2f MiB/s", v/(1024*1024))
	case v >= 1024: return fmt.Sprintf("%.2f KiB/s", v/1024)
	default: return fmt.Sprintf("%.0f B/s", v)
	}
}

func assessCPU(util model.Utilization) string {
	if util.P95 == nil { return "UNKNOWN" }
	switch {
	case *util.P95 < 0.10: return "VERY LOW"
	case *util.P95 < 0.30: return "LOW"
	case *util.P95 < 0.70: return "NORMAL"
	default: return "HIGH"
	}
}

type vmListMsg struct {
	vms   []model.VM
	disks []model.Disk
	err   error
}

type vmMetricsMsg struct {
	utilization model.Utilization
	ioStats     model.IOStats
	err         error
}

type backMsg struct{}
