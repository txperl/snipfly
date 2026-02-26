package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/txperl/GoSnippet/internal/snippet"
)

func newRunnerTestSnippet(t *testing.T, typ snippet.SnippetType, script string) snippet.Snippet {
	t.Helper()
	dir := t.TempDir()
	f := filepath.Join(dir, "test.sh")
	if err := os.WriteFile(f, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return snippet.Snippet{
		Name:        "test",
		Type:        typ,
		Dir:         dir,
		Interpreter: "bash",
		FilePath:    f,
	}
}

func TestRunnerStartStop(t *testing.T) {
	s := newRunnerTestSnippet(t, snippet.TypeService, "#!/usr/bin/env bash\nwhile true; do sleep 0.1; done\n")
	r := New(nil, nil)

	if err := r.Start(s); err != nil {
		t.Fatal(err)
	}

	if !r.HasRunning() {
		t.Error("expected HasRunning = true")
	}

	// Starting same snippet again should fail
	if err := r.Start(s); err == nil {
		t.Error("expected error starting duplicate snippet")
	}

	r.Stop(s)
	proc := r.GetProcess(s.FilePath)
	proc.Wait()

	if r.GetState(s.FilePath) != snippet.StateStopped {
		t.Errorf("state = %v, want Stopped", r.GetState(s.FilePath))
	}
}

func TestRunnerRestart(t *testing.T) {
	s := newRunnerTestSnippet(t, snippet.TypeService, "#!/usr/bin/env bash\nwhile true; do sleep 0.1; done\n")
	r := New(nil, nil)

	if err := r.Start(s); err != nil {
		t.Fatal(err)
	}

	time.Sleep(200 * time.Millisecond)

	if err := r.Restart(s); err != nil {
		t.Fatal(err)
	}

	if r.GetState(s.FilePath) != snippet.StateRunning {
		t.Errorf("state after restart = %v, want Running", r.GetState(s.FilePath))
	}

	r.StopAll()
}

func TestRunnerStopAll(t *testing.T) {
	s1 := newRunnerTestSnippet(t, snippet.TypeService, "#!/usr/bin/env bash\nwhile true; do sleep 0.1; done\n")
	s2 := newRunnerTestSnippet(t, snippet.TypeService, "#!/usr/bin/env bash\nwhile true; do sleep 0.1; done\n")

	r := New(nil, nil)
	r.Start(s1)
	r.Start(s2)

	if !r.HasRunning() {
		t.Error("expected running processes")
	}

	r.StopAll()

	if r.HasRunning() {
		t.Error("expected no running processes after StopAll")
	}
}

func TestRunnerGetBuffer(t *testing.T) {
	s := newRunnerTestSnippet(t, snippet.TypeOneshot, "#!/usr/bin/env bash\necho output\n")
	r := New(nil, nil)

	if buf := r.GetBuffer(s.FilePath); buf != nil {
		t.Error("expected nil buffer before start")
	}

	r.Start(s)
	proc := r.GetProcess(s.FilePath)
	proc.Wait()

	buf := r.GetBuffer(s.FilePath)
	if buf == nil {
		t.Fatal("expected non-nil buffer after run")
	}
	if buf.Len() == 0 {
		t.Error("expected output in buffer")
	}
}

func TestRunnerGetStateIdle(t *testing.T) {
	r := New(nil, nil)
	if r.GetState("/nonexistent") != snippet.StateIdle {
		t.Error("expected StateIdle for unknown snippet")
	}
}
