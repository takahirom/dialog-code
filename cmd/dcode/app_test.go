package main

import (
	"strings"
	"testing"
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
