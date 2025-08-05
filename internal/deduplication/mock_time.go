package deduplication

import (
	"time"
)

// MockTimeProvider provides controllable time for testing
type MockTimeProvider struct {
	currentTime time.Time
	sleepCalls  []time.Duration
	tickers     []*MockTicker
}

// NewMockTimeProvider creates a new mock time provider
func NewMockTimeProvider(startTime time.Time) *MockTimeProvider {
	return &MockTimeProvider{
		currentTime: startTime,
		sleepCalls:  make([]time.Duration, 0),
		tickers:     make([]*MockTicker, 0),
	}
}

// Now returns the current mock time
func (m *MockTimeProvider) Now() time.Time {
	return m.currentTime
}

// Sleep records the sleep duration but doesn't actually sleep
func (m *MockTimeProvider) Sleep(duration time.Duration) {
	m.sleepCalls = append(m.sleepCalls, duration)
	m.currentTime = m.currentTime.Add(duration)
}

// After returns a channel that will receive a value after the duration
func (m *MockTimeProvider) After(duration time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	// In mock, we immediately send the time
	go func() {
		ch <- m.currentTime.Add(duration)
	}()
	return ch
}

// NewTicker creates a new mock ticker
func (m *MockTimeProvider) NewTicker(duration time.Duration) Ticker {
	ticker := &MockTicker{
		duration: duration,
		c:        make(chan time.Time, 1),
		stopped:  false,
	}
	m.tickers = append(m.tickers, ticker)
	return ticker
}

// AdvanceTime advances the mock time by the given duration
func (m *MockTimeProvider) AdvanceTime(duration time.Duration) {
	m.currentTime = m.currentTime.Add(duration)

	// Trigger any tickers that should fire
	for _, ticker := range m.tickers {
		if !ticker.stopped {
			ticker.TriggerIfReady(m.currentTime)
		}
	}
}

// SetTime sets the mock time to a specific value
func (m *MockTimeProvider) SetTime(t time.Time) {
	m.currentTime = t
}

// GetSleepCalls returns all recorded sleep calls
func (m *MockTimeProvider) GetSleepCalls() []time.Duration {
	return m.sleepCalls
}

// MockTicker implements the Ticker interface for testing
type MockTicker struct {
	duration time.Duration
	c        chan time.Time
	stopped  bool
	lastTick time.Time
}

// C returns the ticker channel
func (mt *MockTicker) C() <-chan time.Time {
	return mt.c
}

// Stop stops the ticker
func (mt *MockTicker) Stop() {
	mt.stopped = true
	close(mt.c)
}

// TriggerIfReady sends a tick if enough time has passed
func (mt *MockTicker) TriggerIfReady(currentTime time.Time) {
	if mt.stopped {
		return
	}

	if mt.lastTick.IsZero() || currentTime.Sub(mt.lastTick) >= mt.duration {
		select {
		case mt.c <- currentTime:
			mt.lastTick = currentTime
		default:
			// Channel is full, skip this tick
		}
	}
}

// Trigger manually triggers the ticker (for testing)
func (mt *MockTicker) Trigger() {
	if !mt.stopped {
		select {
		case mt.c <- time.Now():
		default:
			// Channel is full, skip this tick
		}
	}
}
