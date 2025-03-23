package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewMetrics() string {
	var outHeader string
	var outCpu string
	var outMem string

	horizontalPadding := 1
	verticalPadding := 1

	contentWidth := m.width - (horizontalPadding * 2)
	contentHeight := m.height - (verticalPadding * 2)

	// Header ---
	outHeader += fmt.Sprintf("%s %s ", keywordStyle.Render("HostName"), m.SshFetcher.Stats.Hostname)
	outHeader += fmt.Sprintf("%s %s %s %s ", keywordStyle.Render("Load Average"), m.SshFetcher.Stats.Load1, m.SshFetcher.Stats.Load5, m.SshFetcher.Stats.Load10)
	outHeader += fmt.Sprintf("%s %s\n", keywordStyle.Render("Uptime"), formatDurationWithDays(m.SshFetcher.Stats.Uptime))
	outHeader += fmt.Sprintf("%s %s running of %s total\n\n", keywordStyle.Render("Processes"), m.SshFetcher.Stats.RunningProcs, m.SshFetcher.Stats.TotalProcs)

	// CPU ---
	// system
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "System")),
		m.Bars["system"].ViewAs(float64(m.SshFetcher.Stats.CPU.System)/100.0),
	)
	// user
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "User")),
		m.Bars["user"].ViewAs(float64(m.SshFetcher.Stats.CPU.User)/100.0),
	)
	// irq
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Irq")),
		m.Bars["irq"].ViewAs(float64(m.SshFetcher.Stats.CPU.Irq)/100.0),
	)
	// softIrq
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "SoftIrq")),
		m.Bars["softIrq"].ViewAs(float64(m.SshFetcher.Stats.CPU.SoftIrq)/100.0),
	)
	// iowait
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Iowait")),
		m.Bars["iowait"].ViewAs(float64(m.SshFetcher.Stats.CPU.Iowait)/100.0),
	)
	// guest
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Guest")),
		m.Bars["guest"].ViewAs(float64(m.SshFetcher.Stats.CPU.Guest)/100.0),
	)
	// nice
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Nice")),
		m.Bars["nice"].ViewAs(float64(m.SshFetcher.Stats.CPU.Nice)/100.0),
	)
	// steal
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Steal")),
		m.Bars["idle"].ViewAs(float64(m.SshFetcher.Stats.CPU.Steal)/100.0),
	)
	// idle
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Idle")),
		m.Bars["idle"].ViewAs(float64(m.SshFetcher.Stats.CPU.Idle)/100.0),
	)
	// total
	cpuLoad := 100.0 - m.SshFetcher.Stats.CPU.Idle
	outCpu += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Total")),
		m.Bars["total"].ViewAs(float64(cpuLoad)/100.0),
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
		formatBytes(m.SshFetcher.Stats.MemFree))
	// used
	outMem += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Used")),
		formatBytes(m.SshFetcher.Stats.MemTotal-m.SshFetcher.Stats.MemFree))
	// buffers
	outMem += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Buffers")),
		formatBytes(m.SshFetcher.Stats.MemBuffers))
	// cached
	outMem += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Cached")),
		formatBytes(m.SshFetcher.Stats.MemCached))
	// swap
	outMem += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Swap")),
		formatBytes(m.SshFetcher.Stats.SwapTotal-m.SshFetcher.Stats.SwapFree))
	// total
	outMem += fmt.Sprintf("%s %6s\n",
		labelStyle.Render(fmt.Sprintf("%-8s", "Total")),
		formatBytes(m.SshFetcher.Stats.MemTotal))

	memGroup := groupStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			"Memory",
			outMem,
		),
	)

	bigGroup := bigGroupStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Left,
			cpuGroup,
			memGroup,
		),
	)

	return lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Padding(verticalPadding, horizontalPadding).
		Render(outHeader +
			bigGroup +
			"\n\n" +
			m.fsTable.View() +
			"\n\n" +
			m.netTable.View(),
		)
}
