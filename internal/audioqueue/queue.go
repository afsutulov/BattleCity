// Package audioqueue bridges the single emulation thread and audio device.
package audioqueue

import "sync"

type Queue struct {
	mu         sync.Mutex
	buf        []byte
	read, used int
	muted      bool
}

func New() *Queue { return &Queue{buf: make([]byte, 48000*4/8)} } // bounded at 125 ms
func (q *Queue) Read(p []byte) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	// Reads always finish; an underrun produces silence and cannot hang shutdown.
	clear(p)
	n := min(len(p), q.used)
	for i := 0; i < n; i++ {
		p[i] = q.buf[(q.read+i)%len(q.buf)]
	}
	q.read = (q.read + n) % len(q.buf)
	q.used -= n
	return len(p), nil
}
func (q *Queue) Push(samples []int16) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.muted {
		return
	}
	n := len(samples) * 2
	if n > len(q.buf) {
		samples = samples[len(samples)-len(q.buf)/2:]
		n = len(samples) * 2
	}
	if extra := q.used + n - len(q.buf); extra > 0 {
		extra = (extra + 3) &^ 3
		q.read = (q.read + extra) % len(q.buf)
		q.used -= extra
	}
	pos := (q.read + q.used) % len(q.buf)
	for _, s := range samples {
		q.buf[pos] = byte(s)
		q.buf[(pos+1)%len(q.buf)] = byte(s >> 8)
		pos = (pos + 2) % len(q.buf)
	}
	q.used += n
}
func (q *Queue) Clear()      { q.mu.Lock(); q.read = 0; q.used = 0; q.mu.Unlock() }
func (q *Queue) Mute(m bool) { q.mu.Lock(); q.muted = m; q.read = 0; q.used = 0; q.mu.Unlock() }
