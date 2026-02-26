package snippet

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var extensionMap = map[string]string{
	".sh":  "bash",
	".ts":  "npx tsx",
	".js":  "node",
	".py":  "python3",
	".go":  "go run",
	".rb":  "ruby",
	".lua": "lua",
}

// KnownExtensions returns a set of file extensions that have known interpreters.
func KnownExtensions() map[string]bool {
	m := make(map[string]bool, len(extensionMap))
	for ext := range extensionMap {
		m[ext] = true
	}
	return m
}

// ResolveInterpreter determines the interpreter command and args for a snippet file.
// Priority: annotation > shebang > file extension.
func ResolveInterpreter(filePath, annotationInterpreter string) (string, []string, error) {
	if annotationInterpreter != "" {
		cmd, args := parseInterpreterString(annotationInterpreter)
		return cmd, args, nil
	}

	cmd, args := parseShebang(filePath)
	if cmd != "" {
		return cmd, args, nil
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	if interp, ok := extensionMap[ext]; ok {
		cmd, args := parseInterpreterString(interp)
		return cmd, args, nil
	}

	return "", nil, fmt.Errorf("cannot determine interpreter for %s", filePath)
}

// HasShebang checks whether a file starts with a shebang line.
func HasShebang(filePath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		return strings.HasPrefix(scanner.Text(), "#!")
	}
	return false
}

// parseShebang reads the first line of a file and extracts the interpreter
// from a shebang directive. Handles both "#!/usr/bin/env bash" and "#!/bin/bash" forms.
func parseShebang(filePath string) (string, []string) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return "", nil
	}
	line := scanner.Text()
	if !strings.HasPrefix(line, "#!") {
		return "", nil
	}

	line = strings.TrimPrefix(line, "#!")
	line = strings.TrimSpace(line)

	parts := strings.Fields(line)
	if len(parts) == 0 {
		return "", nil
	}

	// #!/usr/bin/env bash  → cmd=bash, args from rest
	if parts[0] == "/usr/bin/env" && len(parts) > 1 {
		return parts[1], parts[2:]
	}

	// #!/bin/bash → cmd=bash (basename)
	cmd := filepath.Base(parts[0])
	return cmd, parts[1:]
}

// parseInterpreterString splits a string like "npx tsx" into command and args.
func parseInterpreterString(s string) (string, []string) {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}
