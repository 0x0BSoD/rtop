package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"

	"github.com/0x0BSoD/rtop/internal/stats"
)

func (m Model) getCurrentLevelCgroup() []*stats.Cgroup {
	if len(m.path) == 0 {
		return m.Stats.Cgroups
	}

	currentParent := m.path[len(m.path)-1]
	return currentParent.Childs
}

func (m Model) getSelectedCgroup() *stats.Cgroup {
	currentCgroup := m.getCurrentLevelCgroup()
	if len(currentCgroup) == 0 || m.cursor >= len(currentCgroup) {
		return nil
	}
	return currentCgroup[m.cursor]
}

func (m Model) viewCgroups(availableWidth int) string {
	var sb strings.Builder
	//viewportWidth := m.viewport.Width

	pathDisplay := ""
	if len(m.path) > 0 {
		pathSegments := make([]string, len(m.path))
		for i, cgroup := range m.path {
			pathSegments[i] = cgroup.Path
		}
		pathDisplay = "Path: " + strings.Join(pathSegments, separatorStyle.Render(" > "))
	} else {
		pathDisplay = "Cgroup Root"
	}
	sb.WriteString(pathStyle.Render(lipgloss.PlaceHorizontal(availableWidth, lipgloss.Left, titleStyle.Render(pathDisplay))))
	sb.WriteString("\n\n")

	currentLevelCgroups := m.getCurrentLevelCgroup()
	if len(currentLevelCgroups) == 0 {
		if len(m.path) > 0 {
			sb.WriteString(inactiveStyle.Render("  (No children in this cgroup)"))
		} else {
			sb.WriteString(errorStyle.Render("No cgroup data available at root"))
		}
		sb.WriteString("\n")
	}

	for i, c := range currentLevelCgroups {
		rowStyle := inactiveStyle
		cursor := " "
		if m.focused == ViewportCgroups && i == m.cursor {
			rowStyle = activeStyle
			cursor = ">"
		}

		indicator := "  "
		if len(c.Childs) > 0 {
			indicator = listIndicatorStyle.Render("▸ ")
		}

		cgroupListItem := fmt.Sprintf("%s%s%s", cursor, indicator, c.Path)

		renderedLine := lipgloss.NewStyle().MaxWidth(availableWidth).Render(rowStyle.Render(cgroupListItem))
		sb.WriteString(renderedLine)
		sb.WriteString("\n")
	}

	sb.WriteString(borderStyle.Width(availableWidth).Render(""))

	selectedCgroup := m.getSelectedCgroup()
	if m.focused == ViewportCgroups && selectedCgroup != nil {
		detailsContent := ""
		memLimitStr := formatBytes(uint64(selectedCgroup.MemoryUsageLimit))
		if selectedCgroup.MemoryUsageLimit == 0 {
			memLimitStr = "Unlimited"
		}

		detailsContent += lipgloss.JoinHorizontal(lipgloss.Left,
			detailsLabelStyle.Render("CPU Time:"),
			detailsValueStyle.Render(fmt.Sprintf("%.2f s", selectedCgroup.CpuUsage)),
		) + "\n"

		detailsContent += lipgloss.JoinHorizontal(lipgloss.Left,
			detailsLabelStyle.Render("Memory:"),
			detailsValueStyle.Render(fmt.Sprintf("%s / %s", formatBytes(uint64(selectedCgroup.MemoryUsageCurrent)), memLimitStr)),
		) + "\n"

		if selectedCgroup.MemoryUsageLimit > 0 && selectedCgroup.MemoryUsageLimit >= selectedCgroup.MemoryUsageCurrent {
			memPercent := float64(selectedCgroup.MemoryUsageCurrent) / float64(selectedCgroup.MemoryUsageLimit)
			barWidth := 20
			filledWidth := int(memPercent * float64(barWidth))
			emptyWidth := barWidth - filledWidth
			memBar := "[" + strings.Repeat("#", filledWidth) + strings.Repeat("-", emptyWidth) + "]"
			detailsContent += lipgloss.JoinHorizontal(lipgloss.Left,
				detailsLabelStyle.Render("Mem Usage:"),
				detailsValueStyle.Render(memBar),
			) + "\n"
		}

		detailsContent += lipgloss.JoinHorizontal(lipgloss.Left,
			detailsLabelStyle.Render("IO Read:"),
			detailsValueStyle.Render(formatBytes(uint64(selectedCgroup.IoReadBytes))),
		) + "\n"

		detailsContent += lipgloss.JoinHorizontal(lipgloss.Left,
			detailsLabelStyle.Render("IO Write:"),
			detailsValueStyle.Render(formatBytes(uint64(selectedCgroup.IoWriteBytes))),
		) + "\n"

		detailsContent += lipgloss.JoinHorizontal(lipgloss.Left,
			detailsLabelStyle.Render("Children:"),
			detailsValueStyle.Render(fmt.Sprintf("%d", len(selectedCgroup.Childs))),
		)

		detailsTitle := fmt.Sprintf("Details: %s", selectedCgroup.Path)
		sb.WriteString(detailsBoxStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				titleStyle.Render(detailsTitle),
				detailsContent,
			),
		))

	} else {
		sb.WriteString(inactiveStyle.Render("Select a cgroup to view details."))
	}

	return sb.String()
}
