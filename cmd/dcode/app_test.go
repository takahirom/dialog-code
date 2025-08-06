package main

import (
	"strings"
	"testing"
	"time"
)

func TestAppWithDialogIntegration(t *testing.T) {
	// Use actual dialog data that includes pre-dialog Claude output
	realDialogLines := []string{
		"⏺ Bash(rm not-found-file)",
		"  ⎿  Running hook PreToolUse:Bash...",
		"  ⎿  Running…",
		"",
		"╭─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮",
		"│ Bash command                                                                                                                                            │",
		"│                                                                                                                                                         │",
		"│   rm not-found-file                                                                                                                                     │",
		"│   Test dialog message for data collection                                                                                                               │",
		"│                                                                                                                                                         │",
		"│ Do you want to proceed?                                                                                                                                 │",
		"│ ❯ 1. Yes                                                                                                                                                │",
		"│   2. Yes, and don't ask again for rm commands in /Users/test/git/dialog-code                                                                          │",
		"│   3. No, and tell Claude what to do differently (esc)                                                                                                   │",
		"╰─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯",
	}

	robot := NewAppRobot(t).
		ReceiveClaudeText(realDialogLines...).
		AssertDialogCaptured().
		PrintCapturedMessage().
		AssertDialogTextContains("Do you want to proceed?").
		AssertDialogTextContains("Test dialog message for data collection").
		AssertButtonCount(3).
		AssertButton(0, "Yes").
		AssertButton(1, "Yes, and don't ask again for rm commands in /Users/test/git/dialog-code").
		AssertButton(2, "No, and tell Claude what to do differently (esc)").
		AssertDialogTextContains("Bash command").
		AssertDialogTextContains("rm not-found-file").
		AssertDialogTextContains("⏺ Bash(rm not-found-file)")

	// Example of exact matching (note: includes timestamp so usually not practical)
	capturedMessage := robot.GetCapturedMessage()
	t.Logf("Complete captured message: %q", capturedMessage)

	// For exact matching without timestamp, you'd need to strip the timestamp part
	// This is usually too brittle for real tests, so AssertDialogTextContains is preferred
}

func TestAppWithEditDialog(t *testing.T) {
	// Test with valid Edit dialog using Robot pattern
	editDialogLines := []string{
		"╭─────────────────────────────────────────────────────────────────╮",
		"│ Edit command                                                    │",
		"│                                                                 │",
		"│   file_path: /test/file.txt                                     │",
		"│   Edit content here                                             │",
		"│                                                                 │",
		"│ Do you want to proceed?                                         │",
		"╰─────────────────────────────────────────────────────────────────╯",
	}

	NewAppRobot(t).
		ReceiveClaudeText(editDialogLines...).
		AssertDialogCaptured().
		AssertDialogTextContains("Edit command").
		AssertDialogTextContains("/test/file.txt")

	t.Logf("Edit dialog test passed")
}

func TestAppTaskDialogFlow(t *testing.T) {
	// Test the complete flow: Claude output → context collection → dialog
	taskDialogLines := []string{
		"╭─────────────────────────────────────────────────────────────────╮",
		"│ Task                                                            │",
		"│                                                                 │",
		"│   description: Test complex task                                │",
		"│   prompt: Execute dangerous operation                           │",
		"│                                                                 │",
		"│ Do you want to proceed?                                         │",
		"│ ❯ 1. Yes                                                        │",
		"│   2. No                                                         │",
		"╰─────────────────────────────────────────────────────────────────╯",
	}

	NewAppRobot(t).
		SetDialogChoice("3"). // Choose "No" option
		ReceiveClaudeText(taskDialogLines...).
		AssertDialogCaptured().
		AssertDialogTextContains("Do you want to proceed?").
		AssertButtonCount(2).
		AssertButton(0, "Yes").
		AssertButton(1, "No").
		AssertDialogTextContains("Task").
		AssertDialogTextContains("Execute dangerous operation").
		LogDebugInfo()

	t.Logf("Task dialog test passed")
}

