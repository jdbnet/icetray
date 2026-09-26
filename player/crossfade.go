package player

import (
	"math"

	"github.com/gopxl/beep"
)

// crossfadeStreamer mixes outgoing and incoming audio with complementary cos/sin gains.
type crossfadeStreamer struct {
	outgoing beep.Streamer
	incoming beep.Streamer
	step     int
	steps    int
}

func newCrossfade(outgoing, incoming beep.Streamer, sr beep.SampleRate) *crossfadeStreamer {
	d := CrossfadeDuration()
	steps := sr.N(d)
	if steps < 1 {
		steps = 1
	}
	return &crossfadeStreamer{
		outgoing: outgoing,
		incoming: incoming,
		steps:    steps,
	}
}

func crossfadeGains(step, steps int) (outGain, inGain float64) {
	if steps <= 1 {
		return 0, 1
	}
	t := float64(step) / float64(steps-1)
	if t > 1 {
		t = 1
	}
	return math.Cos(0.5 * math.Pi * t), math.Sin(0.5 * math.Pi * t)
}

func (c *crossfadeStreamer) Stream(samples [][2]float64) (int, bool) {
	if c.incoming == nil {
		return 0, false
	}

	outBuf := make([][2]float64, len(samples))
	inBuf := make([][2]float64, len(samples))

	// Pull incoming first so Android's synchronous Oto read is not blocked on an
	// empty outgoing ahead-buffer during station handoff.
	inN, inOk := streamFill(c.incoming, inBuf)

	var outN int
	var outOk bool
	if c.outgoing != nil {
		// Incoming is already buffered above; blocking on outgoing keeps the fade-off
		// continuous instead of inserting silence when the old ahead queue hiccups.
		outN, outOk = streamFill(c.outgoing, outBuf)
		if outN == 0 && !outOk {
			c.outgoing = nil
		}
	}

	for i := range samples {
		outGain, inGain := crossfadeGains(c.step, c.steps)
		if c.outgoing == nil {
			outGain = 0
			inGain = 1
		}

		var o0, o1 float64
		if i < outN {
			o0, o1 = outBuf[i][0], outBuf[i][1]
		}
		var i0, i1 float64
		if i < inN {
			i0, i1 = inBuf[i][0], inBuf[i][1]
		}

		samples[i][0] = o0*outGain + i0*inGain
		samples[i][1] = o1*outGain + i1*inGain
		c.step++
		if c.step >= c.steps {
			c.outgoing = nil
		}
	}

	// Keep the stream alive while the incoming leg exists; gaps are silence-filled by the platform output.
	if c.incoming != nil {
		return len(samples), true
	}
	ok := inOk || outOk || c.outgoing != nil
	return len(samples), ok
}

func (c *crossfadeStreamer) Err() error {
	if c.incoming != nil {
		if err := c.incoming.Err(); err != nil {
			return err
		}
	}
	if c.outgoing != nil {
		return c.outgoing.Err()
	}
	return nil
}
