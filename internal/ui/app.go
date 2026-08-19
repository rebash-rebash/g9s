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

type resourceItem struct { name string; count string }
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
		resourceItem{"Compute Engine", "Explore virtual machines"},
		resourceItem{"GKE", "Explore Kubernetes clusters"},
		resourceItem{"Disks", "Explore persistent disks"},
		resourceItem{"Cloud Storage", "Explore buckets"},
		resourceItem{"Cost", "Cost and utilization intelligence"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "G9S — GCP Resource & Cost Explorer"
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
					m.screen = computeScreen; m.compute = newComputeModel(m.width, m.height-4, m.monitoringSvc); return m, m.loadVMs()
				case "Disks":
					m.screen = disksScreen; m.disks = newDisksModel(m.width, m.height-4); return m, loadDisks(m.diskSvc)
				case "Cost":
					m.screen = costScreen; m.cost = newCostModel(m.width, m.height-4); return m, m.loadCost()
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width, msg.Height-4)
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
	switch m.screen { case computeScreen: return m.compute.View(); case disksScreen: return m.disks.View(); case costScreen: return m.cost.View() }
	header := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Project: %s", m.project))
	footer := lipgloss.NewStyle().Faint(true).Render("↑↓ navigate  / search  enter open  q quit")
	return header + "\n\n" + m.list.View() + "\n" + footer + "\n"
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
	if m.loading { return "Loading cost intelligence..." }
	if m.err != nil { return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: " + m.err.Error()) }
	var running, stopped, unsupported int
	var monthly float64
	for _, vm := range m.vms {
		e := m.catalog.EstimateVM(vm)
		if !e.Available { unsupported++; continue }
		if vm.Status == "RUNNING" { running++; monthly += e.MonthlyUSD } else { stopped++ }
	}
	potential := analyzer.PotentialMonthlySavings(m.findings)
	style := lipgloss.NewStyle().Bold(true)
	lines := []string{
		style.Render("G9S — Cost Intelligence"), "",
		fmt.Sprintf("Compute VMs        %d", len(m.vms)),
		fmt.Sprintf("Running VMs        %d", running),
		fmt.Sprintf("Stopped VMs        %d", stopped),
		fmt.Sprintf("Persistent Disks   %d", len(m.disks)),
		fmt.Sprintf("Findings            %d", len(m.findings)), "",
		style.Render("ESTIMATED COMPUTE COST"),
		fmt.Sprintf("Monthly            $%.2f", monthly),
		fmt.Sprintf("Daily              $%.2f", monthly/30.0), "",
		style.Render("POTENTIAL MONTHLY SAVINGS"),
		fmt.Sprintf("$%.2f", potential), "",
	}
	if len(m.findings) == 0 {
		lines = append(lines, "No optimization findings detected.")
	} else {
		lines = append(lines, style.Render("TOP FINDINGS"))
		limit := len(m.findings); if limit > 8 { limit = 8 }
		for _, f := range m.findings[:limit] {
			save := ""; if f.PotentialSave > 0 { save = fmt.Sprintf(" • save $%.2f/mo", f.PotentialSave) }
			lines = append(lines, fmt.Sprintf("[%s] %s — %s%s", f.Severity, f.ResourceID, f.Reason, save))
		}
	}
	lines = append(lines, "", lipgloss.NewStyle().Faint(true).Render("Findings are review candidates; no destructive action is automatic."), lipgloss.NewStyle().Faint(true).Render("Baseline estimate • excludes discounts, network, and actual billing"), lipgloss.NewStyle().Faint(true).Render("esc back"))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

type costListMsg struct { vms []model.VM; disks []model.Disk; err error }
