package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// OutputMsg signals that a snippet has produced new output.
// The actual content is read from the RingBuffer in batch.
type OutputMsg struct {
	SnippetPath string
}

// ProcessExitedMsg signals that a snippet process has exited.
type ProcessExitedMsg struct {
	SnippetPath string
	ExitCode    int
	Err         error
}

// ExecFinishedMsg signals that an interactive process has exited.
type ExecFinishedMsg struct {
	SnippetPath string
	Err         error
}

// OutputThrottleTickMsg is sent when the 50ms throttle timer expires.
type OutputThrottleTickMsg struct{}

// throttleTickCmd returns a Cmd that sends OutputThrottleTickMsg after 50ms.
func throttleTickCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(_ time.Time) tea.Msg {
		return OutputThrottleTickMsg{}
	})
}
