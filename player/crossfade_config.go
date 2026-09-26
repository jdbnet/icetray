package player

import (
	"sync/atomic"
	"time"
)

const (
	defaultCrossfade = 2 * time.Second
	minCrossfade     = 0
	maxCrossfade     = 8 * time.Second
	// Short edge fades for decode/resample click suppression (not station crossfade).
	streamEdgeFade = 75 * time.Millisecond
)

var crossfadeNanos atomic.Int64

func init() {
	crossfadeNanos.Store(int64(defaultCrossfade))
}

// SetCrossfadeDuration configures stream crossfade length (clamped to 0–8s).
func SetCrossfadeDuration(d time.Duration) {
	if d < minCrossfade {
		d = minCrossfade
	} else if d > maxCrossfade {
		d = maxCrossfade
	}
	crossfadeNanos.Store(int64(d))
}

// CrossfadeDuration returns the configured crossfade length.
func CrossfadeDuration() time.Duration {
	return time.Duration(crossfadeNanos.Load())
}

// StreamEdgeFadeDuration is the Hann fade used at stream start/stop (independent of crossfade).
func StreamEdgeFadeDuration() time.Duration {
	return streamEdgeFade
}

// EdgeFadeSleepDuration is how long to wait for a stream-edge fade-out to reach the device.
func EdgeFadeSleepDuration() time.Duration {
	return streamEdgeFade + 100*time.Millisecond
}

// HandoffCompletionDuration is the longest expected handoff when crossfade is disabled (0s).
func HandoffCompletionDuration() time.Duration {
	return 2*streamEdgeFade + 100*time.Millisecond
}
