package stream

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"git.jdbnet.co.uk/jamie/icetray/logger"
	"git.jdbnet.co.uk/jamie/icetray/player"
)

// RingBuffer is a fixed-size circular byte buffer.
type RingBuffer struct {
	buffer   []byte
	size     int
	readPos  int
	writePos int
	mu       sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond
	closed   bool
}

// NewRingBuffer creates a new ring buffer with the given size.
func NewRingBuffer(size int) *RingBuffer {
	rb := &RingBuffer{
		buffer: make([]byte, size),
		size:   size,
	}
	rb.notEmpty = sync.NewCond(&rb.mu)
	rb.notFull = sync.NewCond(&rb.mu)
	return rb
}

// Write writes data to the buffer, blocking if full.
func (rb *RingBuffer) Write(data []byte) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.closed {
		return fmt.Errorf("buffer is closed")
	}

	// Write all data, blocking as needed
	for len(data) > 0 {
		// Wait until there's space
		for rb.isFull() {
			rb.notFull.Wait()
			if rb.closed {
				return fmt.Errorf("buffer is closed")
			}
		}

		// Calculate space available
		space := rb.availableSpace()
		toWrite := len(data)
		if toWrite > space {
			toWrite = space
		}

		// Handle wrap-around: write in chunks
		if rb.writePos+toWrite <= rb.size {
			// Direct write without wrap
			copy(rb.buffer[rb.writePos:], data[:toWrite])
			rb.writePos += toWrite
			if rb.writePos == rb.size {
				rb.writePos = 0
			}
		} else {
			// Write wraps around
			firstPart := rb.size - rb.writePos
			copy(rb.buffer[rb.writePos:], data[:firstPart])
			copy(rb.buffer[0:], data[firstPart:toWrite])
			rb.writePos = toWrite - firstPart
		}

		data = data[toWrite:]
		rb.notEmpty.Broadcast()
	}

	return nil
}

// Read reads bytes from the buffer, blocking if empty. It implements io.Reader.
func (rb *RingBuffer) Read(p []byte) (n int, err error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	// Wait until there's data or the buffer is closed
	for rb.isEmpty() && !rb.closed {
		rb.notEmpty.Wait()
	}

	if rb.isEmpty() {
		if rb.closed {
			return 0, io.EOF
		}
		return 0, nil
	}

	// Calculate how much data is available
	available := rb.availableData()
	toRead := len(p)
	if toRead > available {
		toRead = available
	}

	// Copy data out, handling wrap-around
	if rb.readPos+toRead <= rb.size {
		copy(p[:toRead], rb.buffer[rb.readPos:])
		rb.readPos += toRead
	} else {
		firstPart := rb.size - rb.readPos
		copy(p[:firstPart], rb.buffer[rb.readPos:])
		copy(p[firstPart:toRead], rb.buffer[0:toRead-firstPart])
		rb.readPos = toRead - firstPart
	}

	rb.notFull.Broadcast()
	return toRead, nil
}

// Close closes the buffer and signals all waiters.
func (rb *RingBuffer) Close() error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.closed = true
	rb.notEmpty.Broadcast()
	rb.notFull.Broadcast()
	return nil
}

// AvailableData returns the number of bytes available to read.
func (rb *RingBuffer) AvailableData() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.availableData()
}

// IsClosed returns whether the buffer is closed.
func (rb *RingBuffer) IsClosed() bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.closed
}

// isFull checks if the buffer is full.
func (rb *RingBuffer) isFull() bool {
	return (rb.writePos+1)%rb.size == rb.readPos
}

// isEmpty checks if the buffer is empty.
func (rb *RingBuffer) isEmpty() bool {
	return rb.readPos == rb.writePos
}

// availableSpace returns the number of bytes that can be written.
func (rb *RingBuffer) availableSpace() int {
	if rb.writePos >= rb.readPos {
		return rb.size - rb.writePos + rb.readPos - 1
	}
	return rb.readPos - rb.writePos - 1
}

// availableData returns the number of bytes available to read.
func (rb *RingBuffer) availableData() int {
	if rb.writePos >= rb.readPos {
		return rb.writePos - rb.readPos
	}
	return rb.size - rb.readPos + rb.writePos
}

