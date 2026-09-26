package player

import (
	"github.com/gopxl/beep"
)

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
