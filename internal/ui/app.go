package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rebash-rebash/g9s/internal/analyzer"
	"github.com/rebash-rebash/g9s/internal/cost"
	"github.com/rebash-rebash/g9s/internal/gcp"
	"github.com/rebash-rebash/g9s/internal/model"
)

type resourceItem struct{ name string; count string }
func (i resourceItem) Title() string { return i.name }
func (i resourceItem) Description() string { return i.count }
func (i resourceItem) FilterValue() string { return i.name }

type screen int
const (
	dashboardScreen screen = iota
	computeScreen
	disksScreen
	costScreen
)

type Model struct {
	project string
	list list.Model
	compute computeModel
	disks disksModel
	cost costModel
	computeSvc *gcp.ComputeService
	diskSvc *gcp.DiskService
	monitoringSvc *gcp.MonitoringService
	screen screen
	quitting bool
	width int
	height int
}

func New(project string, computeSvc *gcp.ComputeService, diskSvc *gcp.DiskService, monitoringSvc *gcp.MonitoringService) Model {
	items := []list.Item{
		resourceItem{"Compute Engine", "VM instances"},
		resourceItem{"GKE", "Kubernetes clusters"},
		resourceItem{"Disks", "Persistent disks"},
		resourceItem{"Cloud Storage", "Buckets and objects"},
		resourceItem{"Cost", "Cost, waste and savings"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "RESOURCES"
	l.Styles.Title = sectionStyle
	l.Styles.HelpStyle = mutedStyle
	l.Styles.PaginationStyle = mutedStyle
	l.SetShowStatusBar(false)
	return Model{project: project, list: l, compute: newComputeModel(0, 0, monitoringSvc), disks: newDisksModel(0, 0), cost: newCostModel(0, 0), computeSvc: computeSvc, diskSvc: diskSvc, monitoringSvc: monitoringSvc}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case computeScreen:
		if _, ok := msg.(backMsg); ok { m.screen = dashboardScreen; return m, nil }
		var cmd tea.Cmd; m.compute, cmd = m.compute.Update(msg); return m, cmd
	case disksScreen:
		if _, ok := msg.(backMsg); ok { m.screen = dashboardScreen; return m, nil }
		var cmd tea.Cmd; m.disks, cmd = m.disks.Update(msg); return m, cmd
	case costScreen:
		if _, ok := msg.(backMsg); ok { m.screen = dashboardScreen; return m, nil }
		var cmd tea.Cmd; m.cost, cmd = m.cost.Update(msg); return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" { m.quitting = true; return m, tea.Quit }
		if msg.String() == "enter" {
			if item, ok := m.list.SelectedItem().(resourceItem); ok {
				switch item.name {
				case "Compute Engine":
					m.screen = computeScreen; m.compute = newComputeModel(m.width, m.height-6, m.monitoringSvc); return m, m.loadVMs()
				case "Disks":
					m.screen = disksScreen; m.disks = newDisksModel(m.width, m.height-6); return m, loadDisks(m.diskSvc)
				case "Cost":
					m.screen = costScreen; m.cost = newCostModel(m.width, m.height-6); return m, m.loadCost()
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width-8, msg.Height-18)
	}
	var cmd tea.Cmd; m.list, cmd = m.list.Update(msg); return m, cmd
}

func (m Model) loadVMs() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		vms, err := m.computeSvc.ListVMs(ctx)
		if err != nil { return vmListMsg{err: err} }
		disks, err := m.diskSvc.ListDisks(ctx)
		return vmListMsg{vms: vms, disks: disks, err: err}
	}
}

func (m Model) loadCost() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		vms, err := m.computeSvc.ListVMs(ctx)
		if err != nil { return costListMsg{err: err} }
		disks, err := m.diskSvc.ListDisks(ctx)
		return costListMsg{vms: vms, disks: disks, err: err}
	}
}

func (m Model) View() string {
	if m.quitting { return "" }

	var content string
	switch m.screen {
	case computeScreen:
		content = m.compute.View()
	case disksScreen:
		content = m.disks.View()
	case costScreen:
		content = m.cost.View()
	default:
		content = renderDashboard(m.project, m.list, m.width, m.height)
	}

	return content
}

type costModel struct {
	loading bool
	err error
	vms []model.VM
	disks []model.Disk
	findings []model.Finding
	catalog cost.Catalog
	width int
	height int
}

