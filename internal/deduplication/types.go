package deduplication

import (
	"regexp"
	"sync"
	"time"
)

// TimeProvider interface allows for dependency injection of time functions
type TimeProvider interface {
	Now() time.Time
	Sleep(duration time.Duration)
	After(duration time.Duration) <-chan time.Time
	NewTicker(duration time.Duration) Ticker
}

// Ticker interface wraps time.Ticker for testing
type Ticker interface {
	C() <-chan time.Time
	Stop()
}

// RealTimeProvider implements TimeProvider using real time functions
type RealTimeProvider struct{}

func (r *RealTimeProvider) Now() time.Time {
	return time.Now()
}

func (r *RealTimeProvider) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

func (r *RealTimeProvider) After(duration time.Duration) <-chan time.Time {
	return time.After(duration)
}

func (r *RealTimeProvider) NewTicker(duration time.Duration) Ticker {
	return &RealTicker{ticker: time.NewTicker(duration)}
}

// RealTicker wraps time.Ticker
type RealTicker struct {
	ticker *time.Ticker
}

func (rt *RealTicker) C() <-chan time.Time {
	return rt.ticker.C
}

func (rt *RealTicker) Stop() {
	rt.ticker.Stop()
}

// Config holds configuration parameters for deduplication
type Config struct {
	PromptDuplicationSeconds int           // Seconds to block duplicate prompts
	DialogCooldownMs         int           // Milliseconds for dialog cooldown
	ProcessingCooldownMs     int           // Milliseconds for processing cooldown
	MaxEntries               int           // Maximum entries to keep in memory
	CleanupInterval          time.Duration // Interval for cleaning up expired entries
}

// DefaultConfig returns default configuration values
func DefaultConfig() Config {
	return Config{
		PromptDuplicationSeconds: 5,
		DialogCooldownMs:         500,
		ProcessingCooldownMs:     500,
		MaxEntries:               1000,
		CleanupInterval:          time.Minute * 5,
	}
}

// CooldownState tracks cooldown information for a specific key
type CooldownState struct {
	LastProcessed time.Time
	JustShown     bool
	CooldownUntil time.Time
}

// ProcessedEntry tracks when a prompt was processed
type ProcessedEntry struct {
	ProcessedAt time.Time
	Count       int // How many times this prompt has been processed
}

// DeduplicationManager manages prompt deduplication and cooldown logic
type DeduplicationManager struct {
	processedPrompts map[string]ProcessedEntry
	cooldownStates   map[string]CooldownState
	config           Config
	mutex            sync.RWMutex
	ansiRegex        *regexp.Regexp
	cleanupTicker    Ticker
	stopCleanup      chan struct{}
	timeProvider     TimeProvider
}
