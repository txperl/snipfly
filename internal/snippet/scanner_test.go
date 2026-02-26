package snippet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanDir(t *testing.T) {
	dir := t.TempDir()

	// Root level file
	os.WriteFile(filepath.Join(dir, "hello.sh"), []byte("#!/usr/bin/env bash\n# @name: hello\necho hi\n"), 0o644)

	// Group subdirectory
	os.MkdirAll(filepath.Join(dir, "tools"), 0o755)
	os.WriteFile(filepath.Join(dir, "tools", "build.py"), []byte("# @name: build\n# @desc: Build tool\nprint('building')\n"), 0o644)

	// Hidden file (should be skipped)
	os.WriteFile(filepath.Join(dir, ".hidden.sh"), []byte("#!/bin/bash\necho hidden\n"), 0o644)

	// Hidden directory (should be skipped)
	os.MkdirAll(filepath.Join(dir, ".config"), 0o755)
	os.WriteFile(filepath.Join(dir, ".config", "test.sh"), []byte("#!/bin/bash\necho test\n"), 0o644)

	// Unknown extension without shebang (should be skipped)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not a snippet\n"), 0o644)

	snippets, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(snippets) != 2 {
		t.Fatalf("expected 2 snippets, got %d", len(snippets))
	}

	// Root group should come first
	if snippets[0].Group != "" {
		t.Errorf("first snippet should be root group, got group=%q", snippets[0].Group)
	}
	if snippets[0].Name != "hello" {
		t.Errorf("first snippet name = %q, want %q", snippets[0].Name, "hello")
	}

	if snippets[1].Group != "tools" {
		t.Errorf("second snippet should be in tools group, got group=%q", snippets[1].Group)
	}
	if snippets[1].Name != "build" {
		t.Errorf("second snippet name = %q, want %q", snippets[1].Name, "build")
	}
}

func TestScanDirDefaults(t *testing.T) {
	dir := t.TempDir()

	// File with no annotations — should get defaults
	os.WriteFile(filepath.Join(dir, "my-script.sh"), []byte("#!/usr/bin/env bash\necho hi\n"), 0o644)

	snippets, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(snippets) != 1 {
		t.Fatalf("expected 1 snippet, got %d", len(snippets))
	}

	s := snippets[0]
	if s.Name != "my-script" {
		t.Errorf("default name = %q, want %q", s.Name, "my-script")
	}
	if s.Type != TypeOneshot {
		t.Errorf("default type = %q, want %q", s.Type, TypeOneshot)
	}
	if s.Dir != dir {
		t.Errorf("default dir = %q, want %q", s.Dir, dir)
	}
	if s.Interpreter != "bash" {
		t.Errorf("interpreter = %q, want %q", s.Interpreter, "bash")
	}
}

func TestScanDirSortOrder(t *testing.T) {
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "beta"), 0o755)
	os.MkdirAll(filepath.Join(dir, "alpha"), 0o755)

	os.WriteFile(filepath.Join(dir, "root.sh"), []byte("#!/bin/bash\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "beta", "z.sh"), []byte("#!/bin/bash\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "beta", "a.sh"), []byte("#!/bin/bash\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "alpha", "x.sh"), []byte("#!/bin/bash\n"), 0o644)

	snippets, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(snippets) != 4 {
		t.Fatalf("expected 4 snippets, got %d", len(snippets))
	}

	expected := []struct{ group, name string }{
		{"", "root"},
		{"alpha", "x"},
		{"beta", "a"},
		{"beta", "z"},
	}

	for i, e := range expected {
		if snippets[i].Group != e.group || snippets[i].Name != e.name {
			t.Errorf("snippets[%d] = {group=%q, name=%q}, want {group=%q, name=%q}",
				i, snippets[i].Group, snippets[i].Name, e.group, e.name)
		}
	}
}

func TestScanDirNoExtensionWithShebang(t *testing.T) {
	dir := t.TempDir()

	// File with no extension but has shebang
	os.WriteFile(filepath.Join(dir, "myscript"), []byte("#!/usr/bin/env python3\nprint('hi')\n"), 0o644)

	snippets, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(snippets) != 1 {
		t.Fatalf("expected 1 snippet, got %d", len(snippets))
	}

	if snippets[0].Interpreter != "python3" {
		t.Errorf("interpreter = %q, want python3", snippets[0].Interpreter)
	}
}

func TestScanDirRelativePathReturnsAbsoluteFilePath(t *testing.T) {
	workspace := t.TempDir()
	examplesDir := filepath.Join(workspace, "examples", "tools")
	if err := os.MkdirAll(examplesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	scriptPath := filepath.Join(examplesDir, "hello.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash\necho hello\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	if err := os.Chdir(workspace); err != nil {
		t.Fatal(err)
	}

	snippets, err := ScanDir("./examples")
	if err != nil {
		t.Fatal(err)
	}
	if len(snippets) != 1 {
		t.Fatalf("expected 1 snippet, got %d", len(snippets))
	}

	if !filepath.IsAbs(snippets[0].FilePath) {
		t.Fatalf("expected absolute file path, got %q", snippets[0].FilePath)
	}

	gotFilePath, err := filepath.EvalSymlinks(snippets[0].FilePath)
	if err != nil {
		t.Fatal(err)
	}
	wantFilePath, err := filepath.EvalSymlinks(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if gotFilePath != wantFilePath {
		t.Fatalf("file path = %q, want %q", gotFilePath, wantFilePath)
	}

	gotDir, err := filepath.EvalSymlinks(snippets[0].Dir)
	if err != nil {
		t.Fatal(err)
	}
	wantDir, err := filepath.EvalSymlinks(filepath.Dir(scriptPath))
	if err != nil {
		t.Fatal(err)
	}
	if gotDir != wantDir {
		t.Fatalf("dir = %q, want %q", gotDir, wantDir)
	}
}