// StreamReader reads from an HTTP stream into a ring buffer.
type StreamReader struct {
	url        string
	buffer     *RingBuffer
	connected  atomic.Bool
	stopChan   chan struct{}
	onConnect  func()
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewStreamReader creates a new stream reader.
func NewStreamReader(url string, bufferSize int) *StreamReader {
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamReader{
		url:        url,
		buffer:     NewRingBuffer(bufferSize),
		stopChan:   make(chan struct{}),
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

// Start begins reading from the stream.
func (sr *StreamReader) Start() error {
	req, err := http.NewRequestWithContext(sr.ctx, "GET", sr.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("stream server returned status %d", resp.StatusCode)
	}

	sr.connected.Store(true)
	defer sr.connected.Store(false)

	if sr.onConnect != nil {
		sr.onConnect()
	}

	defer sr.buffer.Close()

	// Read from the response body into the ring buffer
	buf := make([]byte, 4096)
	for {
		select {
		case <-sr.stopChan:
			return nil
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			if err := sr.buffer.Write(buf[:n]); err != nil {
				return err
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// IsConnected returns whether the stream is currently connected.
func (sr *StreamReader) IsConnected() bool {
	return sr.connected.Load()
}

// Stop stops reading from the stream.
func (sr *StreamReader) Stop() {
	if sr.cancelFunc != nil {
		sr.cancelFunc()
	}
	select {
	case <-sr.stopChan:
		// already closed
	default:
		close(sr.stopChan)
	}
	if sr.buffer != nil {
		sr.buffer.Close()
	}
}

// GetBuffer returns the underlying ring buffer.
func (sr *StreamReader) GetBuffer() *RingBuffer {
	return sr.buffer
}

// Supervisor monitors the stream and handles auto-reconnection with exponential backoff.
type Supervisor struct {
	player     *player.Player
	streamURL  string
	reader     *StreamReader
	mu         sync.RWMutex
	stopChan   chan struct{}
	isRunning  atomic.Bool
	session    atomic.Uint64
	wg         sync.WaitGroup
	backoff    time.Duration
	maxBackoff time.Duration
}

// NewSupervisor creates a new stream supervisor.
func NewSupervisor(p *player.Player) *Supervisor {
	return &Supervisor{
		player:     p,
		stopChan:   make(chan struct{}),
		backoff:    time.Second,
		maxBackoff: 30 * time.Second,
	}
}

// Start begins monitoring the stream with auto-reconnect.
func (s *Supervisor) Start(streamURL string) {
	if s.isRunning.Load() {
		s.Stop()
	}

	s.mu.Lock()
	s.streamURL = streamURL
	s.session.Add(1)
	session := s.session.Load()
	s.stopChan = make(chan struct{})
	stopChan := s.stopChan
	s.mu.Unlock()

	s.isRunning.Store(true)
	s.backoff = time.Second

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.supervise(stopChan, session)
	}()
}

// supervise is the main supervision loop.
func (s *Supervisor) supervise(stopChan chan struct{}, session uint64) {
	for {
		select {
		case <-stopChan:
			return
		default:
		}

		s.mu.RLock()
		url := s.streamURL
		s.mu.RUnlock()

		err := s.attemptConnection(url, session)
		if err != nil {
			logger.LogError("Stream connection lost", err)
		}

		select {
		case <-stopChan:
			return
		case <-time.After(s.backoff):
			s.backoff = s.backoff * 2
			if s.backoff > s.maxBackoff {
				s.backoff = s.maxBackoff
			}
			logger.Log(fmt.Sprintf("Reconnecting with backoff: %v", s.backoff))
		}
	}
}

// attemptConnection tries to connect and read from the stream.
func (s *Supervisor) attemptConnection(url string, session uint64) error {
	reader := NewStreamReader(url, 1024*1024) // 1MB buffer
	reader.onConnect = func() {
		if s.session.Load() != session {
			return
		}
		// SetSource releases the player lock before touching the speaker so we
		// never block HTTP reads while waiting on the audio thread.
		s.player.SetSource(reader.GetBuffer())
	}
	s.mu.Lock()
	s.reader = reader
	s.mu.Unlock()

	logger.Log("Attempting to connect to stream: " + url)
	err := reader.Start()

	if err == nil {
		s.backoff = time.Second
	}

	s.mu.Lock()
	if s.session.Load() == session && s.reader == reader {
		s.player.ClearSource()
		s.reader = nil
	}
	s.mu.Unlock()

	return err
}

// Stop stops the supervisor and closes the stream.
func (s *Supervisor) Stop() {
	if !s.isRunning.Swap(false) {
		return
	}

	s.session.Add(1)

	s.mu.Lock()
	stopChan := s.stopChan
	s.stopChan = nil
	reader := s.reader
	s.reader = nil
	s.mu.Unlock()

	if stopChan != nil {
		close(stopChan)
	}
	if reader != nil {
		reader.Stop()
	}

	s.wg.Wait()
	logger.Log("Stream supervisor stopped")
}

// IsRunning returns whether the supervisor is currently running.
func (s *Supervisor) IsRunning() bool {
	return s.isRunning.Load()
}
