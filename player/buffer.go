package player

import (
	"sync"
	"time"

	"github.com/gopxl/beep"
)

type playbackBuffer interface {
	waitReady(min int, timeout time.Duration)
	stopFill()
}

// aheadStreamer decodes into a PCM queue on a background goroutine so the
// audio device is never fed silence just because a decode or network read is slow.
type aheadStreamer struct {
	src    beep.Streamer
	mu     sync.Mutex
	cond   *sync.Cond
	buf    [][2]float64
	ahead  int
	closed bool
	eof    bool
}

func newAheadStreamer(src beep.Streamer, ahead int) *aheadStreamer {
	if ahead < 1 {
		ahead = 1
	}
	a := &aheadStreamer{
		src:   src,
		ahead: ahead,
		buf:   make([][2]float64, 0, ahead),
	}
	a.cond = sync.NewCond(&a.mu)
	go a.fill()
	return a
}

func (a *aheadStreamer) fill() {
	tmp := make([][2]float64, 4096)
	for {
		a.mu.Lock()
		for len(a.buf) >= a.ahead && !a.closed {
			a.cond.Wait()
		}
		closed := a.closed
		a.mu.Unlock()
		if closed {
			return
		}

		n, ok := a.src.Stream(tmp)
		if n == 0 && ok {
			time.Sleep(2 * time.Millisecond)
		}
		a.mu.Lock()
		if n > 0 {
			a.buf = append(a.buf, tmp[:n]...)
		}
		if !ok {
			a.eof = true
			a.cond.Broadcast()
			a.mu.Unlock()
			return
		}
		a.cond.Broadcast()
		a.mu.Unlock()
	}
}

func (a *aheadStreamer) Stream(samples [][2]float64) (int, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for len(a.buf) == 0 && !a.eof && !a.closed {
		a.cond.Wait()
	}
	if len(a.buf) == 0 {
		return 0, false
	}
	n := copy(samples, a.buf)
	a.buf = a.buf[n:]
	a.cond.Signal()
	return n, true
}

func (a *aheadStreamer) Err() error {
	return a.src.Err()
}

func (a *aheadStreamer) stopFill() {
	a.mu.Lock()
	a.closed = true
	a.cond.Broadcast()
	a.mu.Unlock()
}

func (a *aheadStreamer) waitReady(min int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		ready := len(a.buf) >= min || a.eof || a.closed
		a.mu.Unlock()
		if ready {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}
