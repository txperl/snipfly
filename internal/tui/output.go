package tui

import (
	"fmt"
	"regexp"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// SearchMode represents the current search state of the output panel.
type SearchMode int

const (
	SearchInactive   SearchMode = iota
	SearchTyping                // User is typing in the search input
	SearchNavigating            // User confirmed search, navigating with n/N
)

// OutputModel wraps a viewport with auto-scroll (sticky bottom) behavior
// and search/highlight functionality.
type OutputModel struct {
	viewport    viewport.Model
	searchInput textinput.Model
	searchMode  SearchMode
	searchQuery string // confirmed/active query used for highlighting
	matchCount  int
	matchIndex  int // 0-based, -1 if no matches
}

// NewOutputModel creates a new output panel.
func NewOutputModel() OutputModel {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 256

	vp := viewport.New()
	vp.HighlightStyle = StyleSearchMatch
	vp.SelectedHighlightStyle = StyleSearchMatchCurrent

	return OutputModel{
		viewport:    vp,
		searchInput: ti,
		matchIndex:  -1,
	}
}

// SetSize updates the output panel dimensions.
func (m *OutputModel) SetSize(width, height int) {
	m.viewport.SetWidth(width)
	m.viewport.SetHeight(height)
}

// SetContent updates the viewport content with sticky-bottom auto-scroll.
// Reapplies search highlights after content update (viewport.SetContent clears them).
func (m *OutputModel) SetContent(content string) {
	wasAtBottom := m.viewport.AtBottom() || m.viewport.TotalLineCount() == 0
	m.viewport.SetContent(content)
	if wasAtBottom {
		m.viewport.GotoBottom()
	}
	m.reapplyHighlights()
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

// UpdateSearchInput forwards a message to the textinput and returns any command.
func (m OutputModel) UpdateSearchInput(msg tea.Msg) (OutputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

// --- Search ---

// IsSearchActive returns true if search mode is not inactive.
func (m *OutputModel) IsSearchActive() bool {
	return m.searchMode != SearchInactive
}

// SearchBarHeight returns 1 when the search bar is visible, 0 otherwise.
func (m *OutputModel) SearchBarHeight() int {
	if m.searchMode != SearchInactive {
		return 1
	}
	return 0
}

// EnterSearch switches to typing mode and focuses the text input.
func (m *OutputModel) EnterSearch() tea.Cmd {
	m.searchMode = SearchTyping
	return m.searchInput.Focus()
}

// ExitSearch clears search state and highlights.
func (m *OutputModel) ExitSearch() {
	m.searchMode = SearchInactive
	m.searchQuery = ""
	m.searchInput.SetValue("")
	m.searchInput.Blur()
	m.matchCount = 0
	m.matchIndex = -1
	m.viewport.ClearHighlights()
}

// ConfirmSearch locks in the current input value and switches to navigation mode.
func (m *OutputModel) ConfirmSearch() {
	query := m.searchInput.Value()
	if query == "" {
		m.ExitSearch()
		return
	}
	m.searchMode = SearchNavigating
	m.searchQuery = query
	m.searchInput.Blur()
	m.reapplyHighlights()
}

// LiveUpdateHighlights updates highlights in real time as the user types.
func (m *OutputModel) LiveUpdateHighlights() {
	m.searchQuery = m.searchInput.Value()
	m.reapplyHighlights()
}

// NextMatch moves to the next match.
func (m *OutputModel) NextMatch() {
	if m.matchCount == 0 {
		return
	}
	m.viewport.HighlightNext()
	m.matchIndex = (m.matchIndex + 1) % m.matchCount
}

// PrevMatch moves to the previous match.
func (m *OutputModel) PrevMatch() {
	if m.matchCount == 0 {
		return
	}
	m.viewport.HighlightPrevious()
	m.matchIndex = (m.matchIndex - 1 + m.matchCount) % m.matchCount
}

// LastMatch jumps to the last match.
func (m *OutputModel) LastMatch() {
	if m.matchCount == 0 {
		return
	}
	// From current position, go backward once to reach the last match.
	// HighlightPrevious wraps, so from match 0 it goes to matchCount-1.
	for m.matchIndex != m.matchCount-1 {
		m.viewport.HighlightNext()
		m.matchIndex = (m.matchIndex + 1) % m.matchCount
	}
}

// SearchBarView renders the search input bar.
func (m *OutputModel) SearchBarView(width int) string {
	prompt := StyleSearchPrompt.Render("/")

	var countStr string
	if m.searchQuery != "" {
		if m.matchCount > 0 {
			countStr = StyleSearchCount.Render(fmt.Sprintf(" [%d/%d]", m.matchIndex+1, m.matchCount))
		} else {
			countStr = StyleSearchCount.Render(" [0/0]")
		}
	}

	inputWidth := width - lipgloss.Width(prompt) - lipgloss.Width(countStr) - 1
	if inputWidth < 1 {
		inputWidth = 1
	}
	m.searchInput.SetWidth(inputWidth)

	return lipgloss.JoinHorizontal(lipgloss.Left, prompt, m.searchInput.View(), countStr)
}

// reapplyHighlights finds all matches and sets highlights, preserving the
// current match position across content refreshes.
//
// viewport.SetContent() clears highlights, so this must be called after every
// content update. The trick: viewport.SetHighlights() always resets its internal
// highlight index via findNearestMatch(). To restore our saved position, we
// temporarily scroll to YOffset=0 so SetHighlights deterministically selects
// match 0, then navigate forward to the saved index, then restore the scroll.
func (m *OutputModel) reapplyHighlights() {
	if m.searchQuery == "" {
		m.viewport.ClearHighlights()
		m.matchCount = 0
		m.matchIndex = -1
		return
	}

	content := m.viewport.GetContent()
	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(m.searchQuery))
	if err != nil {
		m.viewport.ClearHighlights()
		m.matchCount = 0
		m.matchIndex = -1
		return
	}

	matches := re.FindAllStringIndex(content, -1)
	m.matchCount = len(matches)

	if m.matchCount == 0 {
		m.viewport.ClearHighlights()
		m.matchIndex = -1
		return
	}

	// Clamp saved index to valid range.
	targetIdx := m.matchIndex
	if targetIdx < 0 || targetIdx >= m.matchCount {
		targetIdx = 0
	}

	// Force SetHighlights to select match 0 by temporarily scrolling to top.
	savedYOffset := m.viewport.YOffset()
	m.viewport.SetYOffset(0)
	m.viewport.SetHighlights(matches)

	// Navigate from match 0 to target.
	for range targetIdx {
		m.viewport.HighlightNext()
	}
	m.matchIndex = targetIdx

	// Restore scroll position (showHighlight inside HighlightNext may have
	// scrolled to the target match — only restore if we had a valid saved offset).
	if m.searchMode == SearchNavigating {
		// In navigation mode, let the viewport stay at the match position
		// (showHighlight already scrolled there).
	} else {
		m.viewport.SetYOffset(savedYOffset)
	}
}
