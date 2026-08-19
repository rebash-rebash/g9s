package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rebash-rebash/g9s/internal/gcp"
)

type resourceItem struct {
	name  string
	count string
}

func (i resourceItem) Title() string       { return i.name }
func (i resourceItem) Description() string { return i.count }
func (i resourceItem) FilterValue() string { return i.name }

type screen int

const (
	dashboardScreen screen = iota
	computeScreen
)

type Model struct {
	project    string
	list       list.Model
	compute    computeModel
	computeSvc *gcp.ComputeService
	screen     screen
	quitting   bool
	width      int
	height     int
}

func New(project string, computeSvc *gcp.ComputeService) Model {
	items := []list.Item{
		resourceItem{"Compute Engine", "Explore virtual machines"},
		resourceItem{"GKE", "Explore Kubernetes clusters"},
		resourceItem{"Disks", "Explore persistent disks"},
		resourceItem{"Cloud Storage", "Explore buckets"},
		resourceItem{"Cost", "Cost and utilization intelligence"},
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "G9S — GCP Resource & Cost Explorer"

	return Model{
		project:    project,
		list:       l,
		compute:    newComputeModel(0, 0),
		computeSvc: computeSvc,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.screen == computeScreen {
		if _, ok := msg.(backMsg); ok {
			m.screen = dashboardScreen
			return m, nil
		}
		var cmd tea.Cmd
		m.compute, cmd = m.compute.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		if msg.String() == "enter" {
			if item, ok := m.list.SelectedItem().(resourceItem); ok && item.name == "Compute Engine" {
				m.screen = computeScreen
				m.compute = newComputeModel(m.width, m.height-4)
				return m, m.loadVMs()
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-4)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) loadVMs() tea.Cmd {
	return func() tea.Msg {
		vms, err := m.computeSvc.ListVMs(context.Background())
		return vmListMsg{vms: vms, err: err}
	}
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.screen == computeScreen {
		return m.compute.View()
	}

	header := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Project: %s", m.project))
	footer := lipgloss.NewStyle().Faint(true).Render("↑↓ navigate  / search  enter open  q quit")
	return header + "\n\n" + m.list.View() + "\n" + footer + "\n"
}
