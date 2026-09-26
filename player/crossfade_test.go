package player

import (
	"testing"
	"time"

	"github.com/gopxl/beep"
)

func TestCrossfadeSequentialZeroDuration(t *testing.T) {
	prev := CrossfadeDuration()
	SetCrossfadeDuration(0)
	t.Cleanup(func() { SetCrossfadeDuration(prev) })

	const sr = beep.SampleRate(44100)
	cf, done := newCrossfade(constantStreamer{v: 1}, constantStreamer{v: 0.25}, sr, nil)

	outSteps := sr.N(StreamEdgeFadeDuration())
	if outSteps < 2 {
		outSteps = 2
	}

	buf := make([][2]float64, 64)
	n, ok := cf.Stream(buf)
	if n == 0 || !ok {
		t.Fatal("expected samples")
	}
	if buf[0][0] < 0.5 {
		t.Fatalf("first sequential sample should be mostly outgoing, got %v", buf[0][0])
	}
	if buf[0][0] < 0.9 && cf.step >= outSteps {
		t.Fatalf("unexpected early incoming bleed at step %d", cf.step)
	}

	total := cf.steps
	pumped := n
	for pumped < total+256 {
		n, ok = cf.Stream(buf)
		if n == 0 && !ok {
			break
		}
		pumped += n
		select {
		case <-done:
			return
		default:
		}
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("sequential crossfade did not complete")
	}
}

func TestFadeOutSleepDurationUsesEdgeFade(t *testing.T) {
	d := fadeOutSleepDuration(false)
	if d < StreamEdgeFadeDuration() {
		t.Fatalf("stop fade wait should cover edge fade, got %v", d)
	}
}
