package snippet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAnnotations(t *testing.T) {
	dir := t.TempDir()

	content := `#!/usr/bin/env bash
# @name: my-script
# @desc: A test script with colons: in the value
# @type: service
# @dir: ~/projects
# @env: FOO=bar
# @env: BAZ=qux
# @interpreter: python3

echo "hello"
`
	file := filepath.Join(dir, "test.sh")
	os.WriteFile(file, []byte(content), 0o644)

	ann, err := ParseAnnotations(file)
	if err != nil {
		t.Fatal(err)
	}

	if ann.Name != "my-script" {
		t.Errorf("Name = %q, want %q", ann.Name, "my-script")
	}
	if ann.Desc != "A test script with colons: in the value" {
		t.Errorf("Desc = %q", ann.Desc)
	}
	if ann.Type != "service" {
		t.Errorf("Type = %q, want %q", ann.Type, "service")
	}
	if ann.Dir != "~/projects" {
		t.Errorf("Dir = %q, want %q", ann.Dir, "~/projects")
	}
	if len(ann.Env) != 2 || ann.Env[0] != "FOO=bar" || ann.Env[1] != "BAZ=qux" {
		t.Errorf("Env = %v", ann.Env)
	}
	if ann.Interpreter != "python3" {
		t.Errorf("Interpreter = %q", ann.Interpreter)
	}
}

func TestParseAnnotationsSlashComments(t *testing.T) {
	dir := t.TempDir()

	content := `// @name: go-snippet
// @desc: A Go snippet
// @type: oneshot

package main
`
	file := filepath.Join(dir, "test.go")
	os.WriteFile(file, []byte(content), 0o644)

	ann, err := ParseAnnotations(file)
	if err != nil {
		t.Fatal(err)
	}

	if ann.Name != "go-snippet" {
		t.Errorf("Name = %q, want %q", ann.Name, "go-snippet")
	}
	if ann.Desc != "A Go snippet" {
		t.Errorf("Desc = %q", ann.Desc)
	}
}

func TestParseAnnotationsStopsAtCode(t *testing.T) {
	dir := t.TempDir()

	content := `# @name: before
echo "code line"
# @name: after
`
	file := filepath.Join(dir, "test.sh")
	os.WriteFile(file, []byte(content), 0o644)

	ann, err := ParseAnnotations(file)
	if err != nil {
		t.Fatal(err)
	}

	if ann.Name != "before" {
		t.Errorf("Name = %q, should not read past code lines", ann.Name)
	}
}

func TestParseAnnotationsUnknownKeysIgnored(t *testing.T) {
	dir := t.TempDir()

	content := `# @name: test
# @unknown: should be ignored
# @desc: works
`
	file := filepath.Join(dir, "test.sh")
	os.WriteFile(file, []byte(content), 0o644)

	ann, err := ParseAnnotations(file)
	if err != nil {
		t.Fatal(err)
	}

	if ann.Name != "test" || ann.Desc != "works" {
		t.Errorf("unexpected parsing result: %+v", ann)
	}
}
