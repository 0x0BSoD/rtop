package tui

import (
	"fmt"
	"github.com/muesli/reflow/wordwrap"
	"strings"
)

func (m Model) viewProcesses() string {
	var sb strings.Builder

	// TODO: now all this stuff is statics, next will be dynamic
	cursor := " "          // No cursor
	indicator := ""        // Right now without it
	prefix := cursor + " " // Base prefix with cursor and space
	linePrefix := prefix + indicator
	paddingWidth := len(linePrefix) // Calculate width needed for prefix on first line
	indentation := strings.Repeat(" ", paddingWidth)

	availableWidth := m.width - paddingWidth
	if availableWidth < 1 {
		availableWidth = 1
	}

	// TODO: For now I'll add proc info to cGroups view
	sb.WriteString(titleStyle.Render(" Top 10 Processes by CPU usage"))
	sb.WriteString("\n\n")

	if m.stats == nil {
		return ""
	}

	for i := 0; i < len(m.stats.Procs); i++ {
		process := m.stats.Procs[i]
		var renderedLines []string

		// Calculate available width for the log text itself
		// Subtract prefix width. Ensure it's at least 1.
		availableWidth := m.width - paddingWidth
		if availableWidth < 1 {
			availableWidth = 1
		}

		wrappedCmd := wordwrap.String(process.Command, availableWidth)
		// Split the potentially multi-line wrapped string
		lines := strings.Split(wrappedCmd, "\n")

		// Add the first line with the full prefix
		if len(lines) > 0 {
			renderedLines = append(renderedLines, linePrefix+lines[0])
		}

		// Add subsequent wrapped lines with indentation
		if len(lines) > 1 {
			for _, line := range lines[1:] {
				// Only add indentation if the line is not empty
				if strings.TrimSpace(line) != "" {
					renderedLines = append(renderedLines, indentation+line)
				} else {
					renderedLines = append(renderedLines, "") // Keep empty lines if wordwrap created them
				}
			}
		}

		sb.WriteString(fmt.Sprintf("%d) %s %s, %s %s, %s %s, %s >\n%s\n",
			i+1,
			keywordStyle.Render("PID"),
			process.Pid,
			keywordStyle.Render("CPU"),
			process.CpuPercent,
			keywordStyle.Render("MEM"),
			process.MemPercent,
			keywordStyle.Render("CMD"),
			strings.Join(renderedLines, "\n")+"\n"))
	}

	return sb.String()
}
