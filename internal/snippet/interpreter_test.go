package snippet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseInterpreterString(t *testing.T) {
	tests := []struct {
		input    string
		wantCmd  string
		wantArgs []string
	}{
		{"bash", "bash", nil},
		{"npx tsx", "npx", []string{"tsx"}},
		{"go run", "go", []string{"run"}},
		{"", "", nil},
	}

	for _, tt := range tests {
		cmd, args := parseInterpreterString(tt.input)
		if cmd != tt.wantCmd {
			t.Errorf("parseInterpreterString(%q) cmd = %q, want %q", tt.input, cmd, tt.wantCmd)
		}
		if len(args) != len(tt.wantArgs) {
			t.Errorf("parseInterpreterString(%q) args = %v, want %v", tt.input, args, tt.wantArgs)
		}
	}
}

func TestResolveInterpreter(t *testing.T) {
	// Create temp files for testing
	dir := t.TempDir()

	// Test annotation priority
	shFile := filepath.Join(dir, "test.sh")
	os.WriteFile(shFile, []byte("#!/usr/bin/env bash\necho hi\n"), 0o644)

	cmd, args, err := ResolveInterpreter(shFile, "python3")
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "python3" {
		t.Errorf("annotation should override: got cmd=%q", cmd)
	}
	if len(args) != 0 {
		t.Errorf("expected no args, got %v", args)
	}

	// Test shebang priority over extension
	cmd, args, err = ResolveInterpreter(shFile, "")
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "bash" {
		t.Errorf("shebang should give bash: got cmd=%q", cmd)
	}

	// Test extension fallback
	pyFile := filepath.Join(dir, "test.py")
	os.WriteFile(pyFile, []byte("print('hi')\n"), 0o644)

	cmd, _, err = ResolveInterpreter(pyFile, "")
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "python3" {
		t.Errorf("extension should give python3: got cmd=%q", cmd)
	}

	// Test unknown extension
	txtFile := filepath.Join(dir, "test.txt")
	os.WriteFile(txtFile, []byte("hello\n"), 0o644)

	_, _, err = ResolveInterpreter(txtFile, "")
	if err == nil {
		t.Error("expected error for unknown extension")
	}
}

func TestHasShebang(t *testing.T) {
	dir := t.TempDir()

	withShebang := filepath.Join(dir, "with")
	os.WriteFile(withShebang, []byte("#!/usr/bin/env bash\necho hi\n"), 0o644)

	withoutShebang := filepath.Join(dir, "without")
	os.WriteFile(withoutShebang, []byte("echo hi\n"), 0o644)

	if !HasShebang(withShebang) {
		t.Error("expected shebang to be detected")
	}
	if HasShebang(withoutShebang) {
		t.Error("expected no shebang")
	}
	if HasShebang(filepath.Join(dir, "nonexistent")) {
		t.Error("expected false for nonexistent file")
	}
}

func TestParseShebangForms(t *testing.T) {
	dir := t.TempDir()

	// #!/usr/bin/env bash
	envForm := filepath.Join(dir, "env")
	os.WriteFile(envForm, []byte("#!/usr/bin/env bash\n"), 0o644)
	cmd, _ := parseShebang(envForm)
	if cmd != "bash" {
		t.Errorf("env form: got %q, want bash", cmd)
	}

	// #!/bin/bash
	directForm := filepath.Join(dir, "direct")
	os.WriteFile(directForm, []byte("#!/bin/bash\n"), 0o644)
	cmd, _ = parseShebang(directForm)
	if cmd != "bash" {
		t.Errorf("direct form: got %q, want bash", cmd)
	}

	// #!/usr/bin/env node --experimental
	envWithArgs := filepath.Join(dir, "envargs")
	os.WriteFile(envWithArgs, []byte("#!/usr/bin/env node --experimental\n"), 0o644)
	cmd, args := parseShebang(envWithArgs)
	if cmd != "node" || len(args) != 1 || args[0] != "--experimental" {
		t.Errorf("env with args: got cmd=%q args=%v", cmd, args)
	}
}

func TestKnownExtensions(t *testing.T) {
	exts := KnownExtensions()
	for _, ext := range []string{".sh", ".ts", ".js", ".py", ".go", ".rb", ".lua"} {
		if !exts[ext] {
			t.Errorf("expected %s to be a known extension", ext)
		}
	}
	if exts[".txt"] {
		t.Error(".txt should not be a known extension")
	}
}
