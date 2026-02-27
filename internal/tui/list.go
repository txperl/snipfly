package tui

import (
	"fmt"
	"strings"

	"github.com/txperl/GoSnippet/internal/snippet"

	"charm.land/lipgloss/v2"
)

// ListItemKind distinguishes group headers from snippet entries.
type ListItemKind int

const (
	ListItemGroup   ListItemKind = iota // Unselectable group header
	ListItemSnippet                     // Selectable snippet entry
)

// ListItem represents a single row in the list panel.
type ListItem struct {
	Kind    ListItemKind
	Group   string
	Snippet *snippet.Snippet
}

// ListModel manages the snippet list panel with cursor navigation and virtual scrolling.
type ListModel struct {
	items    []ListItem
	cursor   int
	offset   int
	width    int
	height   int
	focused  bool
	snippets []snippet.Snippet
	// getState retrieves the process state for a snippet by path.
	getState func(path string) snippet.ProcessState
}

// NewListModel creates a list panel from the given snippets.
func NewListModel(snippets []snippet.Snippet, getState func(string) snippet.ProcessState) ListModel {
	m := ListModel{
		snippets: snippets,
		getState: getState,
	}
	m.buildItems()
	// Position cursor on the first selectable item
	m.cursor = m.firstSelectableIndex()
	return m
}

// buildItems flattens snippets into a list of group headers and snippet entries.
func (m *ListModel) buildItems() {
	m.items = nil
	currentGroup := "\x00" // sentinel
	for i := range m.snippets {
		s := &m.snippets[i]
		if s.Group != currentGroup {
			currentGroup = s.Group
			// Root group (empty string) doesn't get a header
			if s.Group != "" {
				m.items = append(m.items, ListItem{
					Kind:  ListItemGroup,
					Group: s.Group,
				})
			}
		}
		m.items = append(m.items, ListItem{
			Kind:    ListItemSnippet,
			Snippet: s,
		})
	}
}

// firstSelectableIndex returns the index of the first snippet item, or 0.
func (m *ListModel) firstSelectableIndex() int {
	for i, item := range m.items {
		if item.Kind == ListItemSnippet {
			return i
		}
	}
	return 0
}

// SelectedSnippet returns the currently selected snippet, or nil.
func (m *ListModel) SelectedSnippet() *snippet.Snippet {
	if m.cursor >= 0 && m.cursor < len(m.items) && m.items[m.cursor].Kind == ListItemSnippet {
		return m.items[m.cursor].Snippet
	}
	return nil
}

// MoveUp moves the cursor up, skipping group headers. Does not wrap.
func (m *ListModel) MoveUp() {
	for i := m.cursor - 1; i >= 0; i-- {
		if m.items[i].Kind == ListItemSnippet {
			m.cursor = i
			m.ensureVisible()
			return
		}
	}
}

// MoveDown moves the cursor down, skipping group headers. Does not wrap.
func (m *ListModel) MoveDown() {
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.items[i].Kind == ListItemSnippet {
			m.cursor = i
			m.ensureVisible()
			return
		}
	}
}

// ensureVisible adjusts the scroll offset so the cursor is within the visible window.
func (m *ListModel) ensureVisible() {
	visible := m.height - 1 // -1 for title bar
	if visible <= 0 {
		return
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
}

// UpdateSnippets rebuilds the list with new snippets, preserving cursor position by FilePath.
func (m *ListModel) UpdateSnippets(snippets []snippet.Snippet) {
	// Remember selected snippet path
	var selectedPath string
	if sel := m.SelectedSnippet(); sel != nil {
		selectedPath = sel.FilePath
	}

	m.snippets = snippets
	m.buildItems()

	// Try to restore cursor position
	restored := false
	if selectedPath != "" {
		for i, item := range m.items {
			if item.Kind == ListItemSnippet && item.Snippet.FilePath == selectedPath {
				m.cursor = i
				restored = true
				break
			}
		}
	}
	if !restored {
		m.cursor = m.firstSelectableIndex()
	}
	m.ensureVisible()
}

// SetSize updates the panel dimensions.
func (m *ListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.ensureVisible()
}

// View renders the visible portion of the list.
func (m *ListModel) View() string {
	var b strings.Builder

	// Render title bar
	if m.focused {
		b.WriteString(StylePanelTitle.Width(m.width - 2).Render("◆ GoSnippet"))
	} else {
		b.WriteString(StylePanelTitleBlurred.Width(m.width - 2).Render("  GoSnippet"))
	}

	if len(m.items) == 0 {
		b.WriteByte('\n')
		b.WriteString(StyleSnippetItem.Render("No snippets found"))
		return StyleListPanel.Width(m.width).Height(m.height).Render(b.String())
	}

	visibleRows := m.height - 1 // -1 for title bar
	end := m.offset + visibleRows
	if end > len(m.items) {
		end = len(m.items)
	}

	for i := m.offset; i < end; i++ {
		b.WriteByte('\n')
		item := m.items[i]
		switch item.Kind {
		case ListItemGroup:
			b.WriteString(m.renderGroupHeader(item.Group))
		case ListItemSnippet:
			b.WriteString(m.renderSnippetItem(item.Snippet, i == m.cursor))
		}
	}

	// Pad remaining lines if the list is shorter than the viewport
	rendered := end - m.offset
	for i := rendered; i < visibleRows; i++ {
		b.WriteByte('\n')
	}

	return StyleListPanel.Width(m.width).Height(m.height).Render(b.String())
}

func (m *ListModel) renderGroupHeader(group string) string {
	return StyleGroupHeader.Width(m.width - 2).Render(fmt.Sprintf("── %s ──", group))
}

func (m *ListModel) renderSnippetItem(s *snippet.Snippet, selected bool) string {
	var icon string
	if s.Error != "" {
		if selected {
			// Plain character so the outer selected style applies uniformly.
			icon = IconFailed
		} else {
			icon = lipgloss.NewStyle().Foreground(ColorRed).Render(IconFailed)
		}
	} else {
		state := snippet.StateIdle
		if m.getState != nil {
			state = m.getState(s.FilePath)
		}
		if selected {
			// Use unstyled icon to avoid ANSI resets breaking the
			// selected row's purple background.
			icon = StateIconChar(state)
		} else {
			icon = StateIcon(state)
		}
	}
	content := fmt.Sprintf("%s %s", icon, s.Name)

	style := StyleSnippetItem
	if selected {
		style = StyleSnippetSelected
	}
	return style.Width(m.width - 2).Render(content)
}

// Render renders the list panel with proper lipgloss styling.
func (m *ListModel) Render() string {
	return lipgloss.NewStyle().Width(m.width).Height(m.height).Render(m.View())
}
