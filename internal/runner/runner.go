package runner

import (
	"fmt"
	"sync"

	"github.com/txperl/GoSnippet/internal/snippet"
)

// Runner manages the lifecycle of multiple snippet processes.
type Runner struct {
	mu        sync.Mutex
	processes map[string]*Process
	onOutput  OutputCallback
	onExit    ExitCallback
}

// New creates a Runner with the given callbacks.
func New(onOutput OutputCallback, onExit ExitCallback) *Runner {
	return &Runner{
		processes: make(map[string]*Process),
		onOutput:  onOutput,
		onExit:    onExit,
	}
}

// SetCallbacks replaces the output and exit callbacks (e.g., after TUI is ready).
func (r *Runner) SetCallbacks(onOutput OutputCallback, onExit ExitCallback) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onOutput = onOutput
	r.onExit = onExit
}

// Start launches a new process for the given snippet.
// Returns an error if the snippet is already running.
func (r *Runner) Start(s snippet.Snippet) error {
	r.mu.Lock()
	if proc, ok := r.processes[s.FilePath]; ok {
		if proc.State() == snippet.StateRunning {
			r.mu.Unlock()
			return fmt.Errorf("snippet %s is already running", s.FilePath)
		}
	}
	proc := NewProcess(s, r.onOutput, r.onExit)
	r.processes[s.FilePath] = proc
	r.mu.Unlock()

	return proc.Start()
}

// Stop stops the process for the given snippet.
func (r *Runner) Stop(s snippet.Snippet) {
	r.mu.Lock()
	proc, ok := r.processes[s.FilePath]
	r.mu.Unlock()

	if ok {
		proc.Stop()
	}
}

// Restart stops the current process, waits for it to fully exit, then starts a new one.
func (r *Runner) Restart(s snippet.Snippet) error {
	r.mu.Lock()
	proc, ok := r.processes[s.FilePath]
	r.mu.Unlock()

	if ok {
		proc.Stop()
		proc.Wait()
	}

	return r.Start(s)
}

// StopAll stops all running processes in parallel and waits for them to complete.
func (r *Runner) StopAll() {
	r.mu.Lock()
	procs := make([]*Process, 0, len(r.processes))
	for _, proc := range r.processes {
		if proc.State() == snippet.StateRunning {
			procs = append(procs, proc)
		}
	}
	r.mu.Unlock()

	var wg sync.WaitGroup
	for _, proc := range procs {
		wg.Add(1)
		go func(p *Process) {
			defer wg.Done()
			p.Stop()
			p.Wait()
		}(proc)
	}
	wg.Wait()
}

// HasRunning returns true if any process is currently running.
func (r *Runner) HasRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, proc := range r.processes {
		if proc.State() == snippet.StateRunning {
			return true
		}
	}
	return false
}

// GetProcess returns the Process for a given file path, or nil if not found.
func (r *Runner) GetProcess(filePath string) *Process {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.processes[filePath]
}

// GetBuffer returns the output buffer for a given file path, or nil if not found.
func (r *Runner) GetBuffer(filePath string) *RingBuffer {
	r.mu.Lock()
	proc, ok := r.processes[filePath]
	r.mu.Unlock()
	if !ok {
		return nil
	}
	return proc.Buffer()
}

// GetState returns the state of the process for a given file path.
// Returns StateIdle if no process exists.
func (r *Runner) GetState(filePath string) snippet.ProcessState {
	r.mu.Lock()
	proc, ok := r.processes[filePath]
	r.mu.Unlock()
	if !ok {
		return snippet.StateIdle
	}
	return proc.State()
}
