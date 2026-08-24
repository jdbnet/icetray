package player

import (
	"bytes"
	"fmt"
	"io"
)

// isMP3FrameHeader reports whether b0/b1 look like a valid MPEG-1/2 Layer III frame header.
func isMP3FrameHeader(b0, b1, b2 byte) bool {
	if b0 != 0xFF || b1&0xE0 != 0xE0 {
		return false
	}
	version := (b1 >> 3) & 3
	layer := (b1 >> 1) & 3
	if version == 1 || layer != 1 {
		return false
	}
	if b2 == 0 || b2 == 0xFF {
		return false
	}
	return true
}

// findMP3FrameStart returns the index of the first MPEG audio Layer III frame sync.
func findMP3FrameStart(b []byte) int {
	for i := 0; i < len(b)-2; i++ {
		if isMP3FrameHeader(b[i], b[i+1], b[i+2]) {
			return i
		}
	}
	return -1
}

const mp3SyncSearchLimit = 64 * 1024

// syncedMP3Reader skips leading bytes until the first MP3 frame header so live
// Icecast joins mid-stream can still be decoded.
type syncedMP3Reader struct {
	r io.Reader
}

func newSyncedMP3Reader(r io.Reader) (*syncedMP3Reader, error) {
	synced, err := findMP3SyncReader(r)
	if err != nil {
		return nil, err
	}
	return &syncedMP3Reader{r: synced}, nil
}

func findMP3SyncReader(r io.Reader) (io.Reader, error) {
	var searched int
	var carry []byte
	chunk := make([]byte, 4096)

	for searched < mp3SyncSearchLimit {
		n, err := r.Read(chunk)
		if n > 0 {
			data := append(carry, chunk[:n]...)
			idx := findMP3FrameStart(data)
			if idx >= 0 {
				rest := data[idx:]
				return io.MultiReader(bytes.NewReader(rest), r), nil
			}
			searched += len(data)
			if len(data) > 0 {
				carry = data[len(data)-1:]
			} else {
				carry = nil
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("mp3 frame sync not found in %d bytes", searched)
			}
			return nil, err
		}
	}
	return nil, fmt.Errorf("mp3 frame sync not found within %d bytes", mp3SyncSearchLimit)
}

func (s *syncedMP3Reader) Read(p []byte) (int, error) {
	return s.r.Read(p)
}
