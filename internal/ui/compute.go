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
func (i vmItem) Title() string { return fmt.Sprintf("%-42s %-15s %-18s %s", truncate(i.vm.Name, 42), truncate(i.vm.Zone, 15), truncate(i.vm.MachineType, 18), i.vm.Status) }
func (i vmItem) Description() string { return "" }
func (i vmItem) FilterValue() string { return i.vm.Name + " " + i.vm.Zone + " " + i.vm.MachineType + " " + i.vm.Status }

type computeModel struct { list list.Model; loading bool; err error; detail *model.VM; disks []model.Disk; monitoring *gcp.MonitoringService; utilization model.Utilization; ioStats model.IOStats; metricsLoading bool; metricsErr error; catalog cost.Catalog }

func newComputeModel(width, height int, monitoring *gcp.MonitoringService) computeModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	l.Title = ""
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.Styles.HelpStyle = mutedStyle
	l.Styles.StatusBar = mutedStyle
	return computeModel{list: l, loading: true, monitoring: monitoring, catalog: cost.NewUSCentral1Catalog()}
}

func (m computeModel) Update(msg tea.Msg) (computeModel, tea.Cmd) {
	if m.detail != nil {
		switch key := msg.(type) {
		case tea.KeyMsg:
			if key.String() == "esc" { m.detail = nil; m.utilization = model.Utilization{}; m.ioStats = model.IOStats{}; m.metricsLoading = false; m.metricsErr = nil }
		case vmMetricsMsg:
			m.metricsLoading = false; m.utilization = key.utilization; m.ioStats = key.ioStats; m.metricsErr = key.err
		}
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(maxInt(30, msg.Width-4), maxInt(6, msg.Height-8))
	case vmListMsg:
		m.loading = false
		if msg.err != nil { m.err = msg.err; break }
		m.disks = msg.disks
		items := make([]list.Item, 0, len(msg.vms))
		for _, vm := range msg.vms { items = append(items, vmItem{vm: vm}) }
		m.list.SetItems(items)
	case tea.KeyMsg:
		switch msg.String() {
		case "esc": return m, func() tea.Msg { return backMsg{} }
		case "enter":
			if item, ok := m.list.SelectedItem().(vmItem); ok {
				vm := item.vm; m.detail = &vm; m.utilization = model.Utilization{}; m.ioStats = model.IOStats{}; m.metricsErr = nil
				m.metricsLoading = vm.Status == "RUNNING" && m.monitoring != nil
				if m.metricsLoading { return m, m.loadMetrics(vm.InstanceID) }
			}
		}
	}
	var cmd tea.Cmd; m.list, cmd = m.list.Update(msg); return m, cmd
}

func (m computeModel) loadMetrics(instanceID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background(); cpu, cpuErr := m.monitoring.GetCPUUtilization(ctx, instanceID)
		if cpuErr != nil { return vmMetricsMsg{utilization: cpu, err: cpuErr} }
		ioStats, ioErr := m.monitoring.GetIOStats(ctx, instanceID)
		return vmMetricsMsg{utilization: cpu, ioStats: ioStats, err: ioErr}
	}
}

func (m computeModel) View() string {
	var content string
	if m.detail != nil { content = renderVMDetails(*m.detail, m.disks, m.monitoring != nil, m.metricsLoading, m.utilization, m.ioStats, m.metricsErr, m.catalog, m.list.Width()) } else if m.loading { content = lipgloss.JoinVertical(lipgloss.Left, renderTopBar("", "Compute Engine"), "", mutedStyle.Render("Loading VM inventory...")) } else if m.err != nil { content = lipgloss.JoinVertical(lipgloss.Left, renderTopBar("", "Compute Engine"), "", statusBadStyle.Render("Error: "+m.err.Error())) } else {
		header := mutedStyle.Render(fmt.Sprintf("%-42s %-15s %-18s %s", "NAME", "ZONE", "MACHINE TYPE", "STATUS"))
		content = lipgloss.JoinVertical(lipgloss.Left, renderTopBar("", "Compute Engine"), mutedStyle.Render(fmt.Sprintf("%d instances  •  select a VM to inspect utilization, I/O and cost", len(m.list.Items()))), "", header, m.list.View(), "", renderFooter("↑↓ navigate  •  / search  •  enter details  •  esc back"))
	}
	return renderFrame(m.list.Width()+4, m.list.Height()+8, content)
}

