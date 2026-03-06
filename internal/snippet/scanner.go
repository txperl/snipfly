package snippet

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ScanDir scans a directory for snippet files. Root-level files have group="",
// one-level subdirectories become group names.
func ScanDir(dir string) ([]Snippet, error) {
	dir, err := expandTilde(dir)
	if err != nil {
		return nil, err
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	var snippets []Snippet

	// Scan root-level files
	rootEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range rootEntries {
		if isHidden(entry.Name()) {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			// Scan one-level subdirectory as a group
			group := entry.Name()
			subEntries, err := os.ReadDir(fullPath)
			if err != nil {
				continue
			}
			for _, subEntry := range subEntries {
				if subEntry.IsDir() || isHidden(subEntry.Name()) {
					continue
				}
				subPath := filepath.Join(fullPath, subEntry.Name())
				if isSnippetFile(subPath) {
					if s, err := buildSnippet(subPath, group); err == nil {
						snippets = append(snippets, s)
					}
				}
			}
		} else {
			if isSnippetFile(fullPath) {
				if s, err := buildSnippet(fullPath, ""); err == nil {
					snippets = append(snippets, s)
				}
			}
		}
	}

	sortSnippets(snippets)
	return snippets, nil
}

func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

func isSnippetFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if KnownExtensions()[ext] {
		return true
	}
	// No extension — check for shebang
	if ext == "" {
		return HasShebang(path)
	}
	return false
}

func buildSnippet(filePath, group string) (Snippet, error) {
	ann, err := ParseAnnotations(filePath)
	if err != nil {
		return Snippet{}, err
	}

	s := Snippet{
		FilePath: filePath,
		Group:    group,
		State:    StateIdle,
	}

	interp, interpArgs, err := ResolveInterpreter(filePath, ann.Interpreter)
	if err != nil {
		s.Error = err.Error()
	} else {
		s.Interpreter = interp
		s.InterpreterArgs = interpArgs
	}

	// Apply annotations with defaults
	s.Name = ann.Name
	if s.Name == "" {
		base := filepath.Base(filePath)
		ext := filepath.Ext(base)
		s.Name = strings.TrimSuffix(base, ext)
	}

	s.Desc = ann.Desc

	s.Type = ann.Type
	if s.Type == "" {
		s.Type = TypeOneshot
	}

	s.Dir = ann.Dir
	if s.Dir != "" {
		expanded, err := expandTilde(s.Dir)
		if err == nil {
			s.Dir = expanded
		}
	}
	if s.Dir == "" {
		s.Dir = filepath.Dir(filePath)
	}

	s.Env = ann.Env
	s.PTY = strings.EqualFold(ann.PTY, "true")

	return s, nil
}

func expandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, path[1:]), nil
}

func sortSnippets(snippets []Snippet) {
	sort.Slice(snippets, func(i, j int) bool {
		// Root group ("") sorts first
		gi, gj := snippets[i].Group, snippets[j].Group
		if gi != gj {
			if gi == "" {
				return true
			}
			if gj == "" {
				return false
			}
			return gi < gj
		}
		return snippets[i].Name < snippets[j].Name
	})
}
