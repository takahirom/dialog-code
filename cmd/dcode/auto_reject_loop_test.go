package main

import (
	"strings"
	"testing"
)

func TestAutoRejectLoopPrevention(t *testing.T) {
	// Test that dialog box containing auto-reject message doesn't trigger dialog detection
	// This reproduces the actual infinite loop issue

	t.Run("Dialog box with rejection message inside should not trigger dialog", func(t *testing.T) {
		// This is the ACTUAL problem: A dialog box that contains a rejection message
		// The "Do you want to make this edit" text is INSIDE a dialog box (with │ borders)
		dialogWithRejectionInside := []string{
			"╭─────────────────────────────────────────────────────────────────────────────╮",
			"│ > Rejected command:                                                           │",
			"│ Do you want to make this edit to gradle.properties?                        │",
			"│                                                                             │",
			"│ The command was automatically rejected. If using Task tools, please restart │",
			"│ them. Otherwise, try a different command.                                  │",
			"╰─────────────────────────────────────────────────────────────────────────────╯",
		}

		robot := NewAppRobot(t).
			ReceiveClaudeText(dialogWithRejectionInside...)

		// This should NOT be detected as a dialog requiring user input
		robot.AssertNoDialogCaptured()

		// No dialog choices should be sent
		terminalOutput := robot.GetTerminalOutput()
		if strings.Contains(terminalOutput, "1") || strings.Contains(terminalOutput, "2") {
			t.Errorf("Dialog box with rejection message triggered false detection: %q", terminalOutput)
		}

		t.Logf("✓ Dialog box containing rejection message handled correctly")
	})

	t.Run("Claude Code input box with 'Do you want to' text", func(t *testing.T) {
		// This simulates Claude Code's input box displaying "Do you want to" text
		inputBoxWithDoYouWant := []string{
			"╭──────────────────────────────────────────────────────────────────────────────╮",
			"│ > Do you want to proceed with the changes?                                  │",
			"╰──────────────────────────────────────────────────────────────────────────────╯",
		}

		robot := NewAppRobot(t).
			ReceiveClaudeText(inputBoxWithDoYouWant...)

		// This should NOT be detected as a dialog (it's an input box with >)
		robot.AssertNoDialogCaptured()

		// No dialog choices should be sent  
		terminalOutput := robot.GetTerminalOutput()
		if strings.Contains(terminalOutput, "1") || strings.Contains(terminalOutput, "2") {
			t.Errorf("Input box with 'Do you want to' triggered false detection: %q", terminalOutput)
		}

		t.Logf("✓ Input box with 'Do you want to' handled correctly")
	})

	t.Run("Claude outputs 'Do you want to make this edit' in input box", func(t *testing.T) {
		// Input box with regular space separator
		claudeOutputInInputBox := []string{
			"╭──────────────────────────────────────────────────────────────────────────────╮",
			"│ > Do you want to make this edit to file.txt?                                │",
			"╰──────────────────────────────────────────────────────────────────────────────╯",
		}

		robot := NewAppRobot(t).
			ReceiveClaudeText(claudeOutputInInputBox...)

		robot.AssertNoDialogCaptured()

		terminalOutput := robot.GetTerminalOutput()
		if strings.Contains(terminalOutput, "1") || strings.Contains(terminalOutput, "2") {
			t.Errorf("Claude's 'Do you want to make this edit' in input box triggered dialog detection: %q", terminalOutput)
		}

		t.Logf("✓ Claude's 'Do you want to make this edit' in input box handled correctly")
	})

	t.Run("Input box with non-breaking space", func(t *testing.T) {
		// Claude Code uses non-breaking space (U+00A0) instead of regular space
		bugScenario := []string{
			"╭──────────────────────────────────────────────────────────────────────────────╮",
			"│\u00a0>\u00a0Do you want to edit                                                        │",
			"╰──────────────────────────────────────────────────────────────────────────────╯",
		}

		robot := NewAppRobot(t).
			ReceiveClaudeText(bugScenario...)

		robot.AssertNoDialogCaptured()

		terminalOutput := robot.GetTerminalOutput()
		if strings.Contains(terminalOutput, "1") || strings.Contains(terminalOutput, "2") {
			t.Errorf("Input box with non-breaking space triggered dialog detection: %q", terminalOutput)
		}

		t.Logf("✓ Input box with non-breaking space handled correctly")
	})
}
