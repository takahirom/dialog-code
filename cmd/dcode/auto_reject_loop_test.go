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
}