func TestDialogExactMatch(t *testing.T) {
	realDialogLines := []string{
		"⏺ Bash(rm test-file)",
		"  ⎿  Running hook PreToolUse:Bash...",
		"  ⎿  Running…",
		"",
		"╭─────────────────────────────────────────────────────────────────────────────╮",
		"│ Bash command                                                                │",
		"│                                                                             │",
		"│   rm test-file                                                              │",
		"│   Remove test file                                                          │",
		"│                                                                             │",
		"│ Do you want to proceed?                                                     │",
		"│ ❯ 1. Yes                                                                    │",
		"│   2. No                                                                     │",
		"╰─────────────────────────────────────────────────────────────────────────────╯",
	}

	robot := NewAppRobot(t).
		ReceiveClaudeText(realDialogLines...).
		AssertDialogCaptured()

	// Test the new clean message format (without Context header and with organized structure)
	expectedMessage := `Trigger text: ⏺ Bash(rm test-file)
Trigger timestamp: 1672574400000000000
Reason: Bash command execution
───────────────────────────────────
Bash command

  rm test-file
  Remove test file

Do you want to proceed?`
	robot.AssertExactFormatSnapshotTest(expectedMessage)
}

func TestRealWorldDialogData_TriggerTextMissing(t *testing.T) {
	// This test reproduces the issue where Trigger text and Reason are missing
	// when using real dialog data from test_data.txt

	realWorldDialogLines := []string{
		"⏺ Bash(rm not-found-file)",
		"  ⎿  Running hook PreToolUse:Bash...",
		"  ⎿  Running…",
		"",
		"╭─────────────────────────────────────────────────────────────────────────────╮",
		"│ Bash command                                                                │",
		"│                                                                             │",
		"│   rm not-found-file                                                         │",
		"│   Test dialog message for data collection                                   │",
		"│                                                                             │",
		"│ Do you want to proceed?                                                     │",
		"│ ❯ 1. Yes                                                                    │",
		"│   2. Yes, and don't ask again for rm commands in /Users/test/git/dialog-code │",
		"│   3. No, and tell Claude what to do differently (esc)                       │",
		"╰─────────────────────────────────────────────────────────────────────────────╯",
	}

	robot := NewAppRobot(t).
		ReceiveClaudeText(realWorldDialogLines...).
		AssertDialogCaptured()

	// Get the actual captured message
	actualMessage := robot.GetCapturedMessage()
	t.Logf("ACTUAL MESSAGE:\n%s", actualMessage)

	// Current problem: Missing Trigger text and Reason
	// Expected: Should contain "Trigger text: ⏺ Bash(rm not-found-file)"
	// Expected: Should contain "Reason: Bash command execution" (or similar)

	// This test should FAIL until we fix the issue
	expectedMessage := `Trigger text: ⏺ Bash(rm not-found-file)
Trigger timestamp: 1672574400000000000
Reason: Bash command execution
───────────────────────────────────
Bash command

  rm not-found-file
  Test dialog message for data collection

Do you want to proceed?`

	// This assertion should fail, demonstrating the problem
	if actualMessage == expectedMessage {
		t.Log("✅ Dialog format is correct!")
	} else {
		t.Errorf("❌ Dialog format is incorrect!\n\nExpected:\n%s\n\nGot:\n%s\n\n🔍 Problem: Missing Trigger text and/or Reason in actual output", expectedMessage, actualMessage)
	}
}

func TestPipeCharacterCleanup(t *testing.T) {
	// Test case where context doesn't contain ⏺ and triggerLine has pipe characters
	// This reproduces the issue where "Trigger text: │ Do you want to proceed?" appears

	dialogLinesWithoutTrigger := []string{
		"╭─────────────────────────────────────────────────────────────────────────────╮",
		"│ Bash command                                                                │",
		"│                                                                             │",
		"│   rm no-file                                                                │",
		"│   Remove file named 'no-file'                                               │",
		"│                                                                             │",
		"│ Do you want to proceed?                                                     │",
		"│ ❯ 1. Yes                                                                    │",
		"│   2. No                                                                     │",
		"╰─────────────────────────────────────────────────────────────────────────────╯",
	}

	robot := NewAppRobot(t).
		ReceiveClaudeText(dialogLinesWithoutTrigger...).
		AssertDialogCaptured()

	// Get the actual captured message
	actualMessage := robot.GetCapturedMessage()
	t.Logf("ACTUAL MESSAGE WITH MISSING TRIGGER:\n%s", actualMessage)

	// Check if pipe characters appear in the output
	if strings.Contains(actualMessage, "│") {
		t.Errorf("❌ Pipe characters found in dialog message!\nMessage: %s", actualMessage)
	}

	// Check if triggerLine fallback creates incorrect trigger text
	if strings.Contains(actualMessage, "Trigger text: │") {
		t.Errorf("❌ Incorrect trigger text with pipe character!\nMessage: %s", actualMessage)
	}
}

