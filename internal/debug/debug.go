package debug

import (
	"fmt"
	"os"
	"sync"
)

var (
	instance *Logger
	once     sync.Once
)

// Logger handles debug logging with singleton pattern
type Logger struct {
	enabled bool
	file    *os.File
	mutex   sync.Mutex
}

// GetLogger returns the singleton logger instance
func GetLogger() *Logger {
	once.Do(func() {
		instance = &Logger{
			enabled: false,
			file:    nil,
		}
	})
	return instance
}

// Enable turns on debug logging and creates the debug file
func (l *Logger) Enable() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	if l.enabled {
		return nil
	}
	
	file, err := os.OpenFile("debug_output.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	
	l.file = file
	l.enabled = true
	return nil
}

// Disable turns off debug logging and closes the file
func (l *Logger) Disable() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	if !l.enabled {
		return
	}
	
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}
	l.enabled = false
}

// IsEnabled returns whether debug logging is enabled
func (l *Logger) IsEnabled() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.enabled
}

// Printf writes formatted debug output if enabled
func (l *Logger) Printf(format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	if l.enabled && l.file != nil {
		fmt.Fprintf(l.file, format, args...)
	}
}

// Println writes debug output with newline if enabled
func (l *Logger) Println(args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	if l.enabled && l.file != nil {
		fmt.Fprintln(l.file, args...)
	}
}

// Package-level convenience functions
func Enable() error {
	return GetLogger().Enable()
}

func Disable() {
	GetLogger().Disable()
}

func IsEnabled() bool {
	return GetLogger().IsEnabled()
}

func Printf(format string, args ...interface{}) {
	GetLogger().Printf(format, args...)
}

func Println(args ...interface{}) {
	GetLogger().Println(args...)
}