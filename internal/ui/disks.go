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
	if i.disk.Attached { status = "ATTACHED" }
	return fmt.Sprintf("%s  •  %d GiB  •  %s  •  %s", i.disk.Zone, i.disk.SizeGB, i.disk.Type, status)
}
func (i diskItem) FilterValue() string { return i.disk.Name }

type disksModel struct { list list.Model; loading bool; err error; disks []model.Disk }

func newDisksModel(width, height int) disksModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	l.Title = "PERSISTENT DISKS"
	l.Styles.Title = sectionStyle
	l.Styles.HelpStyle = mutedStyle
	l.Styles.PaginationStyle = mutedStyle
	return disksModel{list: l, loading: true}
}

func (m disksModel) Update(msg tea.Msg) (disksModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg: m.list.SetSize(msg.Width, msg.Height-6)
	case diskListMsg: m.loading = false; m.disks = msg.disks; m.err = msg.err
	case tea.KeyMsg:
		if msg.String() == "esc" { return m, func() tea.Msg { return backMsg{} } }
	}
	var cmd tea.Cmd; m.list, cmd = m.list.Update(msg); return m, cmd
}

func (m disksModel) View() string {
	if m.loading { return lipgloss.JoinVertical(lipgloss.Left, renderTopBar("", "Persistent Disks"), "", mutedStyle.Render("Loading disk inventory...")) }
	if m.err != nil { return lipgloss.JoinVertical(lipgloss.Left, renderTopBar("", "Persistent Disks"), "", statusBadStyle.Render("Error: "+m.err.Error())) }

	var monthly, unattachedMonthly float64
	var unattached int
	items := make([]list.Item, 0, len(m.disks))
	for _, d := range m.disks {
		e := cost.EstimateDisk(d)
		if e.Available { monthly += e.MonthlyUSD; if !d.Attached { unattachedMonthly += e.MonthlyUSD } }
		if !d.Attached { unattached++ }
		items = append(items, diskItem{disk: d})
	}
	m.list.SetItems(items)

	header := renderTopBar("", "Persistent Disks")
	cards := lipgloss.JoinHorizontal(lipgloss.Top,
		panelStyle.Width(20).Render(renderMetric("DISKS", fmt.Sprintf("%d", len(m.disks)), appTitleStyle)), "  ",
		panelStyle.Width(20).Render(renderMetric("UNATTACHED", fmt.Sprintf("%d", unattached), statusWarnStyle)), "  ",
		panelStyle.Width(28).Render(renderMetric("MONTHLY", fmt.Sprintf("$%.2f", monthly), accentStyle)), "  ",
		panelStyle.Width(32).Render(renderMetric("WASTE CANDIDATE", fmt.Sprintf("$%.2f/mo", unattachedMonthly), statusBadStyle)),
	)
	footer := renderFooter("↑↓ navigate  •  / search  •  esc back")
	return lipgloss.JoinVertical(lipgloss.Left, header, "", cards, "", m.list.View(), "", footer)
}

func loadDisks(svc *gcp.DiskService) tea.Cmd {
	return func() tea.Msg { disks, err := svc.ListDisks(context.Background()); return diskListMsg{disks: disks, err: err} }
}

type diskListMsg struct { disks []model.Disk; err error }
