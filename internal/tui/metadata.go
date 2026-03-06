package tui

import (
	"fmt"
	"strings"

	"github.com/txperl/snipfly/internal/snippet"

	"charm.land/lipgloss/v2"
)

// metadataHeight returns the number of terminal lines the metadata area
// occupies (annotation lines + 1 separator line). Returns 0 when s is nil.
func metadataHeight(s *snippet.Snippet) int {
	if s == nil {
		return 0
	}
	n := 1 // @name is always shown
	if s.Desc != "" {
		n++
	}
	if s.Type != "" && s.Type != snippet.TypeOneshot {
		n++
	}
	if s.Dir != "" {
		n++
	}
	n += len(s.Env)
	if s.Interpreter != "" {
		n++
	}
	if s.PTY {
		n++
	}
	return n + 1 // +1 for separator line
}

// renderMetadata builds the metadata header string for the output panel.
// It renders @key: value lines in gray with a subtle separator at the bottom.
func renderMetadata(s *snippet.Snippet, width int) string {
	if s == nil {
		return ""
	}

	style := lipgloss.NewStyle().Foreground(ColorGray)
	var lines []string

	// @name — always shown
	lines = append(lines, style.Render(fmt.Sprintf("@name: %s", s.Name)))

	if s.Desc != "" {
		lines = append(lines, style.Render(fmt.Sprintf("@desc: %s", s.Desc)))
	}
	if s.Type != "" && s.Type != snippet.TypeOneshot {
		lines = append(lines, style.Render(fmt.Sprintf("@type: %s", s.Type)))
	}
	if s.Dir != "" {
		lines = append(lines, style.Render(fmt.Sprintf("@dir: %s", s.Dir)))
	}
	for _, e := range s.Env {
		lines = append(lines, style.Render(fmt.Sprintf("@env: %s", e)))
	}
	if s.Interpreter != "" {
		lines = append(lines, style.Render(fmt.Sprintf("@interpreter: %s", s.Interpreter)))
	}
	if s.PTY {
		lines = append(lines, style.Render("@pty: true"))
	}

	// Separator
	sep := lipgloss.NewStyle().Foreground(ColorSubtle).Render(strings.Repeat("─", width))
	lines = append(lines, sep)

	return strings.Join(lines, "\n")
}
