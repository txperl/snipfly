package snippet

import (
	"bufio"
	"os"
	"strings"
)

// Annotations holds parsed metadata from a snippet file's header comments.
type Annotations struct {
	Name        string
	Desc        string
	Type        string
	Dir         string
	Env         []string
	Interpreter string
	PTY         string
}

// ParseAnnotations reads the header comments of a file and extracts @key: value annotations.
// Supports # and // comment prefixes. Stops at the first non-comment, non-empty line.
func ParseAnnotations(filePath string) (*Annotations, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	a := &Annotations{}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip shebang
		if strings.HasPrefix(line, "#!") {
			continue
		}

		// Skip empty lines
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Try to extract comment content
		content, ok := extractCommentContent(trimmed)
		if !ok {
			// First non-comment, non-empty line — stop
			break
		}

		// Look for @key: value
		parseAnnotationLine(content, a)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return a, nil
}

// extractCommentContent strips a comment prefix (# or //) and returns the inner text.
func extractCommentContent(line string) (string, bool) {
	if after, ok := strings.CutPrefix(line, "//"); ok {
		return strings.TrimSpace(after), true
	}
	if after, ok := strings.CutPrefix(line, "#"); ok {
		return strings.TrimSpace(after), true
	}
	return "", false
}

// parseAnnotationLine parses a single "@key: value" from comment content.
func parseAnnotationLine(content string, a *Annotations) {
	if !strings.HasPrefix(content, "@") {
		return
	}

	// Split only at the first colon to preserve colons in values
	idx := strings.Index(content, ":")
	if idx < 0 {
		return
	}

	key := strings.TrimSpace(content[1:idx]) // strip leading @
	value := strings.TrimSpace(content[idx+1:])

	switch strings.ToLower(key) {
	case "name":
		a.Name = value
	case "desc":
		a.Desc = value
	case "type":
		a.Type = value
	case "dir":
		a.Dir = value
	case "env":
		a.Env = append(a.Env, value)
	case "interpreter":
		a.Interpreter = value
	case "pty":
		a.PTY = value
	}
}
