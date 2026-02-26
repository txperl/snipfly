package runner

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/txperl/GoSnippet/internal/snippet"
)

func newTestSnippet(t *testing.T, dir string, typ snippet.SnippetType, script string) snippet.Snippet {
	t.Helper()
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

func TestProcessOneshotSuccess(t *testing.T) {
	dir := t.TempDir()
	s := newTestSnippet(t, dir, snippet.TypeOneshot, "#!/usr/bin/env bash\necho hello\necho world\n")

	var mu sync.Mutex
	var lines []string
	var exitCode int
	exitDone := make(chan struct{})

	proc := NewProcess(s, func(path, line string) {
		mu.Lock()
		lines = append(lines, line)
		mu.Unlock()
	}, func(path string, code int, err error) {
		exitCode = code
		close(exitDone)
	})

	if err := proc.Start(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-exitDone:
	case <-time.After(5 * time.Second):
		t.Fatal("process did not exit in time")
	}

	proc.Wait()

	if proc.State() != snippet.StateDone {
		t.Errorf("state = %v, want Done", proc.State())
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(lines) != 2 || lines[0] != "hello" || lines[1] != "world" {
		t.Errorf("output lines = %v", lines)
	}
}

func TestProcessOneshotFailure(t *testing.T) {
	dir := t.TempDir()
	s := newTestSnippet(t, dir, snippet.TypeOneshot, "#!/usr/bin/env bash\nexit 42\n")

	proc := NewProcess(s, nil, nil)
	if err := proc.Start(); err != nil {
		t.Fatal(err)
	}
	proc.Wait()

	if proc.State() != snippet.StateFailed {
		t.Errorf("state = %v, want Failed", proc.State())
	}
	if proc.ExitCode() != 42 {
		t.Errorf("exitCode = %d, want 42", proc.ExitCode())
	}
}

func TestProcessServiceStop(t *testing.T) {
	dir := t.TempDir()
	s := newTestSnippet(t, dir, snippet.TypeService, "#!/usr/bin/env bash\nwhile true; do echo tick; sleep 0.1; done\n")

	proc := NewProcess(s, nil, nil)
	if err := proc.Start(); err != nil {
		t.Fatal(err)
	}

	// Let it run briefly
	time.Sleep(300 * time.Millisecond)

	if proc.State() != snippet.StateRunning {
		t.Errorf("state = %v, want Running", proc.State())
	}

	proc.Stop()
	proc.Wait()

	if proc.State() != snippet.StateStopped {
		t.Errorf("state = %v, want Stopped", proc.State())
	}

	if proc.Buffer().Len() == 0 {
		t.Error("expected some output in buffer")
	}
}

func TestProcessServiceCrash(t *testing.T) {
	dir := t.TempDir()
	s := newTestSnippet(t, dir, snippet.TypeService, "#!/usr/bin/env bash\nexit 1\n")

	proc := NewProcess(s, nil, nil)
	if err := proc.Start(); err != nil {
		t.Fatal(err)
	}
	proc.Wait()

	if proc.State() != snippet.StateCrashed {
		t.Errorf("state = %v, want Crashed", proc.State())
	}
}

func TestProcessOneshotExitNotBlockedByBackgroundChild(t *testing.T) {
	dir := t.TempDir()
	s := newTestSnippet(t, dir, snippet.TypeOneshot, `#!/usr/bin/env bash
(sleep 2) &
echo parent-done
exit 0
`)

	proc := NewProcess(s, nil, nil)
	if err := proc.Start(); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		proc.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("process wait blocked by inherited stdio in background child")
	}

	if proc.State() != snippet.StateDone {
		t.Errorf("state = %v, want Done", proc.State())
	}
	if proc.ExitCode() != 0 {
		t.Errorf("exitCode = %d, want 0", proc.ExitCode())
	}
}

func TestProcessOnExitCallbackCanQueryState(t *testing.T) {
	dir := t.TempDir()
	s := newTestSnippet(t, dir, snippet.TypeOneshot, "#!/usr/bin/env bash\necho done\n")

	callbackDone := make(chan struct{})
	var proc *Process
	proc = NewProcess(s, nil, func(path string, code int, err error) {
		// Regression test: callback must not run while process mutex is held.
		_ = proc.State()
		close(callbackDone)
	})

	if err := proc.Start(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-callbackDone:
	case <-time.After(1 * time.Second):
		t.Fatal("onExit callback appears blocked when querying process state")
	}

	proc.Wait()
	if proc.State() != snippet.StateDone {
		t.Errorf("state = %v, want Done", proc.State())
	}
}
