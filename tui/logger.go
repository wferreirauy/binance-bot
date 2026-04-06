package tui

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"
	"time"
)

// stripColorTags removes tview dynamic color tags like [red], [green::b], [-] etc.
var colorTagRegex = regexp.MustCompile(`\[([a-zA-Z]*:?:?[a-zA-Z]*)\]|\[-\]`)

func stripColorTags(s string) string {
	return colorTagRegex.ReplaceAllString(s, "")
}

// FileLogger writes log lines to a file, safe for concurrent use.
type FileLogger struct {
	mu     sync.Mutex
	file   *os.File
	logger *log.Logger
}

// NewFileLogger creates a FileLogger that writes to the given path.
// The file is opened in append mode (created if it doesn't exist).
func NewFileLogger(path string) (*FileLogger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not open log file: %w", err)
	}
	l := log.New(f, "", 0) // no prefix; we format timestamps ourselves
	return &FileLogger{file: f, logger: l}, nil
}

// Log writes a timestamped, tag-stripped line to the log file.
func (fl *FileLogger) Log(level, msg string) {
	clean := stripColorTags(msg)
	ts := time.Now().Format("2006-01-02 15:04:05")
	fl.mu.Lock()
	defer fl.mu.Unlock()
	fl.logger.Printf("%s [%s] %s", ts, level, clean)
}

// Close flushes and closes the underlying file.
func (fl *FileLogger) Close() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	return fl.file.Close()
}
