package player

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"git.jdbnet.co.uk/jamie/icetray/logger"
)

// Player manages the mpv subprocess and IPC communication.
type Player struct {
	cmd        *exec.Cmd
	proc       *os.Process
	conn       net.Conn
	socketPath string
	mu         sync.RWMutex
	isRunning  bool
}

// NewPlayer creates a new Player instance.
func NewPlayer() *Player {
	return &Player{
		isRunning: false,
	}
}

// getSocketPath returns the platform-specific IPC socket path.
func (p *Player) getSocketPath() string {
	pid := os.Getpid()
	if os.PathSeparator == '\\' {
		// Windows: use named pipe
		return `\\.\pipe\icetray-` + strconv.Itoa(pid)
	}
	// Linux: use Unix socket in /tmp
	return filepath.Join(os.TempDir(), fmt.Sprintf("icetray-%d.sock", pid))
}

// findMpv searches for the mpv executable in PATH.
func findMpv() (string, error) {
	path, err := exec.LookPath("mpv")
	if err != nil {
		return "", fmt.Errorf("mpv not found in PATH: %w", err)
	}
	return path, nil
}

// Play starts playing an audio stream with the given URL.
func (p *Player) Play(streamURL string) error {
	p.mu.Lock()
	if p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("player is already running")
	}
	p.mu.Unlock()

	// Find mpv executable
	mpvPath, err := findMpv()
	if err != nil {
		logger.LogError("mpv not found", err)
		return err
	}

	// Clean up any previous socket
	p.cleanupSocket()

	// Get socket path
	p.socketPath = p.getSocketPath()

	// Prepare mpv command
	p.cmd = exec.Command(mpvPath,
		"--input-ipc-server="+p.socketPath,
		"--no-video",
		"--quiet",
		streamURL,
	)

	// Start the process
	if err := p.cmd.Start(); err != nil {
		logger.LogError("failed to start mpv", err)
		return err
	}

	p.proc = p.cmd.Process

	p.mu.Lock()
	p.isRunning = true
	p.mu.Unlock()

	logger.Log("Play: started mpv for stream " + streamURL)

	// Wait for the socket to become available (max 2 seconds)
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if err := p.connectSocket(); err == nil {
			logger.Log("Play: connected to mpv IPC socket")
			break
		}
	}

	return nil
}

// connectSocket establishes a connection to the mpv IPC socket.
func (p *Player) connectSocket() error {
	if os.PathSeparator == '\\' {
		// Windows: connect to named pipe
		conn, err := net.Dial("pipe", p.socketPath)
		if err != nil {
			return err
		}
		p.conn = conn
	} else {
		// Linux: connect to Unix socket
		conn, err := net.Dial("unix", p.socketPath)
		if err != nil {
			return err
		}
		p.conn = conn
	}
	return nil
}

// sendIPC sends a command to mpv over the IPC socket.
func (p *Player) sendIPC(command interface{}) error {
	p.mu.RLock()
	if !p.isRunning || p.conn == nil {
		p.mu.RUnlock()
		return fmt.Errorf("player not running or not connected")
	}
	conn := p.conn
	p.mu.RUnlock()

	data, err := json.Marshal(map[string]interface{}{
		"command": command,
	})
	if err != nil {
		return err
	}

	_, err = conn.Write(append(data, '\n'))
	return err
}

// Pause pauses the playback.
func (p *Player) Pause() error {
	err := p.sendIPC([]interface{}{"set_property", "pause", true})
	if err == nil {
		logger.Log("Pause: paused playback")
	} else {
		logger.LogError("Pause: failed to send pause command", err)
	}
	return err
}

// Resume resumes playback.
func (p *Player) Resume() error {
	err := p.sendIPC([]interface{}{"set_property", "pause", false})
	if err == nil {
		logger.Log("Resume: resumed playback")
	} else {
		logger.LogError("Resume: failed to send resume command", err)
	}
	return err
}

// Stop stops the playback and terminates the mpv process.
func (p *Player) Stop() error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()

	// Close the IPC connection
	p.closeConnection()

	// Terminate the mpv process
	if p.proc != nil {
		p.proc.Kill()
		p.proc.Wait()
	}

	p.mu.Lock()
	p.isRunning = false
	p.cmd = nil
	p.proc = nil
	p.mu.Unlock()

	logger.Log("Stop: stopped playback and terminated mpv")

	// Clean up socket
	p.cleanupSocket()

	return nil
}

// SetVolume sets the volume (0-100) and sends it to mpv.
func (p *Player) SetVolume(vol int) error {
	if vol < 0 {
		vol = 0
	} else if vol > 100 {
		vol = 100
	}

	// Convert to mpv volume range (0-130)
	mpvVol := (vol * 130) / 100

	err := p.sendIPC([]interface{}{"set_property", "volume", mpvVol})
	if err == nil {
		logger.Log(fmt.Sprintf("Volume: set to %d", vol))
	} else {
		logger.LogError(fmt.Sprintf("Volume: failed to set volume to %d", vol), err)
	}
	return err
}

// IsRunning returns whether the player is currently running.
func (p *Player) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// closeConnection closes the IPC connection.
func (p *Player) closeConnection() {
	p.mu.Lock()
	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
	}
	p.mu.Unlock()
}

// cleanupSocket removes the socket file if it exists (Unix only).
func (p *Player) cleanupSocket() {
	if os.PathSeparator != '\\' && p.socketPath != "" {
		os.Remove(p.socketPath)
	}
}

// Close stops the player and cleans up resources.
func (p *Player) Close() error {
	p.Stop()
	p.closeConnection()
	p.cleanupSocket()
	return nil
}
