package runner

import "sync"

const defaultBufferSize = 1000

// RingBuffer is a thread-safe circular buffer for storing output lines.
type RingBuffer struct {
	mu    sync.Mutex
	lines []string
	size  int
	head  int // next write position
	count uint64
}

// NewRingBuffer creates a ring buffer with the default capacity (1000 lines).
func NewRingBuffer() *RingBuffer {
	return &RingBuffer{
		lines: make([]string, defaultBufferSize),
		size:  defaultBufferSize,
	}
}

// Write appends a line to the buffer, overwriting the oldest if full.
func (b *RingBuffer) Write(line string) {
	b.mu.Lock()
	b.lines[b.head] = line
	b.head = (b.head + 1) % b.size
	b.count++
	b.mu.Unlock()
}

// Lines returns a copy of all buffered lines in order from oldest to newest.
func (b *RingBuffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()

	n := b.Len()
	result := make([]string, n)

	if b.count <= uint64(b.size) {
		// Buffer hasn't wrapped yet
		copy(result, b.lines[:n])
	} else {
		// Buffer has wrapped: oldest is at head, newest is at head-1
		firstPart := b.size - b.head
		copy(result, b.lines[b.head:])
		copy(result[firstPart:], b.lines[:b.head])
	}

	return result
}

// LineCount returns the total number of lines ever written (monotonically increasing).
// TUI can use this for change detection.
func (b *RingBuffer) LineCount() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count
}

// Len returns the number of lines currently stored in the buffer.
func (b *RingBuffer) Len() int {
	if b.count < uint64(b.size) {
		return int(b.count)
	}
	return b.size
}

// Reset clears the buffer.
func (b *RingBuffer) Reset() {
	b.mu.Lock()
	b.lines = make([]string, b.size)
	b.head = 0
	b.count = 0
	b.mu.Unlock()
}
