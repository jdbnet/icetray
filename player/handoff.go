package player

import (
	"time"

	"github.com/gopxl/beep"
)

type streamHandoff struct {
	cancel         chan struct{}
	ctrl           *beep.Ctrl
	fader          *smoothFader
	retireOutgoing func()
}

func (p *Player) detachActivePlayback() *streamHandoff {
	if p.activeCancel == nil && p.ctrl == nil {
		return nil
	}
	h := &streamHandoff{
		cancel: p.activeCancel,
		ctrl:   p.ctrl,
		fader:  p.smoothFader,
	}
	p.activeCancel = nil
	p.ctrl = nil
	p.smoothFader = nil
	p.volumeEffect = nil
	return h
}

func (p *Player) finishHandoff(h *streamHandoff, gen uint64) {
	time.Sleep(CrossfadeDuration())
	if h.cancel != nil {
		close(h.cancel)
	}
	if h.retireOutgoing != nil {
		h.retireOutgoing()
	}
}

// SetHandoffRetire registers a callback to run when the current crossfade ends (e.g. stop the previous HTTP reader).
func (p *Player) SetHandoffRetire(fn func()) {
	p.mu.Lock()
	p.handoffRetire = fn
	p.mu.Unlock()
}
