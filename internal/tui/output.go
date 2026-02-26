package tui

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

// OutputModel wraps a viewport with auto-scroll (sticky bottom) behavior.
type OutputModel struct {
	viewport viewport.Model
}

// NewOutputModel creates a new output panel.
func NewOutputModel() OutputModel {
	return OutputModel{
		viewport: viewport.New(),
	}
}

// SetSize updates the output panel dimensions.
func (m *OutputModel) SetSize(width, height int) {
	m.viewport.SetWidth(width)
	m.viewport.SetHeight(height)
}

// SetContent updates the viewport content with sticky-bottom auto-scroll.
// If the viewport was at the bottom before the update, it auto-scrolls to the new bottom.
func (m *OutputModel) SetContent(content string) {
	wasAtBottom := m.viewport.AtBottom() || m.viewport.TotalLineCount() == 0
	m.viewport.SetContent(content)
	if wasAtBottom {
		m.viewport.GotoBottom()
	}
}

// View renders the viewport content.
func (m *OutputModel) View() string {
	return m.viewport.View()
}

// Update proxies messages to the underlying viewport.
func (m OutputModel) Update(msg tea.Msg) (OutputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}
