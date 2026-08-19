package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func renderDashboard(project string, resources list.Model, width, height int) string {
	innerWidth := width - 8
	if innerWidth < 70 {
		innerWidth = 70
	}

	rows := []struct {
		resource string
		count    string
		state    string
		action   string
	}{
		{"Compute Engine", "—", "READY", "Enter"},
		{"GKE Clusters", "—", "READY", "Enter"},
		{"Persistent Disks", "—", "READY", "Enter"},
		{"Cloud Storage", "—", "READY", "Enter"},
		{"Cost Intelligence", "—", "READY", "Enter"},
	}

	selected := resources.Index()
	if selected < 0 || selected >= len(rows) {
		selected = 0
	}

	title := renderTopBar(project, "GCP Resource Explorer")
	context := lipgloss.JoinHorizontal(lipgloss.Left,
		statusOKStyle.Render("● CONNECTED"),
		"  ",
		mutedStyle.Render("GCP"),
		"  ",
		projectStyle.Render(project),
	)

	headers := lipgloss.JoinHorizontal(lipgloss.Left,
		accentStyle.Width(24).Render("RESOURCE"),
		accentStyle.Width(12).Align(lipgloss.Right).Render("COUNT"),
		"  ",
		accentStyle.Width(14).Render("STATE"),
		accentStyle.Width(12).Align(lipgloss.Right).Render("ACTION"),
	)

	lines := []string{headers, mutedStyle.Render(fmt.Sprintf("%s", "────────────────────────────────────────────────────────────────────────"))}
	for i, row := range rows {
		prefix := "  "
		nameStyle := appTitleStyle
		if i == selected {
			prefix = "▶ "
			nameStyle = accentStyle
		}
		lines = append(lines,
			lipgloss.JoinHorizontal(lipgloss.Left,
				nameStyle.Width(24).Render(prefix+row.resource),
				mutedStyle.Width(12).Align(lipgloss.Right).Render(row.count),
				"  ",
				statusOKStyle.Width(14).Render(row.state),
				mutedStyle.Width(12).Align(lipgloss.Right).Render(row.action),
			),
		)
	}

	table := panelStyle.Width(innerWidth).Render(lipgloss.JoinVertical(lipgloss.Left,
		sectionStyle.Render("RESOURCES"),
		"",
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	))

	summary := panelStyle.Width(innerWidth).Render(lipgloss.JoinHorizontal(lipgloss.Left,
		renderMetric("SERVICES", "5", appTitleStyle),
		"    ",
		renderMetric("MODE", "READ-ONLY", statusOKStyle),
		"    ",
		renderMetric("NAVIGATION", "K9S-STYLE", accentStyle),
	))

	footer := renderFooter("↑↓ navigate  •  enter open  •  / filter  •  : commands  •  q quit")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		context,
		"",
		table,
		"",
		summary,
		"",
		footer,
	)

	return renderFrame(width, height, content)
}

func dashboardKey(msg tea.KeyMsg) bool {
	return msg.String() == "enter"
}
