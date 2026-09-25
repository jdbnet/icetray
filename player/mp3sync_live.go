package player

import (
	"fmt"
	"io"
	"time"
)

// openMP3Stream waits for MP3 frame sync in the live ring buffer without discarding
// bytes until sync is found, so a failed search can retry as more data arrives.
func openMP3Stream(buf StreamBuffer, cancel <-chan struct{}) (io.Reader, error) {
	peek := make([]byte, 8192)
	for {
		select {
		case <-cancel:
			return nil, fmt.Errorf("mp3 sync cancelled")
		default:
		}

		if buf.IsClosed() && buf.AvailableData() < 3 {
			return nil, fmt.Errorf("mp3 frame sync not found before stream closed")
		}

		available := buf.AvailableData()
		if available < 3 {
			time.Sleep(preBufferPollInterval)
			continue
		}

		toPeek := available
		if toPeek > len(peek) {
			toPeek = len(peek)
		}
		if toPeek > mp3SyncSearchLimit {
			toPeek = mp3SyncSearchLimit
		}

		n, err := buf.Peek(peek[:toPeek])
		if err != nil {
			return nil, err
		}
		if n < 3 {
			time.Sleep(preBufferPollInterval)
			continue
		}

		idx := findMP3FrameStart(peek[:n])
		if idx >= 0 {
			if err := buf.Discard(idx); err != nil {
				return nil, err
			}
			return buf, nil
		}

		if n >= mp3SyncSearchLimit {
			if err := buf.Discard(1); err != nil {
				return nil, err
			}
			continue
		}

		time.Sleep(preBufferPollInterval)
	}
}
