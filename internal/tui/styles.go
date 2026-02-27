package tui

import (
	"github.com/txperl/GoSnippet/internal/snippet"

	"charm.land/lipgloss/v2"
)

// Status icons
const (
	IconRunning = "●"
	IconCrashed = "✗"
	IconDone    = "✓"
	IconStopped = "■"
	IconIdle    = " "
	IconFailed  = "✗"
	IconExited  = "✓"
)

// Colors
var (
	ColorGreen     = lipgloss.Color("#04B575")
	ColorRed       = lipgloss.Color("#FF4672")
	ColorGray      = lipgloss.Color("#626262")
	ColorBlue      = lipgloss.Color("#7D9BFF")
	ColorSubtle    = lipgloss.Color("#383838")
	ColorHighlight = lipgloss.Color("#874BFD")
	ColorWhite     = lipgloss.Color("#FAFAFA")
)

// Styles
var (
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorSubtle).
			PaddingLeft(1).
			PaddingRight(1)

	StyleListPanel = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(ColorSubtle)

	StyleGroupHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorBlue).
				PaddingLeft(1)

	StyleSnippetItem = lipgloss.NewStyle().
				PaddingLeft(2)

	StyleSnippetSelected = lipgloss.NewStyle().
				PaddingLeft(2).
				Background(ColorHighlight).
				Foreground(ColorWhite)

	StyleStatusBar = lipgloss.NewStyle().
			Foreground(ColorGray)

	StyleConfirmBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorHighlight).
			Padding(1, 2).
			Align(lipgloss.Center)

	StylePanelTitle = lipgloss.NewStyle().
			Bold(true).
			Background(ColorHighlight).
			Foreground(ColorWhite).
			PaddingLeft(1)

	StylePanelTitleBlurred = lipgloss.NewStyle().
				Foreground(ColorGray).
				PaddingLeft(1)
)

// StateIcon returns the colored icon string for a given process state.
func StateIcon(state snippet.ProcessState) string {
	switch state {
	case snippet.StateRunning:
		return lipgloss.NewStyle().Foreground(ColorGreen).Render(IconRunning)
	case snippet.StateCrashed:
		return lipgloss.NewStyle().Foreground(ColorRed).Render(IconCrashed)
	case snippet.StateDone:
		return lipgloss.NewStyle().Foreground(ColorGray).Render(IconDone)
	case snippet.StateFailed:
		return lipgloss.NewStyle().Foreground(ColorRed).Render(IconFailed)
	case snippet.StateStopped:
		return lipgloss.NewStyle().Foreground(ColorGray).Render(IconStopped)
	case snippet.StateExited:
		return lipgloss.NewStyle().Foreground(ColorGray).Render(IconExited)
	default:
		return IconIdle
	}
}

// StateIconChar returns the raw icon character for a given process state
// without any ANSI color styling. Used when the caller applies its own
// outer style (e.g. selected-row highlight) that must not be interrupted
// by embedded ANSI resets. Mirrors StateIcon – keep both in sync.
func StateIconChar(state snippet.ProcessState) string {
	switch state {
	case snippet.StateRunning:
		return IconRunning
	case snippet.StateCrashed:
		return IconCrashed
	case snippet.StateDone:
		return IconDone
	case snippet.StateFailed:
		return IconFailed
	case snippet.StateStopped:
		return IconStopped
	case snippet.StateExited:
		return IconExited
	default:
		return IconIdle
	}
}
