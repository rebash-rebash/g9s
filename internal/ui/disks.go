package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rebash-rebash/g9s/internal/cost"
	"github.com/rebash-rebash/g9s/internal/gcp"
	"github.com/rebash-rebash/g9s/internal/model"
)

type diskItem struct{ disk model.Disk }

func (i diskItem) Title() string { return i.disk.Name }
func (i diskItem) Description() string {
	status := "UNATTACHED"
	if i.disk.Attached {
		status = "ATTACHED"
	}
	return fmt.Sprintf("%s  •  %d GiB  •  %s  •  %s", i.disk.Zone, i.disk.SizeGB, i.disk.Type, status)
}
func (i diskItem) FilterValue() string { return i.disk.Name }

type disksModel struct {
	list    list.Model
	loading bool
	err     error
	disks   []model.Disk
}

func newDisksModel(width, height int) disksModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	l.Title = "Persistent Disks — Cost & Usage"
	return disksModel{list: l, loading: true}
}

func (m disksModel) Update(msg tea.Msg) (disksModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-4)
	case diskListMsg:
		m.loading = false
		m.disks = msg.disks
		m.err = msg.err
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return m, func() tea.Msg { return backMsg{} }
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m disksModel) View() string {
	if m.loading {
		return "Loading Persistent Disks..."
	}
	if m.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: " + m.err.Error())
	}

	var monthly, unattachedMonthly float64
	var unattached int
	items := make([]list.Item, 0, len(m.disks))
	for _, d := range m.disks {
		e := cost.EstimateDisk(d)
		if e.Available {
			monthly += e.MonthlyUSD
			if !d.Attached {
				unattachedMonthly += e.MonthlyUSD
			}
		}
		if !d.Attached {
			unattached++
		}
		items = append(items, diskItem{disk: d})
	}
	m.list.SetItems(items)

	header := lipgloss.NewStyle().Bold(true).Render("Persistent Disks — Cost & Usage")
	summary := fmt.Sprintf("Disks %d  •  Unattached %d  •  Estimated monthly $%.2f  •  Unattached monthly waste candidate $%.2f", len(m.disks), unattached, monthly, unattachedMonthly)
	footer := lipgloss.NewStyle().Faint(true).Render("↑↓ navigate  / search  esc back")
	return header + "\n" + summary + "\n\n" + m.list.View() + "\n" + footer + "\n"
}

func loadDisks(svc *gcp.DiskService) tea.Cmd {
	return func() tea.Msg {
		disks, err := svc.ListDisks(context.Background())
		return diskListMsg{disks: disks, err: err}
	}
}

type diskListMsg struct {
	disks []model.Disk
	err   error
}
