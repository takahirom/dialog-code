package main

import (
	"strings"
	"testing"
)

func TestAppWithParserIntegration(t *testing.T) {
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

	completeDialog := strings.Join(realDialogLines, "\n")

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
		AssertParserExtractsToolTypeAndContent(completeDialog, "Bash", "rm not-found-file")

	// Example of exact matching (note: includes timestamp so usually not practical)
	capturedMessage := robot.GetCapturedMessage()
	t.Logf("Complete captured message: %q", capturedMessage)
	
	// For exact matching without timestamp, you'd need to strip the timestamp part
	// This is usually too brittle for real tests, so AssertDialogTextContains is preferred
}

func TestAppWithParserEdgeCases(t *testing.T) {
	robot := NewAppRobot(t)

	// Test with empty context
	parsedInfo, err := robot.app.handler.ProcessWithParser("")
	if err == nil {
		t.Error("Parser should return error for empty input")
	}
	if parsedInfo != nil {
		t.Error("Parser should return nil for empty input")
	}

	// Test with malformed dialog
	malformedDialog := `╭─────────────────────────╮
│ Incomplete dialog       │
Missing closing border`

	parsedInfo, err = robot.app.handler.ProcessWithParser(malformedDialog)
	if err == nil {
		t.Error("Parser should return error for malformed dialog")
	}

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

	completeEditDialog := strings.Join(editDialogLines, "\n")

	NewAppRobot(t).
		ReceiveClaudeText(editDialogLines...).
		AssertParserExtractsToolTypeAndContent(completeEditDialog, "Edit", "/test/file.txt")

	t.Logf("Edge case tests passed")
}

func TestAppParserDialogFlow(t *testing.T) {
	// Test the complete flow: Claude output → context collection → parser → dialog
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

	completeTaskDialog := strings.Join(taskDialogLines, "\n")

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
		AssertParserExtractsToolTypeAndContent(completeTaskDialog, "Task", "Execute dangerous operation").
		LogDebugInfo()

	t.Logf("Complete parser-dialog flow test passed")
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