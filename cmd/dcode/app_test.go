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

func TestNonDialogDoYouWantMessage(t *testing.T) {
	// Test that "Do you want" text outside dialog box does NOT trigger "1" input
	// This reproduces the issue where plain text with "Do you want" causes "1" to be sent to terminal
	// Even though there's no permission dialog
	
	nonDialogLines := []string{
		"⏺ Edit command rejected",
		"Rejected command:",
		"Do you want to make this edit to DefaultFluffyByteIsPlayingAdapter.kt?",
		"",
		"The command was automatically rejected. If using Task tools, please restart them. Otherwise, try a different command.",
	}
	
	robot := NewAppRobot(t).
		ReceiveClaudeText(nonDialogLines...).
		AssertNoDialogCaptured()
	
	// Verify that no "1" was written to terminal
	terminalOutput := robot.GetTerminalOutput()
	if strings.Contains(terminalOutput, "1") {
		t.Errorf("Terminal output contains '1' when it shouldn't: %q", terminalOutput)
	}
	
	// Verify no dialog was detected
	if robot.app.handler.appState.Prompt.CollectedChoices != nil && len(robot.app.handler.appState.Prompt.CollectedChoices) > 0 {
		t.Errorf("Dialog choices were collected when there was no dialog: %v", 
			robot.app.handler.appState.Prompt.CollectedChoices)
	}
	
	t.Logf("Non-dialog 'Do you want' text correctly ignored")
}

func TestDoYouWantWithInputBox(t *testing.T) {
	// Test that "Do you want" text followed by an input box (not a dialog) doesn't trigger "1" input
	// This simulates the case where there's always an input box at the bottom
	
	inputBoxLines := []string{
		"⏺ Edit command rejected",
		"Rejected command:",
		"Do you want to make this edit to DefaultFluffyByteIsPlayingAdapter.kt?",
		"",
		"The command was automatically rejected. If using Task tools, please restart them. Otherwise, try a different command.",
		"",
		"╭─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮",
		"│ >                                                                                                                                                       │",
		"╰─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯",
		"  ⏵⏵ auto-accept edits on (shift+tab to cycle)                                                                                                          ◯",
	}
	
	robot := NewAppRobot(t).
		ReceiveClaudeText(inputBoxLines...).
		AssertNoDialogCaptured()
	
	// Verify that no "1" was written to terminal
	terminalOutput := robot.GetTerminalOutput()
	if strings.Contains(terminalOutput, "1") {
		t.Errorf("Terminal output contains '1' when it shouldn't: %q", terminalOutput)
	}
	
	// Verify this input box is not treated as a permission dialog
	if robot.app.handler.appState.Prompt.CollectedChoices != nil && len(robot.app.handler.appState.Prompt.CollectedChoices) > 0 {
		t.Errorf("Input box was incorrectly treated as dialog: %v", 
			robot.app.handler.appState.Prompt.CollectedChoices)
	}
	
	t.Logf("'Do you want' with input box correctly handled (no '1' input)")
}

