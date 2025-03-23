package tui

import (
	"fmt"
	"strings"

	"github.com/0x0BSoD/rtop/internal/stats"
)

func (m Model) getCurrentLevelCgroup() []*stats.Cgroup {
	if len(m.path) == 0 {
		return m.SshFetcher.Stats.Cgroups
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

func (m Model) viewCgroups() string {
	var sb strings.Builder
	cgroup := m.getCurrentLevelCgroup()

	// Show current path
	if len(m.path) > 0 {
		path := make([]string, len(m.path))
		for i, cgroup := range m.path {
			path[i] = cgroup.Path
		}
		sb.WriteString(titleStyle.Render(" Path: " + strings.Join(path, " > ") + " "))
		sb.WriteString("\n\n")
	} else {
		sb.WriteString(titleStyle.Render(" Root "))
		sb.WriteString("\n\n")
	}

	for i, c := range cgroup {
		prefix := "  "
		style := inactiveStyle
		if i == m.cursor {
			prefix = "> "
			style = activeStyle
		}

		CgroupInfo := fmt.Sprintf("%s%s", prefix, c.Path)
		sb.WriteString(style.Render(CgroupInfo))
		sb.WriteString("\n")
	}

	// Show details of selected Cgroup
	sb.WriteString("\n")
	selectedCgroup := m.getSelectedCgroup()
	if selectedCgroup != nil {
		memLimit := formatBytes(uint64(selectedCgroup.MemoryUsageLimit))
		if selectedCgroup.MemoryUsageLimit == 0 {
			memLimit = "∞"
		}
		sb.WriteString(fmt.Sprintf("CPU: %.2f seconds\n", selectedCgroup.CpuUsage))
		sb.WriteString(fmt.Sprintf("Memory: %s / %s\n", formatBytes(uint64(selectedCgroup.MemoryUsageCurrent)), memLimit))
		sb.WriteString(fmt.Sprintf("IO: Read %s Write %s\n", formatBytes(uint64(selectedCgroup.IoReadBytes)), formatBytes(uint64(selectedCgroup.IoWriteBytes))))
		sb.WriteString(fmt.Sprintf("Children: %d\n", len(selectedCgroup.Childs)))
	}

	return sb.String()
}
