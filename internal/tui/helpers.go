package tui

import (
	"context"
	"fmt"
	"github.com/0x0BSoD/rtop/internal/stats"
	"github.com/0x0BSoD/rtop/pkg/logger"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/sync/semaphore"
	"time"
)

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

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func resizeBars(bars map[string]progress.Model, width int) {
	for _, bar := range bars {
		bar.Width = width - 2*2 - 4
		if bar.Width > 80 {
			bar.Width = 80
		}
	}
}

func InitFsTable(m *Model) {
	columns := []table.Column{
		{Title: "Device", Width: 30},
		{Title: "Mount", Width: 15},
		{Title: "Free", Width: 10},
		{Title: "Total", Width: 10},
	}

	rows := make([]table.Row, 0, len(m.SshFetcher.Stats.FSInfos))
	for _, d := range m.SshFetcher.Stats.FSInfos {
		rows = append(rows, table.Row{
			d.Device, d.MountPoint, formatBytes(d.Free), formatBytes(d.Used + d.Free),
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.NoColor{}).
		Background(lipgloss.NoColor{}).
		Bold(false)
	t.SetStyles(s)

	m.fsTable = t
}

func NewModel(updateInterval time.Duration, fetcher *stats.SshFetcher) Model {
	// Initialize progress bars
	progressBars := make(map[string]progress.Model, 10)
	progressBars["total"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["system"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["user"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["idle"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["irq"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["softIrq"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["iowait"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["guest"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["nice"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["steal"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))

	m := Model{
		SshFetcher:       fetcher,
		UpdateInterval:   updateInterval,
		Bars:             progressBars,
		SessionSemaphore: semaphore.NewWeighted(int64(2)),
	}
	m.folded = make(map[int]bool)

	// Initial fetch
	msg := fetchStatsCmd(m.SessionSemaphore, m.SshFetcher)()
	result, ok := msg.(statsMsg)
	if !ok {
		logger.Error("Failed to parse stats message")
	}
	m.Stats = result.Stats

	InitFsTable(&m)
	InitNetTable(&m)

	return m
}

func InitNetTable(m *Model) {
	columns := []table.Column{
		{Title: "Name", Width: 15},
		{Title: "IPv4", Width: 17},
		{Title: "IPv6", Width: 30},
		{Title: "RX", Width: 10},
		{Title: "TX", Width: 10},
	}

	rows := make([]table.Row, 0, len(m.SshFetcher.Stats.FSInfos))
	for n, d := range m.SshFetcher.Stats.NetIntf {
		rows = append(rows, table.Row{
			n, d.IPv4, d.IPv6, formatBytes(d.Rx), formatBytes(d.Tx),
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.NoColor{}).
		Background(lipgloss.NoColor{}).
		Bold(false)
	t.SetStyles(s)

	m.netTable = t
}

func drawFocused(myViewport, focused Viewports, width int, data string) string {
	if myViewport == focused {
		return bigGroupStyleFocused.Width(width).Render(data)
	}
	return data
}

//func drawFocused(myViewport, focused Viewports, width int, data string) string {
//	if myViewport == focused {
//		bigGroupStyleFocused = lipgloss.NewStyle().
//			Border(lipgloss.RoundedBorder()).
//			BorderForeground(lipgloss.Color("63")).
//			Padding(1).
//			Width(width)
//		return bigGroupStyleFocused.Render(data)
//	}
//	return data
//}

func fetchStatsCmd(sessionSemaphore *semaphore.Weighted, fetcher *stats.SshFetcher) tea.Cmd {
	logger.Debug("Fetching stats")
	ctx := context.Background()
	// Try to acquire semaphore - this will block if limit is reached
	if err := sessionSemaphore.Acquire(ctx, 1); err != nil {
		logger.Error("failed to acquire session semaphore: %v", err)
		return func() tea.Msg {
			return statsMsg{
				Stats: fetcher.Stats,
				Errs:  nil,
			}
		}
	}
	// Release semaphore when function exits (success or error)
	defer sessionSemaphore.Release(1)

	return func() tea.Msg {
		err := fetcher.GetAllStats()
		return statsMsg{
			Stats: fetcher.Stats,
			Errs:  err,
		}
	}
}