func renderVMDetails(vm model.VM, disks []model.Disk, monitoringAvailable, metricsLoading bool, utilization model.Utilization, ioStats model.IOStats, metricsErr error, catalog cost.Catalog, width int) string {
	label := lipgloss.NewStyle().Bold(true)
	line := func(name, value string) string { if value == "" { value = "-" }; return fmt.Sprintf("%-18s %s", label.Render(name), value) }
	contentWidth := width - 4
	if contentWidth < 60 { contentWidth = 60 }
	identityWidth := contentWidth * 58 / 100
	costWidth := contentWidth - identityWidth - 2
	if identityWidth < 40 { identityWidth = 40 }
	if costWidth < 24 { costWidth = 24 }
	computeEstimate := catalog.EstimateVM(vm); diskMonthly := attachedDiskMonthly(vm, disks); status := statusStyle(vm.Status).Render(vm.Status)
	identity := panelStyle.Width(identityWidth).Render(lipgloss.JoinVertical(lipgloss.Left, sectionStyle.Render("INSTANCE"), "", line("Name", vm.Name), line("Status", status), line("Zone", vm.Zone), line("Machine Type", vm.MachineType), line("Internal IP", vm.InternalIP), line("External IP", vm.ExternalIP), line("Boot Disk", vm.BootDisk), line("Attached Disks", fmt.Sprintf("%d", vm.DiskCount)), line("Networks", fmt.Sprintf("%d", vm.NetworkCount)), line("Created", vm.CreationTime)))
	costPanel := panelStyle.Width(costWidth).Render(lipgloss.JoinVertical(lipgloss.Left, sectionStyle.Render("COST ESTIMATE"), "", line("Compute", formatUSD(computeEstimate.MonthlyUSD)+" / mo"), line("Persistent Disk", formatUSD(diskMonthly)+" / mo"), line("Total", accentStyle.Render(formatUSD(computeEstimate.MonthlyUSD+diskMonthly)+" / mo")), "", mutedStyle.Render("Baseline estimate"), mutedStyle.Render("not billing data")))
	body := []string{renderTopBar("", "Compute Engine / VM Details"), "", lipgloss.JoinHorizontal(lipgloss.Top, identity, "  ", costPanel), ""}
	metricsTitle := sectionStyle.Render("UTILIZATION & TELEMETRY  /  24H")
	if vm.Status != "RUNNING" { body = append(body, panelStyle.Width(contentWidth).Render(lipgloss.JoinVertical(lipgloss.Left, metricsTitle, "", mutedStyle.Render("VM is not running — live utilization is not evaluated.")))) } else if !monitoringAvailable { body = append(body, panelStyle.Width(contentWidth).Render(lipgloss.JoinVertical(lipgloss.Left, metricsTitle, "", mutedStyle.Render("Cloud Monitoring is unavailable.")))) } else if metricsLoading { body = append(body, panelStyle.Width(contentWidth).Render(lipgloss.JoinVertical(lipgloss.Left, metricsTitle, "", mutedStyle.Render("Loading Cloud Monitoring metrics...")))) } else if metricsErr != nil { body = append(body, panelStyle.Width(contentWidth).Render(lipgloss.JoinVertical(lipgloss.Left, metricsTitle, "", statusBadStyle.Render("Monitoring error: "+metricsErr.Error())))) } else if utilization.Average == nil { body = append(body, panelStyle.Width(contentWidth).Render(lipgloss.JoinVertical(lipgloss.Left, metricsTitle, "", mutedStyle.Render("No CPU metric data available.")))) } else {
		cpu := lipgloss.JoinHorizontal(lipgloss.Top, renderMetric("CPU CURRENT", formatRatio(utilization.Current), appTitleStyle), "    ", renderMetric("CPU AVG", formatRatio(utilization.Average), appTitleStyle), "    ", renderMetric("CPU P95", formatRatio(utilization.P95), appTitleStyle), "    ", renderMetric("ASSESSMENT", assessCPU(utilization), statusWarnStyle))
		intel := lipgloss.JoinVertical(lipgloss.Left, sectionStyle.Render("INTELLIGENCE"), "", line("Usage", analyzer.VMUsageStatus(vm, utilization, ioStats)), line("Recommendation", analyzer.Recommendation(vm, utilization, ioStats)))
		network := lipgloss.JoinVertical(lipgloss.Left, sectionStyle.Render("NETWORK"), "", line("Receive avg", formatBytesPerSecond(ioStats.NetworkInAverage)), line("Receive P95", formatBytesPerSecond(ioStats.NetworkInP95)), line("Send avg", formatBytesPerSecond(ioStats.NetworkOutAverage)), line("Send P95", formatBytesPerSecond(ioStats.NetworkOutP95)))
		diskIO := lipgloss.JoinVertical(lipgloss.Left, sectionStyle.Render("DISK I/O"), "", line("Read avg", formatBytesPerSecond(ioStats.DiskReadAverage)), line("Read P95", formatBytesPerSecond(ioStats.DiskReadP95)), line("Write avg", formatBytesPerSecond(ioStats.DiskWriteAverage)), line("Write P95", formatBytesPerSecond(ioStats.DiskWriteP95)))
		body = append(body, panelStyle.Width(contentWidth).Render(lipgloss.JoinVertical(lipgloss.Left, metricsTitle, "", cpu, "", intel, "", lipgloss.JoinHorizontal(lipgloss.Top, network, "        ", diskIO))))
	}
	body = append(body, "", renderFooter("esc back")); return lipgloss.JoinVertical(lipgloss.Left, body...)
}

