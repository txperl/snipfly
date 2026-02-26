package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// viewWithConfirm renders the exit confirmation dialog centered on the screen.
func (m *AppModel) viewWithConfirm() string {
	dialog := StyleConfirmBox.Render(
		"Running processes detected!\n\nQuit and stop all? (y/n)",
	)
	return lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars(" "),
	)
}

// handleConfirmKey processes key presses while the confirm dialog is shown.
func (m AppModel) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m, func() tea.Msg {
			m.runner.StopAll()
			return tea.Quit()
		}
	case "n", "N", "escape":
		m.showConfirm = false
		return m, nil
	}
	return m, nil
}
