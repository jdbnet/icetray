package player

import (
	"math"
	"sync"

	"github.com/gopxl/beep"
)

// crossfadeStreamer mixes outgoing and incoming audio with complementary cos/sin gains.
// When CrossfadeDuration is 0, it runs a sequential fade-out then fade-in with no overlap.
type crossfadeStreamer struct {
	outgoing beep.Streamer
	incoming beep.Streamer
	step     int
	steps    int

	sequential bool
	outSteps   int
	inSteps    int

	done     chan struct{}
	once     sync.Once
	onStable func(beep.Streamer)

	outBuf [][2]float64
	inBuf  [][2]float64
}

func newCrossfade(outgoing, incoming beep.Streamer, sr beep.SampleRate, onStable func(beep.Streamer)) (*crossfadeStreamer, <-chan struct{}) {
	d := CrossfadeDuration()
	c := &crossfadeStreamer{
		outgoing: outgoing,
		incoming: incoming,
		done:     make(chan struct{}),
		onStable: onStable,
	}
	if d <= 0 {
		c.sequential = true
		c.outSteps = sr.N(StreamEdgeFadeDuration())
		c.inSteps = sr.N(StreamEdgeFadeDuration())
		if c.outSteps < 2 {
			c.outSteps = 2
		}
		if c.inSteps < 2 {
			c.inSteps = 2
		}
		c.steps = c.outSteps + c.inSteps
	} else {
		c.steps = sr.N(d)
		if c.steps < 2 {
			c.steps = 2
		}
	}
	return c, c.done
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

func sequentialFadeOutGain(step, steps int) float64 {
	if steps <= 1 {
		return 0
	}
	t := float64(step) / float64(steps-1)
	if t > 1 {
		t = 1
	}
	return math.Cos(0.5 * math.Pi * t)
}

func sequentialFadeInGain(step, steps int) float64 {
	if steps <= 1 {
		return 1
	}
	t := float64(step) / float64(steps-1)
	if t > 1 {
		t = 1
	}
	return math.Sin(0.5 * math.Pi * t)
}

func (c *crossfadeStreamer) gainsForStep() (outGain, inGain float64) {
	if !c.sequential {
		return crossfadeGains(c.step, c.steps)
	}
	if c.step < c.outSteps {
		return sequentialFadeOutGain(c.step, c.outSteps), 0
	}
	inStep := c.step - c.outSteps
	return 0, sequentialFadeInGain(inStep, c.inSteps)
}

func (c *crossfadeStreamer) Stream(samples [][2]float64) (int, bool) {
	if c.incoming == nil {
		return 0, false
	}

	if cap(c.outBuf) < len(samples) {
		c.outBuf = make([][2]float64, len(samples))
	} else {
		c.outBuf = c.outBuf[:len(samples)]
	}
	if cap(c.inBuf) < len(samples) {
		c.inBuf = make([][2]float64, len(samples))
	} else {
		c.inBuf = c.inBuf[:len(samples)]
	}
	outBuf := c.outBuf
	inBuf := c.inBuf

	pullIncoming := !c.sequential || c.step >= c.outSteps
	var inN int
	var inOk bool
	if pullIncoming {
		inN, inOk = streamFill(c.incoming, inBuf)
	}

	var outN int
	var outOk bool
	pullOutgoing := c.outgoing != nil && (!c.sequential || c.step < c.outSteps)
	if pullOutgoing {
		outN, outOk = streamFill(c.outgoing, outBuf)
		if outN == 0 && !outOk {
			c.outgoing = nil
		}
	}

	for i := range samples {
		outGain, inGain := c.gainsForStep()
		if c.outgoing == nil {
			outGain = 0
			if !c.sequential || c.step >= c.outSteps {
				inGain = 1
			}
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

	c.signalStableIfComplete()

	if c.incoming != nil {
		return len(samples), true
	}
	ok := inOk || outOk || c.outgoing != nil
	return len(samples), ok
}

func (c *crossfadeStreamer) signalStableIfComplete() {
	if c.step < c.steps {
		return
	}
	c.once.Do(func() {
		close(c.done)
		if c.onStable != nil && c.incoming != nil {
			c.onStable(c.incoming)
		}
	})
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
