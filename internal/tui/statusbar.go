package tui

import (
	"fmt"
	"strings"

	"github.com/txperl/GoSnippet/internal/snippet"

	"charm.land/lipgloss/v2"
)

// renderStatusBar renders the bottom status bar with context-aware key hints
// and the selected snippet's state/PID/exit code.
func (m *AppModel) renderStatusBar() string {
	// Left side: key hints based on focus
	var keys []string
	if m.focus == FocusList {
		keys = append(keys, "↑/↓:navigate", "Enter:run", "s:stop", "r:restart", "Tab:switch")
	} else {
		keys = append(keys, "↑/↓:scroll", "Tab:switch")
	}
	keys = append(keys, "q:quit")
	left := strings.Join(keys, "  ")

	// Right side: selected snippet state info
	right := m.snippetStatusInfo()

	// Layout: left-aligned keys, right-aligned status info
	gap := m.termWidth - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right

	return StyleStatusBar.Width(m.termWidth).Render(line)
}

// snippetStatusInfo returns the status string for the selected snippet.
func (m *AppModel) snippetStatusInfo() string {
	if m.selectedPath == "" {
		return ""
	}

	state := m.runner.GetState(m.selectedPath)
	icon := StateIcon(state)

	switch state {
	case snippet.StateRunning:
		proc := m.runner.GetProcess(m.selectedPath)
		if proc != nil {
			pid := proc.PID()
			if pid > 0 {
				return fmt.Sprintf("%s %s PID:%d", icon, state, pid)
			}
		}
		return fmt.Sprintf("%s %s", icon, state)

	case snippet.StateCrashed, snippet.StateFailed:
		proc := m.runner.GetProcess(m.selectedPath)
		if proc != nil {
			return fmt.Sprintf("%s %s (exit:%d)", icon, state, proc.ExitCode())
		}
		return fmt.Sprintf("%s %s", icon, state)

	case snippet.StateIdle:
		return ""

	default:
		return fmt.Sprintf("%s %s", icon, state)
	}
}
