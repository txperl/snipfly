package runner

import (
	"fmt"
	"testing"
)

func TestRingBufferBasic(t *testing.T) {
	b := NewRingBuffer()

	b.Write("line1")
	b.Write("line2")
	b.Write("line3")

	lines := b.Lines()
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
		t.Errorf("unexpected lines: %v", lines)
	}
	if b.LineCount() != 3 {
		t.Errorf("LineCount = %d, want 3", b.LineCount())
	}
	if b.Len() != 3 {
		t.Errorf("Len = %d, want 3", b.Len())
	}
}

func TestRingBufferWrap(t *testing.T) {
	b := NewRingBuffer()

	// Write more than buffer size
	for i := 0; i < defaultBufferSize+10; i++ {
		b.Write(fmt.Sprintf("line%d", i))
	}

	if b.Len() != defaultBufferSize {
		t.Errorf("Len = %d, want %d", b.Len(), defaultBufferSize)
	}
	if b.LineCount() != uint64(defaultBufferSize+10) {
		t.Errorf("LineCount = %d, want %d", b.LineCount(), defaultBufferSize+10)
	}

	lines := b.Lines()
	if len(lines) != defaultBufferSize {
		t.Fatalf("expected %d lines, got %d", defaultBufferSize, len(lines))
	}

	// Oldest line should be line10
	if lines[0] != "line10" {
		t.Errorf("oldest line = %q, want %q", lines[0], "line10")
	}
	// Newest line should be the last written
	last := fmt.Sprintf("line%d", defaultBufferSize+9)
	if lines[defaultBufferSize-1] != last {
		t.Errorf("newest line = %q, want %q", lines[defaultBufferSize-1], last)
	}
}

func TestRingBufferReset(t *testing.T) {
	b := NewRingBuffer()
	b.Write("line1")
	b.Write("line2")

	b.Reset()

	if b.Len() != 0 {
		t.Errorf("Len after reset = %d, want 0", b.Len())
	}
	if b.LineCount() != 0 {
		t.Errorf("LineCount after reset = %d, want 0", b.LineCount())
	}

	lines := b.Lines()
	if len(lines) != 0 {
		t.Errorf("expected empty lines after reset, got %d", len(lines))
	}
}

func TestRingBufferLinesCopy(t *testing.T) {
	b := NewRingBuffer()
	b.Write("line1")

	lines1 := b.Lines()
	b.Write("line2")
	lines2 := b.Lines()

	// Modifying lines1 should not affect the buffer or lines2
	if len(lines1) != 1 || len(lines2) != 2 {
		t.Errorf("Lines should return independent copies: lines1=%d, lines2=%d", len(lines1), len(lines2))
	}
}
