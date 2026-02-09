package acp

import (
	"sync"
)

// tailBuffer is an io.Writer that retains only the last N bytes written.
// It is safe for concurrent use.
type tailBuffer struct {
	limit int

	mu  sync.Mutex
	buf []byte
}

func newTailBuffer(limit int) *tailBuffer {
	if limit < 0 {
		limit = 0
	}
	return &tailBuffer{limit: limit}
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if b.limit == 0 || n == 0 {
		return n, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if n >= b.limit {
		// Keep the last limit bytes of p.
		b.buf = append(b.buf[:0], p[n-b.limit:]...)
		return n, nil
	}

	space := b.limit - len(b.buf)
	if n <= space {
		b.buf = append(b.buf, p...)
		return n, nil
	}

	// Drop the oldest bytes so the new data fits.
	drop := n - space
	copy(b.buf, b.buf[drop:])
	b.buf = b.buf[:len(b.buf)-drop]
	b.buf = append(b.buf, p...)
	return n, nil
}

func (b *tailBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}