func TestRealWorldPipeCharacterIssue(t *testing.T) {
	// Test case that reproduces the exact issue user reported
	// where pipe characters appear in command details

	realIssueLines := []string{
		"╭─────────────────────────────────────────────────────────────────────────────╮",
		"│ Bash command                                                                │",
		"│                                                                             │",
		"│   rm no-file                                                                │",
		"│   Remove file named 'no-file'                                               │",
		"│                                                                             │",
		"│ Do you want to proceed?                                                     │",
		"│ ❯ 1. Yes                                                                    │",
		"│   2. No                                                                     │",
		"╰─────────────────────────────────────────────────────────────────────────────╯",
	}

	robot := NewAppRobot(t).
		ReceiveClaudeText(realIssueLines...).
		AssertDialogCaptured()

	// Get the actual captured message
	actualMessage := robot.GetCapturedMessage()
	t.Logf("REAL WORLD ISSUE MESSAGE:\n%s", actualMessage)

	// Check if any pipe characters appear anywhere in the message
	if strings.Contains(actualMessage, "│") {
		t.Errorf("❌ Pipe characters still found in message!\nFull message:\n%s", actualMessage)
	}

	// Check specific problematic patterns from user report
	if strings.Contains(actualMessage, "Bash command                                                                │") {
		t.Errorf("❌ Command type line contains pipe characters!\nMessage: %s", actualMessage)
	}
}

func TestCountdownMessagePositionWithAppRobot(t *testing.T) {
	// Test that countdown message appears at the top using AppRobot pattern
	// This test verifies the UX improvement: "This will auto-reject in X seconds..." should appear at dialog top
	realDialogLines := []string{
		"⏺ Bash(rm test-file)",
		"  ⎿  Running hook PreToolUse:Bash...",
		"  ⎿  Running…",
		"",
		"╭─────────────────────────────────────────────────────────────────────────────╮",
		"│ Bash command                                                                │",
		"│                                                                             │",
		"│   rm test-file                                                              │",
		"│   Remove test file                                                          │",
		"│                                                                             │",
		"│ Do you want to proceed?                                                     │",
		"│ ❯ 1. Yes                                                                    │",
		"│   2. No                                                                     │",
		"╰─────────────────────────────────────────────────────────────────────────────╯",
	}

	// Store original timeout to restore later
	originalTimeout := *autoRejectWait

	robot := NewAppRobot(t).
		SetAutoRejectWait(5).
		ReceiveClaudeText(realDialogLines...).
		AssertDialogCaptured().
		TriggerAutoReject("1").
		RestoreAutoRejectWait(originalTimeout)

	// Get the captured message from auto-reject dialog
	capturedMessage := robot.GetCapturedMessage()
	t.Logf("Captured auto-reject message: %q", capturedMessage)

	// Verify countdown message appears at the top
	expectedStart := "This will auto-reject in 5 seconds..."
	if !strings.HasPrefix(capturedMessage, expectedStart) {
		t.Errorf("Expected dialog to start with %q, but got: %q", expectedStart, capturedMessage)
	}

	// Verify the format includes proper spacing
	expectedPattern := "This will auto-reject in 5 seconds...\n\nTrigger text:"
	if !strings.HasPrefix(capturedMessage, expectedPattern) {
		t.Errorf("Expected proper formatting with countdown at top, but got: %q", capturedMessage)
	}

	t.Logf("Countdown message correctly positioned at top")
}