func attachedDiskMonthly(vm model.VM, disks []model.Disk) float64 { var total float64; for _, d := range disks { attached := strings.Contains(strings.ToLower(d.Name), strings.ToLower(vm.BootDisk)) && vm.BootDisk != ""; if !attached { for _, user := range d.Users { if strings.Contains(user, "/instances/"+vm.Name) { attached = true; break } } }; if attached { if e := cost.EstimateDisk(d); e.Available { total += e.MonthlyUSD } } }; return total }
func formatUSD(value float64) string { if value <= 0 { return "$0.00" }; return fmt.Sprintf("$%.2f", value) }
func formatRatio(value *float64) string { if value == nil { return "N/A" }; return fmt.Sprintf("%.1f%%", *value*100) }
func formatBytesPerSecond(value *float64) string { if value == nil { return "N/A" }; v:=*value; switch { case v>=1024*1024*1024:return fmt.Sprintf("%.2f GiB/s",v/(1024*1024*1024)); case v>=1024*1024:return fmt.Sprintf("%.2f MiB/s",v/(1024*1024)); case v>=1024:return fmt.Sprintf("%.2f KiB/s",v/1024); default:return fmt.Sprintf("%.0f B/s",v) } }
func assessCPU(util model.Utilization) string { if util.P95 == nil { return "UNKNOWN" }; switch { case *util.P95<0.10:return "VERY LOW"; case *util.P95<0.30:return "LOW"; case *util.P95<0.70:return "NORMAL"; default:return "HIGH" } }
func maxInt(a,b int) int { if a>b{return a}; return b }

type vmListMsg struct { vms []model.VM; disks []model.Disk; err error }
type vmMetricsMsg struct { utilization model.Utilization; ioStats model.IOStats; err error }
type backMsg struct{}
