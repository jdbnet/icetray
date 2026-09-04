package player

import (
	"testing"
	"time"
)

type blockingStreamer struct {
	release chan struct{}
}

func (b *blockingStreamer) Stream(samples [][2]float64) (int, bool) {
	<-b.release
	for i := range samples {
		samples[i] = [2]float64{0.5, 0.5}
	}
	return len(samples), true
}

func (b *blockingStreamer) Err() error { return nil }

func TestAheadStreamerWaitsUntilPCMOrStop(t *testing.T) {
	src := &blockingStreamer{release: make(chan struct{})}
	ahead := newAheadStreamer(src, 1024)
	defer ahead.stopFill()

	done := make(chan struct{})
	go func() {
		ahead.Stream(make([][2]float64, 8))
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Stream returned before PCM was available")
	case <-time.After(50 * time.Millisecond):
	}

	ahead.stopFill()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Stream did not unblock after stopFill")
	}
}

func TestAheadStreamerCopiesDecodedPCM(t *testing.T) {
	src := &blockingStreamer{release: make(chan struct{})}
	ahead := newAheadStreamer(src, 1024)
	defer ahead.stopFill()

	close(src.release)
	ahead.waitReady(32, time.Second)

	samples := make([][2]float64, 32)
	n, ok := ahead.Stream(samples)
	if !ok || n != 32 {
		t.Fatalf("got n=%d ok=%v", n, ok)
	}
	if samples[0][0] != 0.5 {
		t.Fatalf("expected decoded PCM, got %v", samples[0])
	}
}
