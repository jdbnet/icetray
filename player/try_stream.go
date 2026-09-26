package player

import (
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
)

// tryStreamer can return partial PCM without blocking when a buffer is momentarily empty.
type tryStreamer interface {
	TryStream(samples [][2]float64) (int, bool)
}

// streamFill reads until samples is full or the stream returns no data.
func streamFill(s beep.Streamer, samples [][2]float64) (int, bool) {
	off := 0
	ok := true
	for off < len(samples) {
		n, ok := s.Stream(samples[off:])
		if n == 0 {
			return off, ok
		}
		off += n
	}
	return off, ok
}

// streamFillNonBlocking is like streamFill but never blocks on aheadStreamer queues.
func streamFillNonBlocking(s beep.Streamer, samples [][2]float64) (int, bool) {
	off := 0
	ok := true
	for off < len(samples) {
		n, ok2 := streamNonBlocking(s, samples[off:])
		if n == 0 {
			return off, ok2
		}
		off += n
		ok = ok2
	}
	return off, ok
}

// streamNonBlocking pulls PCM without blocking on decode buffers (silence when empty).
func streamNonBlocking(s beep.Streamer, samples [][2]float64) (int, bool) {
	for i := 0; i < 12; i++ {
		if st, ok := s.(*beep.Ctrl); ok {
			if st.Paused {
				for j := range samples {
					samples[j] = [2]float64{}
				}
				return len(samples), true
			}
			s = st.Streamer
			continue
		}
		if st, ok := s.(*effects.Volume); ok {
			s = st.Streamer
			continue
		}
		if st, ok := s.(*smoothFader); ok {
			return st.tryStream(samples)
		}
		if st, ok := s.(*aheadStreamer); ok {
			return st.TryStream(samples)
		}
		if t, ok := s.(tryStreamer); ok {
			return t.TryStream(samples)
		}
		return s.Stream(samples)
	}
	return 0, false
}
