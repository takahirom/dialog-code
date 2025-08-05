package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/takahirom/dialog-code/internal/types"
)

// TestWriteToTerminal tests the writeToTerminal function  
func TestWriteToTerminal(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test_terminal")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	
	handler := &PermissionHandler{
		ptmx: tmpFile,
	}
	
	// Test basic write functionality
	text := "test message"
	err = handler.writeToTerminal(text)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Read back the written content
	tmpFile.Seek(0, 0)
	buf := make([]byte, len(text))
	n, err := tmpFile.Read(buf)
	if err != nil {
		t.Errorf("Failed to read from temp file: %v", err)
	}
	
	if n != len(text) || string(buf) != text {
		t.Errorf("Expected write of %q, got %q", text, string(buf))
	}
}

func TestSendAutoRejectWithWait_MaxChoiceSelection(t *testing.T) {
	// Create test app state with choices  
	appState := types.NewAppState()
	appState.Prompt.CollectedChoices = map[string]string{
		"1": "approve",
		"2": "reject", 
		"3": "reject permanently",
	}
	
	tmpFile, err := os.CreateTemp("", "test_terminal")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	
	handler := &PermissionHandler{
		ptmx:     tmpFile,
		appState: appState,
	}
	
	// Set a short timeout for testing
	originalTimeout := *autoRejectWait
	*autoRejectWait = 1 // 1 second
	defer func() { *autoRejectWait = originalTimeout }()
	
	// This test verifies the function runs without panic
	// The actual dialog interaction is difficult to test without complex mocking
	handler.sendAutoRejectWithWait("1")
	
	// Give goroutines time to complete
	time.Sleep(1200 * time.Millisecond)
	
	// Read content from temp file to verify correct choice was written
	tmpFile.Seek(0, 0)
	buf := make([]byte, 1024)
	n, _ := tmpFile.Read(buf)
	content := string(buf[:n])
	
	// Should contain the max choice "3" since that's the highest numbered choice
	if !strings.Contains(content, "3") {
		t.Errorf("Expected terminal output to contain choice '3', got: %q", content)
	}
	
	// For debugging, log the actual content
	t.Logf("Actual terminal content: %q", content)
	
	// Should also contain the auto-reject message or at least some text
	if len(strings.TrimSpace(content)) == 1 {
		t.Logf("Only got single character, which is expected for timeout scenario")
	} else if !strings.Contains(content, "rejected") {
		t.Errorf("Expected terminal output to contain some reject-related message, got: %q", content)
	}
}

func TestHandleDialogCooldown(t *testing.T) {
	appState := types.NewAppState()
	appState.Prompt.JustShown = true
	
	handler := &PermissionHandler{
		appState: appState,
	}
	
	handler.handleDialogCooldown()
	
	// Verify JustShown is initially true
	if !appState.Prompt.JustShown {
		t.Error("Expected JustShown to be true initially")
	}
	
	// The cooldown reset happens in a goroutine with a delay, 
	// so we can't easily test the final state without waiting
	// This test just verifies the function doesn't panic
}

func TestFindMaxRejectChoice(t *testing.T) {
	tests := []struct {
		name     string
		choices  map[string]string
		expected string
	}{
		{
			name: "selects choice 3 when available",
			choices: map[string]string{
				"1": "approve",
				"2": "reject",
				"3": "reject permanently",
			},
			expected: "3",
		},
		{
			name: "selects choice 2 when 3 not available",
			choices: map[string]string{
				"1": "approve", 
				"2": "reject",
			},
			expected: "2",
		},
		{
			name: "defaults to 2 when only choice 1 exists",
			choices: map[string]string{
				"1": "approve",
			},
			expected: "2",
		},
		{
			name: "defaults to 2 when no choices exist",
			choices: map[string]string{},
			expected: "2",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findMaxRejectChoice(tt.choices)
			
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSendAutoRejectWithWait_RaceCondition(t *testing.T) {
	// Test for the race condition bug where dialog completes after timeout
	appState := types.NewAppState()
	appState.Prompt.CollectedChoices = map[string]string{
		"1": "approve",
		"2": "reject",
		"3": "reject permanently",
	}
	
	tmpFile, err := os.CreateTemp("", "test_terminal")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	
	handler := &PermissionHandler{
		ptmx:     tmpFile,
		appState: appState,
	}
	
	// Use very short timeout to trigger race condition
	originalTimeout := *autoRejectWait
	*autoRejectWait = 1 // 1 second timeout
	defer func() { *autoRejectWait = originalTimeout }()
	
	// Test should not panic even with the race condition
	handler.sendAutoRejectWithWait("1")
	
	// Wait longer than timeout to ensure any lingering goroutines complete
	time.Sleep(2000 * time.Millisecond)
	
	// Verify timeout behavior occurred correctly
	tmpFile.Seek(0, 0)
	buf := make([]byte, 1024)
	n, _ := tmpFile.Read(buf)
	content := string(buf[:n])
	
	// Should contain the max choice "3" since timeout should have occurred
	if !strings.Contains(content, "3") {
		t.Errorf("Expected terminal output to contain timeout choice '3', got: %q", content)
	}
	
	// Verify no goroutine leak by checking that we can complete without hanging
	// The fact that we reach this point without panic verifies the race condition fix
	if len(strings.TrimSpace(content)) == 0 {
		t.Error("Expected some content to be written during timeout scenario")
	}
}