func TestMixedContentWithDoYouWant(t *testing.T) {
	// Test that "Do you want" in regular text doesn't interfere with actual dialogs
	
	mixedLines := []string{
		"Claude: Do you want me to explain this code?",
		"Let me show you an example.",
		"",
		"⏺ Bash(ls -la)",
		"  ⎿  Running…",
		"",
		"╭─────────────────────────────────────────────────────────────────────────────╮",
		"│ Bash command                                                                │",
		"│                                                                             │",
		"│   ls -la                                                                    │",
		"│   List all files with details                                               │",
		"│                                                                             │",
		"│ Do you want to proceed?                                                     │",
		"│ ❯ 1. Yes                                                                    │",
		"│   2. No                                                                     │",
		"╰─────────────────────────────────────────────────────────────────────────────╯",
	}
	
	robot := NewAppRobot(t).
		ReceiveClaudeText(mixedLines...).
		AssertDialogCaptured().
		AssertButtonCount(2)
	
	// Verify only the actual dialog was captured, not the plain text "Do you want"
	capturedMessage := robot.GetCapturedMessage()
	if strings.Contains(capturedMessage, "Do you want me to explain") {
		t.Errorf("Plain text 'Do you want' was incorrectly captured: %q", capturedMessage)
	}
	
	t.Logf("Mixed content correctly handled")
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

func TestSerenaMCPDialogDetection(t *testing.T) {
	// Test that serena MCP tool with parameters shows proper dialog content
	// This reproduces the actual pattern from test_data.txt with even more content
	
	serenaMCPLines := []string{
		"⏺ serena - search_for_pattern (MCP)(substring_pattern: \"kotlin.*=.*1\\.\", relative_path: \"gradle/libs.versions.toml\")",
		"",
		"╭─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮",
		"│ Tool use                                                                                                                            │",
		"│                                                                                                                                     │",
		"│   serena - search_for_pattern(substring_pattern: \"kotlin.*=.*1\\.\", relative_path: \"gradle/libs.versions.toml\") (MCP)               │",
		"│   Offers a flexible search for arbitrary patterns in the codebase, including the                                                    │",
		"│   possibility to search in non-code files.                                                                                          │",
		"│   Generally, symbolic operations like find_symbol or find_referencing_symbols                                                       │",
		"│   should be preferred if you know which symbols you are looking for.                                                                │",
		"│                                                                                                                                     │",
		"│   Pattern Matching Logic:                                                                                                           │",
		"│       For each match, the returned result will contain the full lines where the                                                     │",
		"│       substring pattern is found, as well as optionally some lines before and after it. The pattern will be compiled with           │",
		"│       DOTALL, meaning that the dot will match all characters including newlines.                                                    │",
		"│       This also means that it never makes sense to have .* at the beginning or end of the pattern,                                  │",
		"│       but it may make sense to have it in the middle for complex patterns.                                                          │",
		"│       If a pattern matches multiple lines, all those lines will be part of the match.                                               │",
		"│       Be careful to not use greedy quantifiers unnecessarily, it is usually better to use non-greedy quantifiers like .*? to avoid  │",
		"│       matching too much content.                                                                                                    │",
		"│                                                                                                                                     │",
		"│   File Selection Logic:                                                                                                             │",
		"│       The files in which the search is performed can be restricted very flexibly.                                                   │",
		"│       Using `restrict_search_to_code_files` is useful if you are only interested in code symbols (i.e., those                       │",
		"│       symbols that can be manipulated with symbolic tools like find_symbol).                                                        │",
		"│       You can also restrict the search to a specific file or directory,                                                             │",
		"│       and provide glob patterns to include or exclude certain files on top of that.                                                 │",
		"│       The globs are matched against relative file paths from the project root (not to the `relative_path` parameter that            │",
		"│       is used to further restrict the search).                                                                                      │",
		"│       Smartly combining the various restrictions allows you to perform very targeted searches. Returns A mapping of file paths to    │",
		"│       lists of matched consecutive lines.                                                                                           │",
		"│                                                                                                                                     │",
		"│ Do you want to proceed?                                                                                                             │",
		"│ ❯ 1. Yes                                                                                                                            │",
		"│   2. No, change the command                                                                                                         │",
		"│   3. No, and tell Claude what to do differently (esc)                                                                               │",
		"╰─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯",
	}
	
	robot := NewAppRobot(t).
		ReceiveClaudeText(serenaMCPLines...).
		AssertDialogCaptured().
		AssertDialogTextContains("Do you want to proceed?").
		AssertDialogTextContains("Tool use").
		AssertButtonCount(3)
	
	// Check that trigger text is properly captured
	capturedMessage := robot.GetCapturedMessage()
	t.Logf("Captured message for serena MCP: %q", capturedMessage)
	
	// Verify trigger text exists and is not empty
	if !strings.Contains(capturedMessage, "Trigger text:") {
		t.Errorf("❌ Missing 'Trigger text:' in captured message")
	}
	
	// The captured message should include the MCP tool information
	if !strings.Contains(capturedMessage, "serena - search_for_pattern") {
		t.Errorf("❌ Missing MCP tool name in captured message")
	}
	
	// Check that important content is included (tool description)
	if !strings.Contains(capturedMessage, "Offers a flexible search") {
		t.Errorf("❌ Missing tool description in captured message")
	}
	
	// Parameters should be included
	if !strings.Contains(capturedMessage, "substring_pattern") || !strings.Contains(capturedMessage, "relative_path") {
		t.Errorf("❌ Missing tool parameters in captured message") 
	}
}