func newCostModel(width, height int) costModel { return costModel{loading: true, catalog: cost.NewUSCentral1Catalog(), width: width, height: height} }

func (m costModel) Update(msg tea.Msg) (costModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case costListMsg:
		m.loading = false; m.vms = msg.vms; m.disks = msg.disks; m.err = msg.err
		if m.err == nil { m.findings = analyzer.BuildFindings(m.vms, m.disks, m.catalog) }
	case tea.KeyMsg:
		if msg.String() == "esc" { return m, func() tea.Msg { return backMsg{} } }
	}
	return m, nil
}

func (m costModel) View() string {
	if m.loading { return lipgloss.JoinVertical(lipgloss.Left, renderTopBar("", "Cost Intelligence"), "", mutedStyle.Render("Loading cost intelligence...")) }
	if m.err != nil { return lipgloss.JoinVertical(lipgloss.Left, renderTopBar("", "Cost Intelligence"), "", statusBadStyle.Render("Error: "+m.err.Error())) }

	var running, stopped, unsupported int
	var monthly float64
	for _, vm := range m.vms {
		if vm.Status == "RUNNING" { running++ } else { stopped++ }
		e := m.catalog.EstimateVM(vm)
		if !e.Available { unsupported++; continue }
		if vm.Status == "RUNNING" { monthly += e.MonthlyUSD }
	}

	potential := analyzer.PotentialMonthlySavings(m.findings)
	contentWidth := m.width - 4
	if contentWidth < 60 { contentWidth = 60 }
	cardWidth := (contentWidth - 6) / 4
	if cardWidth < 12 { cardWidth = 12 }

	cards := lipgloss.JoinHorizontal(lipgloss.Top,
		panelStyle.Width(cardWidth).Render(renderMetric("VMs", fmt.Sprintf("%d", len(m.vms)), appTitleStyle)),
		"  ",
		panelStyle.Width(cardWidth).Render(renderMetric("RUNNING", fmt.Sprintf("%d", running), statusOKStyle)),
		"  ",
		panelStyle.Width(cardWidth).Render(renderMetric("DISKS", fmt.Sprintf("%d", len(m.disks)), appTitleStyle)),
		"  ",
		panelStyle.Width(cardWidth).Render(renderMetric("FINDINGS", fmt.Sprintf("%d", len(m.findings)), statusWarnStyle)),
	)

	costPanel := panelStyle.Width(contentWidth).Render(lipgloss.JoinVertical(lipgloss.Left,
		sectionStyle.Render("COST OVERVIEW"),
		"",
		fmt.Sprintf("Compute estimate     %s / month", accentStyle.Render(fmt.Sprintf("$%.2f", monthly))),
		fmt.Sprintf("Daily run-rate       %s", mutedStyle.Render(fmt.Sprintf("$%.2f", monthly/30.0))),
		fmt.Sprintf("Potential savings    %s / month", statusOKStyle.Render(fmt.Sprintf("$%.2f", potential))),
		"",
		mutedStyle.Render("Baseline on-demand estimate • excludes discounts, network and actual billing"),
	))

	findingsTitle := sectionStyle.Render("TOP FINDINGS")
	findingLines := []string{}
	limit := len(m.findings); if limit > 6 { limit = 6 }
	for _, f := range m.findings[:limit] {
		save := ""
		if f.PotentialSave > 0 { save = fmt.Sprintf("  •  save $%.2f/mo", f.PotentialSave) }
		findingLines = append(findingLines, fmt.Sprintf("%-8s %-40s %s%s", "["+f.Severity+"]", truncate(f.ResourceID, 40), f.Reason, save))
	}
	if len(findingLines) == 0 { findingLines = append(findingLines, mutedStyle.Render("No optimization findings detected.")) }
	findingsPanel := panelStyle.Width(contentWidth).Render(lipgloss.JoinVertical(lipgloss.Left, findingsTitle, "", lipgloss.JoinVertical(lipgloss.Left, findingLines...)))

	return lipgloss.JoinVertical(lipgloss.Left,
		renderTopBar("", "Cost Intelligence"),
		"",
		cards,
		"",
		costPanel,
		"",
		findingsPanel,
		"",
		mutedStyle.Render("Findings are review candidates • no destructive action is automatic • esc back"),
	)
}

func truncate(value string, max int) string {
	if len(value) <= max { return value }
	if max < 4 { return value[:max] }
	return value[:max-3] + "..."
}

type costListMsg struct { vms []model.VM; disks []model.Disk; err error }
