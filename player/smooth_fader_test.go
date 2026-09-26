package player

import (
	"testing"
	"time"

	"github.com/gopxl/beep"
)

type constantStreamer struct {
	v float64
}

func (c constantStreamer) Stream(samples [][2]float64) (int, bool) {
	for i := range samples {
		samples[i][0] = c.v
		samples[i][1] = c.v
	}
	return len(samples), true
}

func (c constantStreamer) Err() error { return nil }

func TestSmoothFaderFadeInReducesStart(t *testing.T) {
	const sr = beep.SampleRate(44100)
	f := newSmoothFaderTimed(constantStreamer{v: 1}, sr, 20*time.Millisecond, 20*time.Millisecond)
	buf := make([][2]float64, 64)
	n, ok := f.Stream(buf)
	if n == 0 || !ok {
		t.Fatal("expected samples")
	}
	if buf[0][0] >= 0.01 {
		t.Fatalf("first sample should be near silent during fade-in, got %v", buf[0][0])
	}
}

func TestSmoothFaderStopEndsStream(t *testing.T) {
	const sr = beep.SampleRate(44100)
	f := newSmoothFaderTimed(constantStreamer{v: 1}, sr, 20*time.Millisecond, 20*time.Millisecond)
	buf := make([][2]float64, 4096)
	for {
		_, ok := f.Stream(buf)
		if !ok {
			break
		}
		if !f.fadingIn {
			break
		}
	}
	f.Stop()
	var ended bool
	for range 20 {
		_, ok := f.Stream(buf)
		if !ok {
			ended = true
			break
		}
	}
	if !ended {
		t.Fatal("expected stream to end after Stop fade-out")
	}
}
