package tui

import "github.com/charmbracelet/lipgloss"

var (
	keywordStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("204")).
			Background(lipgloss.Color("235"))

	groupStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1).
			Width(60)

	bigGroupStyle = lipgloss.NewStyle().
			Padding(0).
			Width(130)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("168"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	inactiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5"))

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#25A065")).
			Background(lipgloss.Color("#FFFDF5"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	indentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B2B2B2"))
)
