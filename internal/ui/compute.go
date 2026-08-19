package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	list         list.Model
	loading      bool
	err          error
	detail       *model.VM
	monitoring   *gcp.MonitoringService
	utilization  model.Utilization
	utilLoading  bool
	utilErr      error
}

func newComputeModel(width, height int, monitoring *gcp.MonitoringService) computeModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	l.Title = "Compute Engine — Virtual Machines"
	return computeModel{list: l, loading: true, monitoring: monitoring}
}

func (m computeModel) Update(msg tea.Msg) (computeModel, tea.Cmd) {
	if m.detail != nil {
		switch key := msg.(type) {
		case tea.KeyMsg:
			if key.String() == "esc" {
				m.detail = nil
				m.utilization = model.Utilization{}
				m.utilLoading = false
				m.utilErr = nil
			}
		case cpuUtilizationMsg:
			m.utilLoading = false
			m.utilization = msg.utilization
			m.utilErr = msg.err
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
				m.utilErr = nil
				m.utilLoading = vm.Status == "RUNNING" && m.monitoring != nil
				if m.utilLoading {
					return m, m.loadCPU(vm.InstanceID)
				}
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m computeModel) loadCPU(instanceID string) tea.Cmd {
	return func() tea.Msg {
		utilization, err := m.monitoring.GetCPUUtilization(context.Background(), instanceID)
		return cpuUtilizationMsg{utilization: utilization, err: err}
	}
}

func (m computeModel) View() string {
	if m.detail != nil {
		return renderVMDetails(*m.detail, m.utilLoading, m.utilization, m.utilErr)
	}
	if m.loading {
		return "Loading Compute Engine instances..."
	}
	if m.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: " + m.err.Error())
	}
	return m.list.View() + "\n" + lipgloss.NewStyle().Faint(true).Render("↑↓ navigate  / search  enter details  esc back") + "\n"
}

func renderVMDetails(vm model.VM, utilLoading bool, utilization model.Utilization, utilErr error) string {
	title := lipgloss.NewStyle().Bold(true).Render("Compute Engine — VM Details")
	label := lipgloss.NewStyle().Bold(true)
	line := func(name, value string) string {
		if value == "" {
			value = "-"
		}
		return fmt.Sprintf("%-16s %s", label.Render(name), value)
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
		"",
		lipgloss.NewStyle().Bold(true).Render("CPU UTILIZATION (24h)"),
	}

	if vm.Status != "RUNNING" {
		body = append(body, "Not running — CPU metrics are not evaluated.")
	} else if mmonitoringUnavailable(utilLoading, utilErr) {
		body = append(body, "Cloud Monitoring is unavailable.")
	} else if utilLoading {
		body = append(body, "Loading Cloud Monitoring metrics...")
	} else if utilErr != nil {
		body = append(body, lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Monitoring error: "+utilErr.Error()))
	} else if utilization.Average == nil {
		body = append(body, "No CPU metric data available.")
	} else {
		body = append(body,
			line("Current", formatRatio(utilization.Current)),
			line("Average", formatRatio(utilization.Average)),
			line("P95", formatRatio(utilization.P95)),
			"",
			line("Assessment", assessCPU(utilization)),
		)
	}

	body = append(body, "", lipgloss.NewStyle().Faint(true).Render("esc back"))
	return lipgloss.JoinVertical(lipgloss.Left, body...)
}

func mmonitoringUnavailable(loading bool, err error) bool {
	return !loading && err == nil && false
}

func formatRatio(value *float64) string {
	if value == nil {
		return "N/A"
	}
	return fmt.Sprintf("%.1f%%", *value*100)
}

func assessCPU(util model.Utilization) string {
	if util.P95 == nil {
		return "UNKNOWN"
	}
	switch {
	case *util.P95 < 0.10:
		return "VERY LOW"
	case *util.P95 < 0.30:
		return "LOW"
	case *util.P95 < 0.70:
		return "NORMAL"
	default:
		return "HIGH"
	}
}

type vmListMsg struct {
	vms []model.VM
	err error
}

type cpuUtilizationMsg struct {
	utilization model.Utilization
	err         error
}

type backMsg struct{}
