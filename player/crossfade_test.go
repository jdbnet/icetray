package player

import (
	"math"
	"testing"

	"github.com/gopxl/beep"
)

func TestCrossfadeGains(t *testing.T) {
	out, in := crossfadeGains(0, 100)
	if out < 0.99 || in > 0.01 {
		t.Fatalf("start: expected mostly outgoing, got out=%v in=%v", out, in)
	}
	out, in = crossfadeGains(99, 100)
	if out > 0.01 || in < 0.99 {
		t.Fatalf("end: expected mostly incoming, got out=%v in=%v", out, in)
	}
	midOut, midIn := crossfadeGains(50, 100)
	sum := midOut*midOut + midIn*midIn
	if math.Abs(sum-1) > 0.2 {
		t.Fatalf("midpoint power should be near 1, got %v", sum)
	}
}

func TestCrossfadeMixesStreams(t *testing.T) {
	const sr = beep.SampleRate(44100)
	cf := newCrossfade(constantStreamer{v: 1}, constantStreamer{v: 0.5}, sr)
	buf := make([][2]float64, 256)
	n, ok := cf.Stream(buf)
	if n == 0 || !ok {
		t.Fatal("expected crossfade samples")
	}
	if buf[0][0] < 0.9 {
		t.Fatalf("first sample should be mostly outgoing, got %v", buf[0][0])
	}
}
