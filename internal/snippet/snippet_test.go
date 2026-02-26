package snippet

import "testing"

func TestProcessStateString(t *testing.T) {
	tests := []struct {
		state ProcessState
		want  string
	}{
		{StateIdle, "Idle"},
		{StateRunning, "Running"},
		{StateStopped, "Stopped"},
		{StateCrashed, "Crashed"},
		{StateExited, "Exited"},
		{StateDone, "Done"},
		{StateFailed, "Failed"},
		{ProcessState(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("ProcessState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}