func TestAutoRejectMessageWithFlag(t *testing.T) {
	// Test AutoRejectMessage content using --auto-reject flag (no cheating!)
	realDialogLines := []string{
		"⏺ Bash(rm dangerous-file)",
		"  ⎿  Running hook PreToolUse:Bash...",
		"  ⎿  Running…",
		"",
		"╭─────────────────────────────────────────────────────────────────────────────╮",
		"│ Bash command                                                                │",
		"│                                                                             │",
		"│   rm dangerous-file                                                         │",
		"│   Remove dangerous file                                                     │",
		"│                                                                             │",
		"│ Do you want to proceed?                                                     │",
		"│ ❯ 1. Yes                                                                    │",
		"│   2. No                                                                     │",
		"╰─────────────────────────────────────────────────────────────────────────────╯",
	}

	// Store original flag value to restore later
	originalAutoReject := *autoReject
	defer func() { *autoReject = originalAutoReject }()

	// Enable --auto-reject flag to trigger automatic rejection
	*autoReject = true

	robot := NewAppRobot(t).
		ReceiveClaudeText(realDialogLines...)

	// Wait for auto-reject goroutines to complete
	// AutoRejectProcessDelayMs = 500, AutoRejectChoiceDelayMs = 500, AutoRejectCRDelayMs = 400
	time.Sleep(1500 * time.Millisecond) // Wait for all delays (1400ms + buffer)
	
	// Test terminal output contains AutoRejectMessage content
	terminalOutput := robot.GetTerminalOutput()
	t.Logf("Terminal output length: %d, content: %q", len(terminalOutput), terminalOutput)

	// Verify AutoRejectMessage content appears in terminal output
	robot.AssertTerminalContains("automatically rejected").
		AssertTerminalContains("Task tools").
		AssertTerminalContains("restart")

	t.Logf("AutoRejectMessage correctly sent via --auto-reject flag")
}

func TestAutoRejectMessageWithCommandDetails(t *testing.T) {
	// Test that AutoRejectMessage includes rejected command details
	realDialogLines := []string{
		"⏺ Bash(rm dangerous-file)",
		"  ⎿  Running hook PreToolUse:Bash...",
		"  ⎿  Running…",
		"",
		"╭─────────────────────────────────────────────────────────────────────────────╮",
		"│ Bash command                                                                │",
		"│                                                                             │",
		"│   rm dangerous-file                                                         │",
		"│   Remove dangerous file for testing                                         │",
		"│                                                                             │",
		"│ Do you want to proceed?                                                     │",
		"│ ❯ 1. Yes                                                                    │",
		"│   2. No                                                                     │",
		"╰─────────────────────────────────────────────────────────────────────────────╯",
	}

	// Store original flag value
	originalAutoReject := *autoReject
	defer func() { *autoReject = originalAutoReject }()

	// Enable --auto-reject flag
	*autoReject = true

	robot := NewAppRobot(t).
		ReceiveClaudeText(realDialogLines...)

	// Wait for auto-reject goroutines to complete 
	// (AutoRejectProcessDelayMs + AutoRejectChoiceDelayMs + AutoRejectCRDelayMs + buffer)
	autoRejectWaitTime := time.Duration(500+500+400+100) * time.Millisecond
	time.Sleep(autoRejectWaitTime)

	// Test terminal output contains command details
	terminalOutput := robot.GetTerminalOutput()
	t.Logf("Terminal output: %q", terminalOutput)

	// Verify AutoRejectMessage includes rejected command details
	robot.AssertTerminalContains("automatically rejected").
		AssertTerminalContains("Rejected command:").
		AssertTerminalContains("rm dangerous-file").
		AssertTerminalContains("Remove dangerous file for testing")

	t.Logf("AutoRejectMessage with command details test completed")
}

