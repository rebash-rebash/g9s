package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type resourceItem struct {
	name  string
	count string
}

func (i resourceItem) Title() string       { return i.name }
func (i resourceItem) Description() string { return i.count }
func (i resourceItem) FilterValue() string { return i.name }

type Model struct {
	project  string
	list     list.Model
	quitting bool
}

func New(project string) Model {
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

	return Model{project: project, list: l}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-4)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	header := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Project: %s", m.project))
	footer := lipgloss.NewStyle().Faint(true).Render("↑↓ navigate  / search  enter open  q quit")
	return header + "\n\n" + m.list.View() + "\n" + footer + "\n"
}
