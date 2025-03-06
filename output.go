package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/crypto/ssh"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	keywordStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("204")).Background(lipgloss.Color("235"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type tickMsg time.Time

type guiModel struct {
	updateInterval time.Duration
	client         *ssh.Client
	stats          Stats
	width          int
	height         int
	prog           progress.Model
}

func (m guiModel) Init() tea.Cmd {
	if m.updateInterval == 0 {
		m.updateInterval = 1 * time.Second
	}
	tea.SetWindowTitle("rtop")
	return tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m guiModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case tickMsg:
		getAllStats(m.client, &m.stats)
		return m, tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.prog.Width = msg.Width - 2*2 - 4
		if m.prog.Width > 80 {
			m.prog.Width = 80
		}
	}

	return m, nil
}

func (m guiModel) View() string {
	var out string

	horizontalPadding := 4
	verticalPadding := 2

	contentWidth := m.width - (horizontalPadding * 2)
	contentHeight := m.height - (verticalPadding * 2)

	out += fmt.Sprintf("%s %s", keywordStyle.Render("HostName"), m.stats.Hostname)
	out += fmt.Sprintf(" %s %s %s\n\n", keywordStyle.Render("Uptime"), m.stats.Uptime, helpStyle.Render(" (hh:mm:ss)"))
	out += fmt.Sprintf("%s %s %s %s\n", keywordStyle.Render("Load Average"), m.stats.Load1, m.stats.Load5, m.stats.Load10)

	cpuLoad := 100.0 - m.stats.CPU.Idle
	out += fmt.Sprintf("%s ", keywordStyle.Render("CPU"))

	out += m.prog.ViewAs(float64(cpuLoad) / 100.0)

	styledContent := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Padding(verticalPadding, horizontalPadding).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Render(out)

	return styledContent
}
