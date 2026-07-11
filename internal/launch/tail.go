package launch

import "sync"

// Tail is a fixed-size ring buffer keeping the last N bytes written.
type Tail struct {
	mu   sync.Mutex
	buf  []byte
	size int
}

func NewTail(size int) *Tail { return &Tail{size: size} }

func (t *Tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if len(t.buf) > t.size {
		t.buf = t.buf[len(t.buf)-t.size:]
	}
	return len(p), nil
}

func (t *Tail) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.buf)
}
