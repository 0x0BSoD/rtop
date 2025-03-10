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
	groupStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(1).Width(60)
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("168"))
	valueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
)

type tickMsg time.Time

type guiModel struct {
	updateInterval time.Duration
	client         *ssh.Client
	stats          Stats
	width          int
	height         int
	bars           map[string]progress.Model
}

func resizeBars(bars map[string]progress.Model, width int) {
	for _, bar := range bars {
		bar.Width = width - 2*2 - 4
		if bar.Width > 80 {
			bar.Width = 80
		}
	}
}

func formatDurationWithDays(d time.Duration) string {
	days := d / (24 * time.Hour)
	d %= 24 * time.Hour

	h := d / time.Hour
	m := (d % time.Hour) / time.Minute
	s := (d % time.Minute) / time.Second

	if days > 0 {
		return fmt.Sprintf("%dd %02d:%02d:%02d", days, h, m, s)
	}
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.2f %cb", float64(bytes)/float64(div), "KMGTPE"[exp])
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
		resizeBars(m.bars, m.width)
	}

	return m, nil
}

func (m guiModel) View() string {
	var outHeader string
	var outCpu string
	var outMem string

	horizontalPadding := 4
	verticalPadding := 2

	contentWidth := m.width - (horizontalPadding * 2)
	contentHeight := m.height - (verticalPadding * 2)

	// Header ---
	outHeader += fmt.Sprintf("%s %s ", keywordStyle.Render("HostName"), m.stats.Hostname)
	outHeader += fmt.Sprintf("%s %s %s %s ", keywordStyle.Render("Load Average"), m.stats.Load1, m.stats.Load5, m.stats.Load10)
	outHeader += fmt.Sprintf("%s %s\n", keywordStyle.Render("Uptime"), formatDurationWithDays(m.stats.Uptime))
	outHeader += fmt.Sprintf("%s %s running of %s total\n\n", keywordStyle.Render("Processes"), m.stats.RunningProcs, m.stats.TotalProcs)

	// CPU ---
	// system
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "System")),
		m.bars["system"].ViewAs(float64(m.stats.CPU.System)/100.0),
	)
	// user
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "User")),
		m.bars["user"].ViewAs(float64(m.stats.CPU.User)/100.0),
	)
	// irq
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Irq")),
		m.bars["irq"].ViewAs(float64(m.stats.CPU.Irq)/100.0),
	)
	// softIrq
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "SoftIrq")),
		m.bars["softIrq"].ViewAs(float64(m.stats.CPU.SoftIrq)/100.0),
	)
	// iowait
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Iowait")),
		m.bars["iowait"].ViewAs(float64(m.stats.CPU.Iowait)/100.0),
	)
	// guest
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Guest")),
		m.bars["guest"].ViewAs(float64(m.stats.CPU.Guest)/100.0),
	)
	// nice
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Nice")),
		m.bars["nice"].ViewAs(float64(m.stats.CPU.Nice)/100.0),
	)
	// steal
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Steal")),
		m.bars["idle"].ViewAs(float64(m.stats.CPU.Steal)/100.0),
	)
	// idle
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Idle")),
		m.bars["idle"].ViewAs(float64(m.stats.CPU.Idle)/100.0),
	)
	// total
	cpuLoad := 100.0 - m.stats.CPU.Idle
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Total")),
		m.bars["total"].ViewAs(float64(cpuLoad)/100.0),
	)

	cpuGroup := groupStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			"CPU",
			outCpu,
		),
	)

	// Memory ---
	// free
	outMem += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Free")),
		formatBytes(m.stats.MemFree))
	// used
	outMem += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Used")),
		formatBytes(m.stats.MemTotal-m.stats.MemFree))
	// buffers
	outMem += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Buffers")),
		formatBytes(m.stats.MemBuffers))
	// cached
	outMem += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Cached")),
		formatBytes(m.stats.MemCached))
	// swap
	outMem += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Swap")),
		formatBytes(m.stats.SwapTotal-m.stats.SwapFree))
	// total
	outMem += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Total")),
		formatBytes(m.stats.MemTotal))

	memGroup := groupStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			"Memory",
			outMem,
		),
	)

	out := outHeader +
		cpuGroup +
		"\n" +
		memGroup +
		fmt.Sprintf("\n\n%s", helpStyle.Render("Press q or ESC to quit"))

	mainSection := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Padding(verticalPadding, horizontalPadding).
		Render(out)

	return mainSection
}
