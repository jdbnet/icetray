//go:build !android

package player

import "testing"

func TestOutputBufferSamplesWindowsIsLarger(t *testing.T) {
	windows := outputBufferSamples("windows")
	other := outputBufferSamples("linux")
	if windows <= other {
		t.Fatalf("windows buffer %d should exceed other platforms %d", windows, other)
	}
	// The speaker buffer is split in half for the oto player. That half must stay
	// inside the decoded-ahead queue or the player underruns while waiting on decode.
	if windows/2 > speakerSampleRate.N(playbackAheadDuration) {
		t.Fatalf("windows player half %d exceeds playback ahead %d", windows/2, speakerSampleRate.N(playbackAheadDuration))
	}
}
