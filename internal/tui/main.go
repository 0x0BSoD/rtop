package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/0x0BSoD/rtop/internal/stats"
)

type tickMsg time.Time

type statsMsg struct {
	Stats *stats.Stats
	Errs  []error
}

type Viewports int

const (
	ViewportStats Viewports = iota
	ViewportCgroups
	ViewportProcesses
)

type Model struct {
	UpdateInterval time.Duration
	SshFetcher     *stats.SshFetcher
	stats          *stats.Stats
	width          int
	height         int
	Bars           map[string]progress.Model
	fsTable        table.Model
	netTable       table.Model
	viewport       viewport.Model
	path           []*stats.Cgroup
	selected       *stats.Cgroup
	cgroupView     bool
	cursor         int
	focused        Viewports
}

func (m Model) Init() tea.Cmd {
	if m.UpdateInterval == 0 {
		m.UpdateInterval = 1 * time.Second
	}

	m.viewport.SetContent(m.viewMetrics())
	tea.SetWindowTitle("rtop")

	return tea.Batch(
		fetchStatsCmd(m.SshFetcher),
		tea.Tick(m.UpdateInterval, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case statsMsg:
		m.stats = msg.Stats
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			currentCgroup := m.getCurrentLevelCgroup()
			if m.cursor < len(currentCgroup)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Left):
			if len(m.path) > 0 {
				lastIndex := len(m.path) - 1
				parentCgroup := m.path[lastIndex]
				m.path = m.path[:lastIndex]

				parentLevelCgroup := m.getCurrentLevelCgroup()
				for i, Cgroup := range parentLevelCgroup {
					if Cgroup == parentCgroup {
						m.cursor = i
						break
					}
				}
			}
		case key.Matches(msg, keys.Right):
			currentCgroup := m.getSelectedCgroup()
			if currentCgroup != nil && len(currentCgroup.Childs) > 0 {
				m.path = append(m.path, currentCgroup)
				currentCgroup.Open = true
				m.cursor = 0
			}
		case key.Matches(msg, keys.Toggle):
			m.cgroupView = !m.cgroupView
		case key.Matches(msg, keys.Tab):
			if m.focused == ViewportProcesses {
				m.focused = ViewportStats
				break
			}
			m.focused++
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 4

		resizeBars(m.Bars, m.width)

		cmds = append(cmds, viewport.Sync(m.viewport))
	case tickMsg:
		var cmd tea.Cmd
		m.fsTable, cmd = m.fsTable.Update(m)
		cmds = append(cmds, cmd)

		m.netTable, cmd = m.netTable.Update(m)
		cmds = append(cmds, cmd)

		if !m.cgroupView {
			cmds = append(cmds, fetchStatsCmd(m.SshFetcher))
		}

		cmds = append(cmds,
			tea.Tick(m.UpdateInterval, func(t time.Time) tea.Msg {
				return tickMsg(t)
			}),
		)
	}

	if m.cgroupView {
		m.viewport.SetContent(
			bigGroupStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					m.viewProcesses(),
					m.viewCgroups(),
				),
			),
		)
	} else {
		m.viewport.SetContent(m.viewMetrics())
	}
	m.selected = m.getSelectedCgroup()

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	help := helpStyle.Render("↑/↓: Navigate  ←/→: Back/Enter  c: Toggle tab: Toggle focus  q: Quit")

	return fmt.Sprintf("%s\n%s", m.viewport.View(), help)
}
