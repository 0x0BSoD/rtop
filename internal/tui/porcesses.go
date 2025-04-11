package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"strings"
)

// Max length to show when a line is folded
const foldedLength = 60

func (m Model) viewProcesses(availableWidth int) string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Top Processes by CPU"))
	sb.WriteString("\n")

	for i := 0; i < len(m.Stats.Procs); i++ {
		process := m.Stats.Procs[i]

		rowStyle := unfocusedStyle
		cursor := " "
		if m.focused == ViewportProcesses && m.cursor == i {
			rowStyle = selectedStyle
			cursor = cursorStyle.Render(">")
		}

		isFolded := !m.folded[i]
		indicator := indicatorStyle.Render("[+] ")
		if !isFolded {
			indicator = indicatorStyle.Render("[-] ")
		}

		pidStr := fmt.Sprintf("%-6s", process.Pid)
		cpuStr := fmt.Sprintf("%5s%%", process.CpuPercent)
		memStr := fmt.Sprintf("%5s%%", process.MemPercent)

		cpuStr = valueStyle.Render(cpuStr)
		memStr = valueStyle.Render(memStr)

		statsLine := lipgloss.JoinHorizontal(lipgloss.Top,
			cursor,
			processNumStyle.Render(fmt.Sprintf("%d)", i+1)),
			" ",
			indicator,
			keywordStyle.Render("PID: "), valueStyle.Render(pidStr), " ",
			keywordStyle.Render("CPU: "), cpuStr, " ",
			keywordStyle.Render("MEM: "), memStr,
		)

		sb.WriteString(rowStyle.Render(statsLine))
		sb.WriteString("\n")

		indentationWidth := 10
		indentation := strings.Repeat(" ", indentationWidth)
		availableWidth := availableWidth - indentationWidth
		if availableWidth < 10 {
			availableWidth = 10
		}

		var commandLines []string
		displayCmd := process.Command

		if isFolded {
			if len(displayCmd) > foldedLength {
				displayCmd = displayCmd[:foldedLength] + "..."
			}
			commandLines = append(commandLines, indentation+displayCmd)
		} else {
			wrappedCmd := wordwrap.String(displayCmd, availableWidth-10)
			lines := strings.Split(wrappedCmd, "\n")
			for _, line := range lines {
				commandLines = append(commandLines, indentation+line)
			}
		}

		for _, line := range commandLines {
			renderedLine := rowStyle.Render(line)
			sb.WriteString(renderedLine)
			sb.WriteString("\n")
		}

		if !isFolded || len(m.Stats.Procs) < 5 {
			sb.WriteString("\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}
