package deduplication

import (
	"regexp"
	"time"
)

// NewDeduplicationManager creates a new deduplication manager with the given config
func NewDeduplicationManager(config Config) *DeduplicationManager {
	return NewDeduplicationManagerWithTimeProvider(config, &RealTimeProvider{})
}

// NewDeduplicationManagerWithTimeProvider creates a manager with custom TimeProvider
func NewDeduplicationManagerWithTimeProvider(config Config, timeProvider TimeProvider) *DeduplicationManager {
	dm := &DeduplicationManager{
		processedPrompts: make(map[string]ProcessedEntry),
		cooldownStates:   make(map[string]CooldownState),
		config:           config,
		ansiRegex:        regexp.MustCompile(`\x1b\[[0-9;?]*[mKHJhlABCDEFGPST]`),
		stopCleanup:      make(chan struct{}),
		timeProvider:     timeProvider,
	}

	// Start periodic cleanup if cleanup interval is set
	if config.CleanupInterval > 0 {
		dm.cleanupTicker = timeProvider.NewTicker(config.CleanupInterval)
		go dm.periodicCleanup()
	}

	return dm
}

// NewDefaultDeduplicationManager creates a manager with default configuration
func NewDefaultDeduplicationManager() *DeduplicationManager {
	return NewDeduplicationManager(DefaultConfig())
}

// Close stops the cleanup goroutine and releases resources
func (dm *DeduplicationManager) Close() {
	if dm.cleanupTicker != nil {
		dm.cleanupTicker.Stop()
		close(dm.stopCleanup)
	}
}

// StripAnsi removes ANSI escape sequences from a string
func (dm *DeduplicationManager) StripAnsi(s string) string {
	return dm.ansiRegex.ReplaceAllString(s, "")
}

// ShouldProcessPrompt determines if a prompt should be processed based on deduplication rules
func (dm *DeduplicationManager) ShouldProcessPrompt(prompt string) bool {
	cleanPrompt := dm.StripAnsi(prompt)
	
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	// Check if we've already processed this exact prompt recently
	if entry, exists := dm.processedPrompts[cleanPrompt]; exists {
		if dm.timeProvider.Now().Sub(entry.ProcessedAt) < time.Duration(dm.config.PromptDuplicationSeconds)*time.Second {
			return false
		}
		// If more than specified time has passed, allow reprocessing
		delete(dm.processedPrompts, cleanPrompt)
	}

	return true
}

// ShouldProcessWithCooldown checks both prompt deduplication and cooldown state
func (dm *DeduplicationManager) ShouldProcessWithCooldown(prompt string, cooldownKey string) bool {
	if !dm.ShouldProcessPrompt(prompt) {
		return false
	}

	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	// Check cooldown state for the given key
	if state, exists := dm.cooldownStates[cooldownKey]; exists {
		if state.JustShown && dm.timeProvider.Now().Before(state.CooldownUntil) {
			return false
		}
	}

	return true
}

// MarkPromptProcessed marks a prompt as processed with current timestamp
func (dm *DeduplicationManager) MarkPromptProcessed(prompt string) {
	cleanPrompt := dm.StripAnsi(prompt)
	
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	// Update or create processed entry
	if entry, exists := dm.processedPrompts[cleanPrompt]; exists {
		entry.Count++
		entry.ProcessedAt = dm.timeProvider.Now()
		dm.processedPrompts[cleanPrompt] = entry
	} else {
		dm.processedPrompts[cleanPrompt] = ProcessedEntry{
			ProcessedAt: dm.timeProvider.Now(),
			Count:       1,
		}
	}

	// Clean up if we have too many entries
	if len(dm.processedPrompts) > dm.config.MaxEntries {
		dm.cleanupExpiredEntries()
	}
}

// SetCooldown sets a cooldown state for a specific key
func (dm *DeduplicationManager) SetCooldown(key string, duration time.Duration) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	now := dm.timeProvider.Now()
	dm.cooldownStates[key] = CooldownState{
		LastProcessed: now,
		JustShown:     true,
		CooldownUntil: now.Add(duration),
	}
}

// SetDialogCooldown sets cooldown using the configured dialog cooldown duration
func (dm *DeduplicationManager) SetDialogCooldown(key string) {
	duration := time.Duration(dm.config.DialogCooldownMs) * time.Millisecond
	dm.SetCooldown(key, duration)
}

// SetProcessingCooldown sets cooldown using the configured processing cooldown duration
func (dm *DeduplicationManager) SetProcessingCooldown(key string) {
	duration := time.Duration(dm.config.ProcessingCooldownMs) * time.Millisecond
	dm.SetCooldown(key, duration)
}

// ClearCooldown removes cooldown state for a specific key
func (dm *DeduplicationManager) ClearCooldown(key string) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	delete(dm.cooldownStates, key)
}

// GetStats returns statistics about the deduplication manager
func (dm *DeduplicationManager) GetStats() (processedCount, cooldownCount int) {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	return len(dm.processedPrompts), len(dm.cooldownStates)
}

// ClearExpiredEntries removes expired entries from both processed prompts and cooldown states
func (dm *DeduplicationManager) ClearExpiredEntries() {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	dm.cleanupExpiredEntries()
}

// cleanupExpiredEntries removes expired entries (must be called with mutex held)
func (dm *DeduplicationManager) cleanupExpiredEntries() {
	now := dm.timeProvider.Now()
	duplicationThreshold := time.Duration(dm.config.PromptDuplicationSeconds) * time.Second
	cooldownThreshold := time.Duration(dm.config.ProcessingCooldownMs) * time.Millisecond

	// Clean up processed prompts
	for prompt, entry := range dm.processedPrompts {
		if now.Sub(entry.ProcessedAt) > duplicationThreshold {
			delete(dm.processedPrompts, prompt)
		}
	}

	// Clean up cooldown states
	for key, state := range dm.cooldownStates {
		if now.After(state.CooldownUntil) && now.Sub(state.LastProcessed) > cooldownThreshold {
			delete(dm.cooldownStates, key)
		}
	}
}

// periodicCleanup runs periodic cleanup in a separate goroutine
func (dm *DeduplicationManager) periodicCleanup() {
	for {
		select {
		case <-dm.cleanupTicker.C():
			dm.ClearExpiredEntries()
		case <-dm.stopCleanup:
			return
		}
	}
}

// GetProcessedPrompts returns a copy of processed prompts for testing/debugging
func (dm *DeduplicationManager) GetProcessedPrompts() map[string]ProcessedEntry {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	result := make(map[string]ProcessedEntry)
	for k, v := range dm.processedPrompts {
		result[k] = v
	}
	return result
}

// GetCooldownStates returns a copy of cooldown states for testing/debugging
func (dm *DeduplicationManager) GetCooldownStates() map[string]CooldownState {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	result := make(map[string]CooldownState)
	for k, v := range dm.cooldownStates {
		result[k] = v
	}
	return result
}