package snippet

// SnippetType defines how a snippet behaves when executed.
type SnippetType = string

const (
	TypeOneshot SnippetType = "oneshot"
	TypeService SnippetType = "service"
)

// ProcessState represents the lifecycle state of a snippet's process.
type ProcessState int

const (
	StateIdle ProcessState = iota
	StateRunning
	StateStopped
	StateCrashed
	StateExited
	StateDone
	StateFailed
)

func (s ProcessState) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateRunning:
		return "Running"
	case StateStopped:
		return "Stopped"
	case StateCrashed:
		return "Crashed"
	case StateExited:
		return "Exited"
	case StateDone:
		return "Done"
	case StateFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// Snippet represents a runnable code snippet with its metadata.
type Snippet struct {
	Name            string
	Desc            string
	Type            SnippetType
	Dir             string
	Env             []string
	Interpreter     string
	InterpreterArgs []string
	FilePath        string
	Group           string
	State           ProcessState
	Error           string // non-empty when the snippet cannot be run (e.g. unknown interpreter)
}
