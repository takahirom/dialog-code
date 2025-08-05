package main

import (
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
		AssertMessageContains("Bash command").
		AssertMessageContains("rm not-found-file").
		AssertMessageContains("⏺ Bash(rm not-found-file)"). // Pre-dialog Claude output
		AssertMessageContains("Running hook PreToolUse:Bash"). // Hook execution info
		AssertDialogCaptured()

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
		AssertMessageContains("Task").
		AssertMessageContains("Execute dangerous operation").
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

	// Test exact matching with TimeProvider ensuring consistent results
	expectedMessage := `Context:
⏺ Bash(rm test-file)
  ⎿  Running hook PreToolUse:Bash...
  ⎿  Running…
╭─────────────────────────────────────────────────────────────────────────────╮
│ Bash command                                                                │
│                                                                             │
│   rm test-file                                                              │
│   Remove test file                                                          │
│                                                                             │
│ Do you want to proceed?                                                     │

│   Remove test file                                                          │|│                                                                             │|│ Do you want to proceed?                                                     │|│ Do you want to proceed?                                                     │|1672574400000000000

Reason: Proceed confirmation`
	robot.AssertExactFormatSnapshotTest(expectedMessage)
}