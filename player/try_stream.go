package player

import (
	"github.com/gopxl/beep"
)

// streamFill reads until samples is full or the stream ends.
func streamFill(s beep.Streamer, samples [][2]float64) (int, bool) {
	off := 0
	streamOK := true
	for off < len(samples) {
		n, ok := s.Stream(samples[off:])
		streamOK = ok
		if n == 0 {
			if !ok {
				return off, false
			}
			// Some streamers briefly return (0, true); keep pulling (aheadStreamer blocks instead).
			continue
		}
		off += n
	}
	return off, streamOK
}
