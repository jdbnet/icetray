package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logFile *os.File
	mu      sync.Mutex
)

// Init initializes the logger with a file in the given config directory.
func Init(configDir string) error {
	mu.Lock()
	defer mu.Unlock()

	logPath := filepath.Join(configDir, "icetray.log")
	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	return nil
}

// Log logs a message to the file.
func Log(msg string) {
	mu.Lock()
	defer mu.Unlock()

	if logFile == nil {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(logFile, "[%s] %s\n", timestamp, msg)
}

// LogError logs an error message to the file.
func LogError(msg string, err error) {
	mu.Lock()
	defer mu.Unlock()

	if logFile == nil {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(logFile, "[%s] ERROR: %s: %v\n", timestamp, msg, err)
}

// LogFatal logs a fatal error message to the file and exits with code 1.
func LogFatal(msg string, err error) {
	mu.Lock()
	defer mu.Unlock()

	if logFile == nil {
		fmt.Fprintf(os.Stderr, "FATAL: %s: %v\n", msg, err)
	} else {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(logFile, "[%s] FATAL: %s: %v\n", timestamp, msg, err)
		logFile.Close()
	}
	os.Exit(1)
}

// Close closes the log file.
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		return logFile.Close()
	}
	return nil
}
