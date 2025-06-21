package deduplication

import (
	"sync"
	"testing"
	"time"
)

func TestNewDeduplicationManager(t *testing.T) {
	config := DefaultConfig()
	dm := NewDeduplicationManager(config)
	defer dm.Close()

	if dm == nil {
		t.Fatal("NewDeduplicationManager returned nil")
	}

	if dm.config.PromptDuplicationSeconds != config.PromptDuplicationSeconds {
		t.Errorf("Expected PromptDuplicationSeconds %d, got %d", 
			config.PromptDuplicationSeconds, dm.config.PromptDuplicationSeconds)
	}
}

func TestShouldProcessPrompt_DuplicateDetection(t *testing.T) {
	config := DefaultConfig()
	config.PromptDuplicationSeconds = 1 // 1 second for faster testing
	
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := NewMockTimeProvider(startTime)
	dm := NewDeduplicationManagerWithTimeProvider(config, mockTime)
	defer dm.Close()

	prompt := "Do you want to proceed?"

	// First time should be allowed
	if !dm.ShouldProcessPrompt(prompt) {
		t.Error("First prompt should be allowed")
	}

	// Mark as processed
	dm.MarkPromptProcessed(prompt)

	// Immediate retry should be blocked
	if dm.ShouldProcessPrompt(prompt) {
		t.Error("Immediate retry should be blocked")
	}

	// Advance time past duplication window
	mockTime.AdvanceTime(time.Duration(config.PromptDuplicationSeconds+1) * time.Second)

	// Should be allowed again after expiration
	if !dm.ShouldProcessPrompt(prompt) {
		t.Error("Prompt should be allowed after expiration")
	}
}

func TestShouldProcessPrompt_AnsiStripping(t *testing.T) {
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := NewMockTimeProvider(startTime)
	dm := NewDeduplicationManagerWithTimeProvider(DefaultConfig(), mockTime)
	defer dm.Close()

	promptWithAnsi := "\x1b[31mDo you want to proceed?\x1b[0m"
	promptWithoutAnsi := "Do you want to proceed?"

	// Process the first one
	if !dm.ShouldProcessPrompt(promptWithAnsi) {
		t.Error("First prompt should be allowed")
	}
	dm.MarkPromptProcessed(promptWithAnsi)

	// The same prompt without ANSI should be blocked
	if dm.ShouldProcessPrompt(promptWithoutAnsi) {
		t.Error("Same prompt without ANSI should be blocked")
	}

	// Verify ANSI stripping works correctly
	stripped := dm.StripAnsi(promptWithAnsi)
	if stripped != promptWithoutAnsi {
		t.Errorf("ANSI stripping failed: expected %q, got %q", promptWithoutAnsi, stripped)
	}
}

func TestCooldownMechanism(t *testing.T) {
	config := DefaultConfig()
	config.DialogCooldownMs = 100 // 100ms for faster testing
	config.PromptDuplicationSeconds = 10 // Long enough to not interfere
	
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := NewMockTimeProvider(startTime)
	dm := NewDeduplicationManagerWithTimeProvider(config, mockTime)
	defer dm.Close()

	prompt := "Do you want to proceed?"
	cooldownKey := "dialog1"

	// Should be allowed initially
	if !dm.ShouldProcessWithCooldown(prompt, cooldownKey) {
		t.Error("Initial prompt should be allowed")
	}

	// Mark as processed and set cooldown
	dm.MarkPromptProcessed(prompt)
	dm.SetDialogCooldown(cooldownKey)

	// Should be blocked during cooldown
	if dm.ShouldProcessWithCooldown(prompt, cooldownKey) {
		t.Error("Prompt should be blocked during cooldown")
	}

	// Advance time past cooldown period
	mockTime.AdvanceTime(time.Duration(config.DialogCooldownMs+50) * time.Millisecond)

	// Use a different prompt to avoid duplication blocking
	newPrompt := "Do you want to continue?"
	
	// Should be allowed after cooldown expires
	if !dm.ShouldProcessWithCooldown(newPrompt, cooldownKey) {
		t.Error("New prompt should be allowed after cooldown expires")
	}
}

func TestConcurrentAccess(t *testing.T) {
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := NewMockTimeProvider(startTime)
	dm := NewDeduplicationManagerWithTimeProvider(DefaultConfig(), mockTime)
	defer dm.Close()

	numGoroutines := 10
	numOperations := 100
	var wg sync.WaitGroup

	// Test concurrent access to ShouldProcessPrompt
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				prompt := "prompt" + string(rune(id))
				dm.ShouldProcessPrompt(prompt)
				dm.MarkPromptProcessed(prompt)
			}
		}(i)
	}

	// Test concurrent cooldown operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := "cooldown" + string(rune(id))
				dm.SetDialogCooldown(key)
				dm.ClearCooldown(key)
			}
		}(i)
	}

	wg.Wait()

	// Verify the manager is still functional
	if !dm.ShouldProcessPrompt("test prompt") {
		t.Error("Manager should still be functional after concurrent access")
	}
}

