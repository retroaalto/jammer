// Package log provides simple file-based logging for jammer.
//
// Log output goes to ~/jammer/jammer.log.  The TUI uses the alternate screen,
// so writing to stderr would be invisible; a dedicated log file is the only
// reliable way to observe runtime behaviour.
//
// Usage:
//
//	log.Init()          // call once at startup (creates/appends to the file)
//	log.Info("msg")
//	log.Infof("key=%s val=%d", k, v)
//	log.Error("something went wrong:", err)
//	log.Close()         // flush and close on exit
package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	mu   sync.Mutex
	file *os.File
)

// Init opens (or creates) the log file at ~/jammer/jammer.log.
// Safe to call more than once; subsequent calls are no-ops.
func Init() error {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, "jammer")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "jammer.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	file = f
	writeRaw("─── jammer started ─────────────────────────────────────────────── " +
		time.Now().Format("2006-01-02 15:04:05") + "\n")
	return nil
}

// Close flushes and closes the log file.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		writeRaw("─── jammer stopped ─────────────────────────────────────────────── " +
			time.Now().Format("2006-01-02 15:04:05") + "\n\n")
		file.Close()
		file = nil
	}
}

// Info logs a message at INFO level.
func Info(args ...any) {
	write("INFO", fmt.Sprint(args...))
}

// Infof logs a formatted message at INFO level.
func Infof(format string, args ...any) {
	write("INFO", fmt.Sprintf(format, args...))
}

// Error logs a message at ERROR level.
func Error(args ...any) {
	write("ERRO", fmt.Sprint(args...))
}

// Errorf logs a formatted message at ERROR level.
func Errorf(format string, args ...any) {
	write("ERRO", fmt.Sprintf(format, args...))
}

// Key logs a keypress event.
func Key(key, context string) {
	write("KEY ", fmt.Sprintf("[%s] view=%s", key, context))
}

func write(level, msg string) {
	mu.Lock()
	defer mu.Unlock()
	if file == nil {
		return
	}
	ts := time.Now().Format("15:04:05.000")
	writeRaw(fmt.Sprintf("%s %s  %s\n", ts, level, msg))
}

func writeRaw(s string) {
	// file is already locked by caller
	_, _ = file.WriteString(s)
}
