package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
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

// BubbleTea helpers

func resizeBars(bars map[string]progress.Model, width int) {
	for _, bar := range bars {
		bar.Width = width - 2*2 - 4
		if bar.Width > 80 {
			bar.Width = 80
		}
	}
}

func initFsTable(m *guiModel) {
	columns := []table.Column{
		{Title: "Device", Width: 30},
		{Title: "Mount", Width: 15},
		{Title: "Free", Width: 10},
		{Title: "Total", Width: 10},
	}

	rows := make([]table.Row, 0, len(m.stats.FSInfos))
	for _, d := range m.stats.FSInfos {
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

func initNetTable(m *guiModel) {
	columns := []table.Column{
		{Title: "Name", Width: 15},
		{Title: "IPv4", Width: 17},
		{Title: "IPv6", Width: 30},
		{Title: "RX", Width: 10},
		{Title: "TX", Width: 10},
	}

	rows := make([]table.Row, 0, len(m.stats.FSInfos))
	for n, d := range m.stats.NetIntf {
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
