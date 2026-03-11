package runner

import (
	"bufio"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/txperl/snipfly/internal/snippet"
)

// OutputCallback is called whenever a process produces a line of output.
type OutputCallback func(snippetPath, line string)

// ExitCallback is called when a process exits.
type ExitCallback func(snippetPath string, exitCode int, err error)

// Process wraps an exec.Cmd with lifecycle management, output buffering, and callbacks.
type Process struct {
	snippet  snippet.Snippet
	cmd      *exec.Cmd
	mu       sync.Mutex
	state    snippet.ProcessState
	exitCode int
	ptmx     *os.File
	buffer   *RingBuffer
	onOutput OutputCallback
	onExit   ExitCallback
	done     chan struct{}
}

// NewProcess creates a new Process for the given snippet.
func NewProcess(s snippet.Snippet, onOutput OutputCallback, onExit ExitCallback) *Process {
	return &Process{
		snippet:  s,
		buffer:   NewRingBuffer(),
		onOutput: onOutput,
		onExit:   onExit,
		done:     make(chan struct{}),
	}
}

// Start launches the process.
func (p *Process) Start() error {
	args := append(p.snippet.InterpreterArgs, p.snippet.FilePath)
	p.cmd = exec.Command(p.snippet.Interpreter, args...)
	p.cmd.Dir = p.snippet.Dir
	p.cmd.Env = append(os.Environ(), p.snippet.Env...)

	p.buffer.Reset()

	var wg sync.WaitGroup

	if p.snippet.PTY {
		// pty.Start() sets Setsid+Setctty which conflicts with Setpgid.
		// Setsid already creates a new process group, so Setpgid is unnecessary.
		p.cmd.SysProcAttr = &syscall.SysProcAttr{}
		ptmx, err := pty.Start(p.cmd)
		if err != nil {
			close(p.done)
			return err
		}
		p.ptmx = ptmx

		p.mu.Lock()
		p.state = snippet.StateRunning
		p.mu.Unlock()

		// PTY merges stdout+stderr into a single fd
		wg.Add(1)
		go p.readPTY(&wg, ptmx)
	} else {
		p.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		stdout, err := p.cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderr, err := p.cmd.StderrPipe()
		if err != nil {
			return err
		}

		if err := p.cmd.Start(); err != nil {
			close(p.done)
			return err
		}

		p.mu.Lock()
		p.state = snippet.StateRunning
		p.mu.Unlock()

		// Read stdout and stderr concurrently
		wg.Add(2)
		go p.readPipe(&wg, stdout)
		go p.readPipe(&wg, stderr)
	}

	// Wait for process exit in background
	go func() {
		defer close(p.done)
		// Call Wait first so os/exec can release pipe writers and allow scanners to see EOF.
		err := p.cmd.Wait()
		// Close PTY master after process exits to unblock readPTY.
		if p.ptmx != nil {
			p.ptmx.Close()
		}
		// Drain remaining output after process exit.
		wg.Wait()
		p.handleExit(err)
	}()

	return nil
}

func (p *Process) readPTY(wg *sync.WaitGroup, r io.Reader) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		p.buffer.Write(line)
		if p.onOutput != nil {
			p.onOutput(p.snippet.FilePath, line)
		}
	}
	// PTY returns EIO when the child exits — this is normal, not an error.
	if err := scanner.Err(); err != nil && !errors.Is(err, syscall.EIO) {
		_ = err // unexpected error, but nothing to do
	}
}

func (p *Process) readPipe(wg *sync.WaitGroup, r io.ReadCloser) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max line
	for scanner.Scan() {
		line := scanner.Text()
		p.buffer.Write(line)
		if p.onOutput != nil {
			p.onOutput(p.snippet.FilePath, line)
		}
	}
}

func (p *Process) handleExit(err error) {
	p.mu.Lock()
	var cb ExitCallback
	var path string
	var code int

	// If already stopped by user, keep StateStopped
	if p.state == snippet.StateStopped {
		p.exitCode = p.cmd.ProcessState.ExitCode()
		cb = p.onExit
		path = p.snippet.FilePath
		code = p.exitCode
		p.mu.Unlock()
		if cb != nil {
			go cb(path, code, err)
		}
		return
	}

	p.exitCode = p.cmd.ProcessState.ExitCode()

	if p.snippet.Type == snippet.TypeService {
		if p.exitCode == 0 {
			p.state = snippet.StateExited
		} else {
			p.state = snippet.StateCrashed
		}
	} else {
		// oneshot
		if p.exitCode == 0 {
			p.state = snippet.StateDone
		} else {
			p.state = snippet.StateFailed
		}
	}

	cb = p.onExit
	path = p.snippet.FilePath
	code = p.exitCode
	p.mu.Unlock()

	if cb != nil {
		go cb(path, code, err)
	}
}

// Stop sends SIGTERM to the process group, then SIGKILL after 5 seconds.
func (p *Process) Stop() {
	p.mu.Lock()
	if p.state != snippet.StateRunning {
		p.mu.Unlock()
		return
	}
	p.state = snippet.StateStopped
	p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return
	}

	pid := p.cmd.Process.Pid
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		// Fallback: signal the process directly
		_ = p.cmd.Process.Signal(syscall.SIGTERM)
	} else {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	}

	// Wait up to 5 seconds for graceful exit, then SIGKILL
	select {
	case <-p.done:
		return
	case <-time.After(5 * time.Second):
	}

	if err != nil {
		_ = p.cmd.Process.Kill()
	} else {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}
}

// State returns the current process state.
func (p *Process) State() snippet.ProcessState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

// ExitCode returns the process exit code.
func (p *Process) ExitCode() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitCode
}

// PID returns the process ID, or 0 if the process is not running.
func (p *Process) PID() int {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

// Buffer returns the process output ring buffer.
func (p *Process) Buffer() *RingBuffer {
	return p.buffer
}

// Wait blocks until the process has exited.
func (p *Process) Wait() {
	<-p.done
}