func TestAutoRejectMessageCleanOutput(t *testing.T) {
	// Test that AutoRejectMessage properly cleans pipe characters and decorations
	realDialogLines := []string{
		"⏺ Bash(rm test-file)",
		"  ⎿  Running hook PreToolUse:Bash...",  
		"  ⎿  Running…",
		"",
		"╭─────────────────────────────────────────────────────────────────────────────╮",
		"│ Bash command                                                                │",
		"│                                                                             │",
		"│   rm test-file                                                              │",
		"│   Remove test-file from directory                                           │", 
		"│                                                                             │",
		"│ Do you want to proceed?                                                     │",
		"│ ❯ 1. Yes                                                                    │",
		"│   2. No                                                                     │",
		"╰─────────────────────────────────────────────────────────────────────────────╯",
	}

	// Store original flag value
	originalAutoReject := *autoReject
	defer func() { *autoReject = originalAutoReject }()

	// Enable --auto-reject flag
	*autoReject = true

	robot := NewAppRobot(t).
		ReceiveClaudeText(realDialogLines...)

	// Wait for auto-reject goroutines to complete
	autoRejectWaitTime := time.Duration(500+500+400+100) * time.Millisecond
	time.Sleep(autoRejectWaitTime)

	// Test terminal output for clean command details
	terminalOutput := robot.GetTerminalOutput()
	t.Logf("Raw terminal output: %q", terminalOutput)

	// Verify AutoRejectMessage should NOT contain pipe characters or decorations
	robot.AssertTerminalContains("automatically rejected").
		AssertTerminalContains("Rejected command:").
		AssertTerminalContains("rm test-file").
		AssertTerminalContains("Remove test-file from directory")

	// Check for problematic characters that should be cleaned
	if strings.Contains(terminalOutput, "│") {
		t.Errorf("❌ PROBLEM: Terminal output contains pipe characters that should be cleaned: %q", terminalOutput)
	}
	
	// Check specifically for decoration lines that start with "> "
	lines := strings.Split(terminalOutput, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "> ") {
			t.Errorf("❌ PROBLEM: Terminal output contains decoration line that should be filtered: %q", line)
		}
	}

	t.Logf("AutoRejectMessage clean output test completed")
}

func TestAutoRejectMessageComplexDialog(t *testing.T) {
	// Test with more complex dialog that might have decoration issues
	complexDialogLines := []string{
		"⏺ Bash(rm -rf /important/data)",
		"  ⎿  Running hook PreToolUse:Bash...",
		"  ⎿  Running…",
		"",
		"╭─────────────────────────────────────────────────────────────────────────────╮",
		"│ Bash command                                                                │",
		"│                                                                             │",
		"│   rm -rf /important/data                                                    │",
		"│   > This will delete all files in /important/data directory                │",
		"│   > Use with extreme caution                                                │",
		"│                                                                             │",
		"│ Do you want to proceed?                                                     │",
		"│ ❯ 1. Yes                                                                    │", 
		"│   2. No                                                                     │",
		"│   3. Cancel and review                                                      │",
		"╰─────────────────────────────────────────────────────────────────────────────╯",
	}

	// Store original flag value
	originalAutoReject := *autoReject
	defer func() { *autoReject = originalAutoReject }()

	// Enable --auto-reject flag
	*autoReject = true

	robot := NewAppRobot(t).
		ReceiveClaudeText(complexDialogLines...)

	// Wait for auto-reject goroutines to complete
	autoRejectWaitTime := time.Duration(500+500+400+100) * time.Millisecond
	time.Sleep(autoRejectWaitTime)

	// Test terminal output
	terminalOutput := robot.GetTerminalOutput()
	t.Logf("Complex dialog terminal output: %q", terminalOutput)

	// This might show the pipe character issue more clearly
	robot.AssertTerminalContains("automatically rejected").
		AssertTerminalContains("Rejected command:")

	t.Logf("Complex dialog test completed - check output for decoration characters")
}

