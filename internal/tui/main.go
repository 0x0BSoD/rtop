package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/sync/semaphore"
	"time"

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
	ViewportProcesses Viewports = iota
	ViewportCgroups
	viewportCount
)

type Model struct {
	UpdateInterval   time.Duration
	SshFetcher       *stats.SshFetcher
	Stats            *stats.Stats
	SessionSemaphore *semaphore.Weighted

	width      int
	height     int
	Bars       map[string]progress.Model
	fsTable    table.Model
	netTable   table.Model
	viewport   viewport.Model
	path       []*stats.Cgroup
	selected   *stats.Cgroup
	cgroupView bool
	cursor     int
	folded     map[int]bool
	focused    Viewports
}

// Init TODO: there a second Init for stats, can be avoided
func (m Model) Init() tea.Cmd {
	initialFetch := fetchStatsCmd(m.SessionSemaphore, m.SshFetcher)

	startTicker := tea.Tick(m.UpdateInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})

	return tea.Batch(
		initialFetch,
		startTicker,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

	switch msg := msg.(type) {
	case statsMsg:
		m.Stats = msg.Stats
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
			itemsLen := len(currentCgroup) - 1
			if m.focused == ViewportProcesses {
				itemsLen = len(m.SshFetcher.Stats.Procs) - 1
			}
			if m.cursor < itemsLen {
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
			m.cursor = 0
			m.focused++
			m.focused %= viewportCount
		case key.Matches(msg, keys.Space):
			m.folded[m.cursor] = !m.folded[m.cursor]
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 0
		footerHeight := 4

		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerHeight - footerHeight

		resizeBars(m.Bars, m.width)
	case tickMsg:
		if !m.cgroupView {
			cmds = append(cmds, fetchStatsCmd(m.SessionSemaphore, m.SshFetcher))

			m.fsTable, cmd = m.fsTable.Update(m)
			cmds = append(cmds, cmd)

			m.netTable, cmd = m.netTable.Update(m)
			cmds = append(cmds, cmd)
		}

		cmds = append(cmds,
			tea.Tick(m.UpdateInterval, func(t time.Time) tea.Msg {
				return tickMsg(t)
			}),
		)
	}

	if m.cgroupView {
		borderWidth := 2
		paddingWidth := 2 // Padding(1) on each side
		focusWrapperWidth := m.width - 2
		// Calculate the width *inside* the focus border+padding
		innerWidth := focusWrapperWidth - borderWidth - paddingWidth // m.width - 6

		processesContent := m.viewProcesses(innerWidth)
		cgroupsContent := m.viewCgroups(innerWidth)

		m.viewport.SetContent(
			lipgloss.JoinVertical(lipgloss.Left,
				drawFocused("Top Processes by CPU", ViewportProcesses, m.focused, focusWrapperWidth, processesContent),
				drawFocused("Cgroups", ViewportCgroups, m.focused, focusWrapperWidth, cgroupsContent),
			),
		)
	} else {
		m.viewport.SetContent(m.viewMetrics())
	}
	m.selected = m.getSelectedCgroup()

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	header := ""
	help := helpStyle.Render("↑/↓: Navigate  ←/→: Back/Enter  c: Toggle tab: Toggle focus  q: Quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		m.viewport.View(),
		help,
	)
}
