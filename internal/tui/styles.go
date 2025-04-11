package tui

import "github.com/charmbracelet/lipgloss"

var (
	detailsValueStyle = lipgloss.NewStyle()
	unfocusedStyle    = detailsValueStyle

	pathStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).Padding(0, 1)

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	listIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("69"))

	detailsBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("197"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("238")).
			MarginBottom(1).MarginTop(1)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230"))

	highCpuStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("237"))

	indicatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))

	processNumStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).Width(3).Align(lipgloss.Right)

	titleStyle = lipgloss.NewStyle().
			Bold(true).MarginBottom(1)

	keywordStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
	detailsLabelStyle = keywordStyle.Width(10)

	groupStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1).
			Width(60)

	bigGroupStyle = lipgloss.NewStyle().
			Padding(0).
			Width(130)
	bigGroupStyleFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("168"))

	inactiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("231"))

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("35")).
			Background(lipgloss.Color("231"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("242"))

	indentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("249"))
)
