//go:build !android

package player

import "github.com/gopxl/beep"

func bufferPlayback(src beep.Streamer) beep.Streamer {
	return src
}