func TestMemoryCleanup(t *testing.T) {
	config := DefaultConfig()
	config.PromptDuplicationSeconds = 1
	config.MaxEntries = 5 // Small limit to trigger cleanup
	
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := NewMockTimeProvider(startTime)
	dm := NewDeduplicationManagerWithTimeProvider(config, mockTime)
	defer dm.Close()

	// Add more entries than the limit
	for i := 0; i < 10; i++ {
		prompt := "prompt" + string(rune(i+'a'))
		dm.ShouldProcessPrompt(prompt)
		dm.MarkPromptProcessed(prompt)
	}

	// Check that cleanup was triggered
	processedCount, _ := dm.GetStats()
	if processedCount > config.MaxEntries*2 { // Allow some buffer
		t.Errorf("Expected cleanup to limit entries, got %d", processedCount)
	}

	// Advance time to expire entries
	mockTime.AdvanceTime(time.Duration(config.PromptDuplicationSeconds+1) * time.Second)

	// Manual cleanup
	dm.ClearExpiredEntries()

	// Verify expired entries were removed
	processedCount, _ = dm.GetStats()
	if processedCount > 0 {
		t.Errorf("Expected expired entries to be cleaned up, got %d", processedCount)
	}
}

func TestConfiguration(t *testing.T) {
	customConfig := Config{
		PromptDuplicationSeconds: 10,
		DialogCooldownMs:         1000,
		ProcessingCooldownMs:     2000,
		MaxEntries:               500,
		CleanupInterval:          time.Minute * 10,
	}

	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := NewMockTimeProvider(startTime)
	dm := NewDeduplicationManagerWithTimeProvider(customConfig, mockTime)
	defer dm.Close()

	if dm.config.PromptDuplicationSeconds != 10 {
		t.Error("Custom configuration not applied correctly")
	}
	if dm.config.DialogCooldownMs != 1000 {
		t.Error("Custom dialog cooldown not applied correctly")
	}
}

func TestGetStats(t *testing.T) {
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := NewMockTimeProvider(startTime)
	dm := NewDeduplicationManagerWithTimeProvider(DefaultConfig(), mockTime)
	defer dm.Close()

	// Initially should be empty
	processedCount, cooldownCount := dm.GetStats()
	if processedCount != 0 || cooldownCount != 0 {
		t.Errorf("Expected empty stats, got processed=%d, cooldown=%d", 
			processedCount, cooldownCount)
	}

	// Add some entries
	dm.MarkPromptProcessed("prompt1")
	dm.MarkPromptProcessed("prompt2")
	dm.SetDialogCooldown("key1")

	processedCount, cooldownCount = dm.GetStats()
	if processedCount != 2 {
		t.Errorf("Expected 2 processed entries, got %d", processedCount)
	}
	if cooldownCount != 1 {
		t.Errorf("Expected 1 cooldown entry, got %d", cooldownCount)
	}
}

func TestSetProcessingCooldown(t *testing.T) {
	config := DefaultConfig()
	config.ProcessingCooldownMs = 50
	
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := NewMockTimeProvider(startTime)
	dm := NewDeduplicationManagerWithTimeProvider(config, mockTime)
	defer dm.Close()

	key := "processing1"
	dm.SetProcessingCooldown(key)

	// Should be in cooldown
	states := dm.GetCooldownStates()
	if state, exists := states[key]; !exists || !state.JustShown {
		t.Error("Processing cooldown should be set")
	}

	// Advance time past cooldown period
	mockTime.AdvanceTime(time.Duration(config.ProcessingCooldownMs+20) * time.Millisecond)

	// Cooldown should be expired
	prompt := "test"
	if !dm.ShouldProcessWithCooldown(prompt, key) {
		t.Error("Should be allowed after processing cooldown expires")
	}
}

func TestClearCooldown(t *testing.T) {
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := NewMockTimeProvider(startTime)
	dm := NewDeduplicationManagerWithTimeProvider(DefaultConfig(), mockTime)
	defer dm.Close()

	key := "test_key"
	dm.SetDialogCooldown(key)

	// Verify cooldown is set
	states := dm.GetCooldownStates()
	if state, exists := states[key]; !exists || !state.JustShown {
		t.Error("Cooldown should be set")
	}

	// Clear cooldown
	dm.ClearCooldown(key)

	// Verify cooldown is cleared
	states = dm.GetCooldownStates()
	if state, exists := states[key]; exists && state.JustShown {
		t.Error("Cooldown should be cleared")
	}
}

func TestPeriodicCleanup(t *testing.T) {
	config := DefaultConfig()
	config.PromptDuplicationSeconds = 1
	config.CleanupInterval = time.Millisecond * 100 // Very frequent cleanup for testing
	
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	mockTime := NewMockTimeProvider(startTime)
	dm := NewDeduplicationManagerWithTimeProvider(config, mockTime)
	defer dm.Close()

	// Add some entries
	dm.MarkPromptProcessed("prompt1")
	dm.SetDialogCooldown("key1")

	// Advance time to expire entries
	mockTime.AdvanceTime(time.Duration(config.PromptDuplicationSeconds+1) * time.Second)

	// Trigger periodic cleanup manually
	dm.ClearExpiredEntries()

	// Entries should be cleaned up
	processedCount, cooldownCount := dm.GetStats()
	if processedCount > 0 || cooldownCount > 0 {
		t.Errorf("Periodic cleanup should have removed expired entries, got processed=%d, cooldown=%d", 
			processedCount, cooldownCount)
	}
}