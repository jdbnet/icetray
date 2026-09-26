package player

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/gopxl/beep"
)

// smoothFader smoothly fades audio in at stream start and out on Stop to avoid clicks and pops.
// Adapted from https://github.com/gopxl/beep/pull/215 (in-tree while beep is unreleased).
type smoothFader struct {
	Streamer beep.Streamer

	fadingOut atomic.Bool
	fadingIn  bool
	stopped   bool

	fadeIndex     atomic.Int32
	fadeInWindow  []float64
	fadeOutWindow []float64
}

func newSmoothFader(streamer beep.Streamer, sr beep.SampleRate) *smoothFader {
	d := StreamEdgeFadeDuration()
	return newSmoothFaderTimed(streamer, sr, d, d)
}

func newSmoothFaderTimed(streamer beep.Streamer, sr beep.SampleRate, in, out time.Duration) *smoothFader {
	s := &smoothFader{Streamer: streamer}
	if in > 0 {
		s.fadeInWindow = hannWindow(sr, in)
		s.fadingIn = true
		s.fadeIndex.Store(0)
	}
	s.fadeOutWindow = hannWindow(sr, out)
	return s
}

func newSmoothFaderNoFadeIn(streamer beep.Streamer, sr beep.SampleRate) *smoothFader {
	return newSmoothFaderTimed(streamer, sr, 0, StreamEdgeFadeDuration())
}

func hannWindow(sr beep.SampleRate, length time.Duration) []float64 {
	n := sr.N(length)
	if n < 2 {
		n = 2
	}
	win := make([]float64, n)
	denom := float64(n - 1)
	for i := range win {
		win[i] = 0.5 * (1 - math.Cos(math.Pi*float64(i)/denom))
	}
	return win
}

func (s *smoothFader) Stream(samples [][2]float64) (n int, ok bool) {
	if s.stopped {
		return 0, false
	}

	n, ok = s.Streamer.Stream(samples)
	for i := 0; i < n; i++ {
		var gain float64
		switch {
		case s.fadingIn:
			idx := int(s.fadeIndex.Load())
			last := len(s.fadeInWindow) - 1
			if idx > last {
				s.fadingIn = false
				gain = 1
			} else {
				gain = s.fadeInWindow[idx]
				s.fadeIndex.Add(1)
				if s.fadeIndex.Load() > int32(last) {
					s.fadingIn = false
					s.fadeIndex.Store(0)
				}
			}
		case s.fadingOut.Load():
			idx := int(s.fadeIndex.Load())
			last := len(s.fadeOutWindow) - 1
			if idx < 0 {
				idx = 0
			} else if idx > last {
				idx = last
			}
			gain = s.fadeOutWindow[idx]
			s.fadeIndex.Add(-1)
		default:
			gain = 1
		}

		samples[i][0] *= gain
		samples[i][1] *= gain

		if s.fadingOut.Load() && s.fadeIndex.Load() <= 0 {
			s.stopped = true
			return i + 1, false
		}
	}
	return n, ok
}

// Stop begins a fade-out. Safe to call from another goroutine while the speaker is playing.
func (s *smoothFader) Stop() {
	if len(s.fadeOutWindow) == 0 {
		s.stopped = true
		return
	}
	s.fadeIndex.Store(int32(len(s.fadeOutWindow) - 1))
	s.fadingOut.Store(true)
}

func (s *smoothFader) Err() error {
	return s.Streamer.Err()
}
