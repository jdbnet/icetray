package player

import (
	"time"

	"github.com/gopxl/beep"
)

// fadeIn applies a linear gain ramp at the start of a stream to hide resampler and decoder warm-up.
type fadeIn struct {
	src   beep.Streamer
	total int
	pos   int
}

func newFadeIn(src beep.Streamer, rate beep.SampleRate, duration time.Duration) beep.Streamer {
	if duration <= 0 {
		return src
	}
	total := rate.N(duration)
	if total < 1 {
		return src
	}
	return &fadeIn{src: src, total: total}
}

func (f *fadeIn) Stream(samples [][2]float64) (int, bool) {
	n, ok := f.src.Stream(samples)
	for i := 0; i < n; i++ {
		if f.pos < f.total {
			gain := float64(f.pos+1) / float64(f.total)
			samples[i][0] *= gain
			samples[i][1] *= gain
			f.pos++
		}
	}
	return n, ok
}

func (f *fadeIn) Err() error {
	return f.src.Err()
}
