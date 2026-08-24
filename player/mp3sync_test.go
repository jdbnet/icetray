package player

import (
	"bytes"
	"io"
	"testing"
)

func TestFindMP3FrameStart(t *testing.T) {
	data := []byte{0x00, 0x01, 0xFF, 0xFB, 0x92, 0x00}
	if got := findMP3FrameStart(data); got != 2 {
		t.Fatalf("expected sync at 2, got %d", got)
	}
	if findMP3FrameStart([]byte{0xFF, 0xFE, 0x60}) != -1 {
		t.Fatal("expected false positive 0xFFFE to be rejected")
	}
}

func TestSyncedMP3ReaderSkipsLeadingGarbage(t *testing.T) {
	payload := []byte{0xFF, 0xFB, 0x92, 0x00, 0x01, 0x02}
	input := append([]byte{0xAA, 0xBB, 0xCC}, payload...)
	r, err := newSyncedMP3Reader(bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, payload) {
		t.Fatalf("expected %v, got %v", payload, out)
	}
}
