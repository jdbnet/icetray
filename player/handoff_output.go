//go:build !android

package player

import "github.com/gopxl/beep"

func stabilizeHandoffOutput(_ beep.Streamer) {}