func TestAutoRejectMessageRealWorldPipeIssue(t *testing.T) {
	// Test that reproduces the exact issue user reported where pipe appears in output
	// User reported seeing: "rm test-file                                                                                                                          
	//                         │
	//                       Remove file named test-file                                                                                                           
	//                         │"
	
	// Simulate exactly what user sees in their context - spaced pipe characters 
	realWorldDialogLines := []string{
		"⏺ Bash(rm test-file)",
		"  ⎿  Running hook PreToolUse:Bash...",
		"  ⎿  Running…",
		"",
		"╭─────────────────────────────────────────────────────────────────────────────╮",
		"│ Bash command                                                                │",
		"│                                                                             │",
		"│   rm test-file                                                                                                                          │", 
		"  │",  // This is the problematic line - spaced pipe that might not be trimmed correctly
		"│   Remove file named test-file                                                                                                           │",
		"  │",  // Another problematic line
		"│                                                                             │", 
		"│ Do you want to proceed?                                                     │",
		"│ ❯ 1. Yes                                                                    │",
		"│   2. No                                                                     │",
		"╰─────────────────────────────────────────────────────────────────────────────╯",
	}

	// Store original flag value
	originalAutoReject := *autoReject
	defer func() { *autoReject = originalAutoReject }()

	// Enable --auto-reject flag
	*autoReject = true

	robot := NewAppRobot(t).
		ReceiveClaudeText(realWorldDialogLines...)

	// Wait for auto-reject goroutines to complete
	autoRejectWaitTime := time.Duration(500+500+400+100) * time.Millisecond
	time.Sleep(autoRejectWaitTime)

	// Test terminal output for pipe characters
	terminalOutput := robot.GetTerminalOutput()
	t.Logf("Real world pipe issue terminal output: %q", terminalOutput)

	// This should fail if pipe characters are still present
	robot.AssertTerminalContains("automatically rejected").
		AssertTerminalContains("Rejected command:").
		AssertTerminalContains("rm test-file").
		AssertTerminalContains("Remove file named test-file")

	// Check line by line for pipe characters that should be filtered out
	lines := strings.Split(terminalOutput, "\n")
	for i, line := range lines {
		if strings.Contains(line, "│") {
			t.Errorf("❌ PIPE CHARACTER FOUND at line %d: %q\nFull output: %q", i, line, terminalOutput)
		}
		
		// Check for standalone pipe characters (the actual issue user reported)
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "│" {
			t.Errorf("❌ STANDALONE PIPE CHARACTER FOUND at line %d: %q", i, line)
		}
	}

	t.Logf("Real world pipe issue test completed")
}

func TestBuildAutoRejectMessageDebug(t *testing.T) {
	// Debug test to understand how buildAutoRejectMessage processes lines
	testContext := []string{
		"⏺ Bash(rm test-file)",
		"  ⎿  Running hook PreToolUse:Bash...",
		"  ⎿  Running…",
		"",
		"╭─────────────────────────────────────────────────────────────────────────────╮",
		"│ Bash command                                                                │",
		"│                                                                             │",
		"│   rm test-file                                                              │",
		"  │",  // This should be filtered as empty
		"│   Remove file named test-file                                               │",
		"  │",  // This should be filtered as empty
		"│                                                                             │",
		"│ Do you want to proceed?                                                     │",
		"│ ❯ 1. Yes                                                                    │",
		"│   2. No                                                                     │",
		"╰─────────────────────────────────────────────────────────────────────────────╯",
	}

	robot := NewAppRobot(t)
	handler := robot.app.handler
	
	// Set up context in the handler
	handler.appState.Prompt.Context = testContext
	
	// Call buildAutoRejectMessage directly and examine result
	result := handler.buildAutoRejectMessage()
	t.Logf("buildAutoRejectMessage result: %q", result)
	
	// Debug: Process each line and show what gets included
	t.Logf("=== Processing each context line ===")
	for i, line := range testContext {
		isValid := isValidCommandLine(line)
		cleanLine := strings.TrimSpace(strings.Trim(line, "│ \t"))
		
		t.Logf("Line %d: %q -> isValid=%t, cleanLine=%q", 
			i, line, isValid, cleanLine)
	}
	
	// Quality gate: Ensure no pipe characters leak through
	if strings.Contains(result, "│") {
		t.Errorf("❌ Result contains pipe characters: %q", result)
	}
	
	// Quality gate: Ensure no dialog choices leak through  
	if strings.Contains(result, "1. Yes") || strings.Contains(result, "2. No") {
		t.Errorf("❌ Result contains dialog choices that should be filtered: %q", result)
	}
	
	// Quality gate: Ensure no "Do you want to proceed" text leaks through
	if strings.Contains(result, "Do you want to proceed") {
		t.Errorf("❌ Result contains dialog question that should be filtered: %q", result)
	}
	
	// Verify the result contains expected command details
	if !strings.Contains(result, "rm test-file") {
		t.Errorf("❌ Result should contain 'rm test-file' command: %q", result)
	}
	
	if !strings.Contains(result, "Remove file named test-file") {
		t.Errorf("❌ Result should contain command description: %q", result)
	}
}
