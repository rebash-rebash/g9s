package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rebash-rebash/g9s/internal/model"
)

type vmItem struct{ vm model.VM }

func (i vmItem) Title() string { return i.vm.Name }
func (i vmItem) Description() string {
	return fmt.Sprintf("%s  •  %s  •  %s", i.vm.Zone, i.vm.MachineType, i.vm.Status)
}
func (i vmItem) FilterValue() string { return i.vm.Name }

type computeModel struct {
	list    list.Model
	loading bool
	err     error
	detail  *model.VM
}

func newComputeModel(width, height int) computeModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	l.Title = "Compute Engine — Virtual Machines"
	return computeModel{list: l, loading: true}
}

func (m computeModel) Update(msg tea.Msg) (computeModel, tea.Cmd) {
	if m.detail != nil {
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
			m.detail = nil
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
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m computeModel) View() string {
	if m.detail != nil {
		return renderVMDetails(*m.detail)
	}
	if m.loading {
		return "Loading Compute Engine instances..."
	}
	if m.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: " + m.err.Error())
	}
	return m.list.View() + "\n" + lipgloss.NewStyle().Faint(true).Render("↑↓ navigate  / search  enter details  esc back") + "\n"
}

func renderVMDetails(vm model.VM) string {
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
		lipgloss.NewStyle().Faint(true).Render("esc back"),
	}
	return lipgloss.JoinVertical(lipgloss.Left, body...)
}

type vmListMsg struct {
	vms []model.VM
	err error
}

type backMsg struct{}
