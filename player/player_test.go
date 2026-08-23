package player

import (
	"testing"
)

func TestIsPlayingReflectsAudioOutput(t *testing.T) {
	p := &Player{
		volume:       100,
		speakerReady: true,
		isRunning:    true,
		isPaused:     false,
		audioActive:  false,
	}

	if p.IsPlaying() {
		t.Fatal("expected not playing before audio is active")
	}

	p.mu.Lock()
	p.audioActive = true
	p.mu.Unlock()

	if !p.IsPlaying() {
		t.Fatal("expected playing when audio is active")
	}

	p.mu.Lock()
	p.isPaused = true
	p.mu.Unlock()

	if p.IsPlaying() {
		t.Fatal("expected not playing when paused")
	}
}

func TestStateChangeListeners(t *testing.T) {
	p := &Player{speakerReady: true}
	called := false
	p.AddStateChangeListener(func() {
		called = true
	})

	p.mu.Lock()
	p.audioActive = true
	p.mu.Unlock()
	p.notifyStateChange()

	if !called {
		t.Fatal("expected state change listener to run")
	}
}
