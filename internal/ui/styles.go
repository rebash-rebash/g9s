package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorAccent = lipgloss.Color("#7C5CFC")
	colorCyan   = lipgloss.Color("#5CC8FF")
	colorGreen  = lipgloss.Color("#56D364")
	colorYellow = lipgloss.Color("#E3B341")
	colorRed    = lipgloss.Color("#FF6B6B")
	colorMuted  = lipgloss.Color("#7D8590")
	colorBorder = lipgloss.Color("#30363D")

	appTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F0F6FC"))
	projectStyle  = lipgloss.NewStyle().Foreground(colorMuted)
	sectionStyle  = lipgloss.NewStyle().Bold(true).Foreground(colorCyan)
	accentStyle   = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	mutedStyle    = lipgloss.NewStyle().Foreground(colorMuted)

	panelStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorBorder).Padding(0, 1)

	statusOKStyle = lipgloss.NewStyle().Bold(true).Foreground(colorGreen)
	statusWarnStyle = lipgloss.NewStyle().Bold(true).Foreground(colorYellow)
	statusBadStyle = lipgloss.NewStyle().Bold(true).Foreground(colorRed)
)

func statusStyle(status string) lipgloss.Style {
	switch status {
	case "RUNNING", "ATTACHED", "ACTIVE": return statusOKStyle
	case "TERMINATED", "UNATTACHED", "IDLE": return statusBadStyle
	default: return statusWarnStyle
	}
}

func renderTopBar(project, page string) string {
	brand := lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Render("G9S")
	separator := mutedStyle.Render("/")
	pageText := appTitleStyle.Render(page)
	if project == "" {
		return lipgloss.JoinHorizontal(lipgloss.Center, brand, " ", separator, " ", pageText)
	}
	projectText := projectStyle.Render("project: " + project)
	return lipgloss.JoinHorizontal(lipgloss.Center, brand, " ", separator, " ", pageText, "  ", projectText)
}

func renderFooter(text string) string { return mutedStyle.Render(text) }

func renderMetric(label, value string, valueStyle lipgloss.Style) string {
	return lipgloss.JoinVertical(lipgloss.Left, mutedStyle.Render(label), valueStyle.Render(value))
}
