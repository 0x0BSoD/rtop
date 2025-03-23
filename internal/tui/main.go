package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/0x0BSoD/rtop/internal/stats"
)

type tickMsg time.Time

type Model struct {
	UpdateInterval time.Duration
	SshFetcher     *stats.SshFetcher
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
}

func (m Model) Init() tea.Cmd {
	if m.UpdateInterval == 0 {
		m.UpdateInterval = 1 * time.Second
	}

	m.viewport.SetContent(m.viewMetrics())
	tea.SetWindowTitle("rtop")

	return tea.Tick(m.UpdateInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
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
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 4

		resizeBars(m.Bars, m.width)

		cmds = append(cmds, viewport.Sync(m.viewport))

	case tickMsg:
		m.SshFetcher.GetAllStats()
		m.fsTable.Update(m)
		m.netTable.Update(m)

		return m, tea.Tick(m.UpdateInterval, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}

	if m.cgroupView {
		m.viewport.SetContent(m.viewCgroups())
	} else {
		m.viewport.SetContent(m.viewMetrics())
	}
	m.selected = m.getSelectedCgroup()

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	help := helpStyle.Render("↑/↓: Navigate  ←/→: Back/Enter  c: Toggle  q: Quit")

	return fmt.Sprintf("%s\n%s", m.viewport.View(), help)
}
