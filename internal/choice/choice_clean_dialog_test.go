package choice

import (
	"regexp"
	"strings"
	"testing"

	"github.com/takahirom/dialog-code/internal/types"
)

func TestGetCleanDialogMessage(t *testing.T) {
	// Create regex patterns for testing
	ansiEscape := regexp.MustCompile(`\x1b\[[0-9;?]*[mKHJhlABCDEFGPST]`)
	regexPatterns := &types.RegexPatterns{AnsiEscape: ansiEscape}

	tests := []struct {
		name          string
		prompt        string
		context       []string
		triggerReason string
		triggerLine   string
		timestamp     string
		expected      string
	}{
		{
			name:          "basic bash command dialog",
			prompt:        "│   rm test-file                                                              │",
			context: []string{
				"⏺ Bash(rm test-file)",
				"  ⎿  Running hook PreToolUse:Bash...",
				"  ⎿  Running…",
				"╭─────────────────────────────────────────────────────────────────────────────╮",
				"│ Bash command                                                                │",
				"│                                                                             │",
				"│   rm test-file                                                              │",
				"│   Remove test file                                                          │",
				"│                                                                             │",
				"│ Do you want to proceed?                                                     │",
				"╰─────────────────────────────────────────────────────────────────────────────╯",
			},
			triggerReason: "Proceed confirmation",
			triggerLine:   "│   rm test-file                                                              │",
			timestamp:     "1672574400000000000",
			expected: `Trigger text: ⏺ Bash(rm test-file)
Trigger timestamp: 1672574400000000000
Reason: Proceed confirmation
───────────────────────────────────
Bash command

  rm test-file
  Remove test file

Do you want to proceed?`,
		},
		{
			name:          "edit command dialog",
			prompt:        "│   file_path: /test/file.txt                                     │",
			context: []string{
				"╭─────────────────────────────────────────────────────────────────╮",
				"│ Edit command                                                    │",
				"│                                                                 │",
				"│   file_path: /test/file.txt                                     │",
				"│   Edit content here                                             │",
				"│                                                                 │",
				"│ Do you want to proceed?                                         │",
				"╰─────────────────────────────────────────────────────────────────╯",
			},
			triggerReason: "File modification",
			triggerLine:   "│   file_path: /test/file.txt                                     │",
			timestamp:     "1672574400000000000",
			expected: `Trigger text: file_path: /test/file.txt
Trigger timestamp: 1672574400000000000
Reason: File modification
───────────────────────────────────
Edit command

  file_path: /test/file.txt
  Edit content here

Do you want to proceed?`,
		},
		{
			name:          "task command dialog",
			prompt:        "│   description: Test complex task                                │",
			context: []string{
				"╭─────────────────────────────────────────────────────────────────╮",
				"│ Task                                                            │",
				"│                                                                 │",
				"│   description: Test complex task                                │",
				"│   prompt: Execute dangerous operation                           │",
				"│                                                                 │",
				"│ Do you want to proceed?                                         │",
				"╰─────────────────────────────────────────────────────────────────╯",
			},
			triggerReason: "Proceed confirmation",
			triggerLine:   "│   description: Test complex task                                │",
			timestamp:     "1672574400000000000",
			expected: `Trigger text: description: Test complex task
Trigger timestamp: 1672574400000000000
Reason: Proceed confirmation
───────────────────────────────────
Task

  description: Test complex task
  prompt: Execute dangerous operation

Do you want to proceed?`,
		},
		{
			name:          "minimal dialog without context",
			prompt:        "Simple question",
			context:       []string{},
			triggerReason: "Basic confirmation",
			triggerLine:   "Simple question",
			timestamp:     "1672574400000000000",
			expected: `Trigger text: Simple question
Trigger timestamp: 1672574400000000000
Reason: Basic confirmation
───────────────────────────────────
Do you want to proceed?`,
		},
		{
			name:          "dialog with trigger text but no box",
			prompt:        "Direct prompt",
			context: []string{
				"⏺ Direct(command)",
				"Some context line",
			},
			triggerReason: "Direct execution",
			triggerLine:   "Direct prompt",
			timestamp:     "1672574400000000000",
			expected: `Trigger text: ⏺ Direct(command)
Trigger timestamp: 1672574400000000000
Reason: Direct execution
───────────────────────────────────
Do you want to proceed?`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCleanDialogMessage(
				tt.prompt,
				tt.context,
				tt.triggerReason,
				tt.triggerLine,
				tt.timestamp,
				regexPatterns,
			)

			if result != tt.expected {
				t.Errorf("Test %s failed.\nExpected:\n%s\n\nGot:\n%s", tt.name, tt.expected, result)
			}
		})
	}
}

func TestGetCleanDialogMessage_EdgeCases(t *testing.T) {
	ansiEscape := regexp.MustCompile(`\x1b\[[0-9;?]*[mKHJhlABCDEFGPST]`)
	regexPatterns := &types.RegexPatterns{AnsiEscape: ansiEscape}

	t.Run("empty inputs", func(t *testing.T) {
		result := GetCleanDialogMessage("", []string{}, "", "", "", regexPatterns)
		expected := `───────────────────────────────────
Do you want to proceed?`
		if result != expected {
			t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
		}
	})

	t.Run("nil regex patterns", func(t *testing.T) {
		// Should not panic
		result := GetCleanDialogMessage("test", []string{}, "reason", "line", "123", nil)
		if !strings.Contains(result, "Do you want to proceed?") {
			t.Error("Should handle nil regex patterns gracefully")
		}
	})

	t.Run("context with ANSI codes", func(t *testing.T) {
		context := []string{
			"\x1b[31m⏺ Bash(rm test-file)\x1b[0m",
			"╭─────────────────────────────────────────────────────────────────────────────╮",
			"│ \x1b[1mBash command\x1b[0m                                                       │",
			"│                                                                             │",
			"│   rm test-file                                                              │",
			"╰─────────────────────────────────────────────────────────────────────────────╯",
		}

		result := GetCleanDialogMessage("test", context, "Test", "test", "123", regexPatterns)
		
		// Should strip ANSI codes
		if strings.Contains(result, "\x1b[") {
			t.Error("Should strip ANSI codes from result")
		}
		
		// Should still contain the trigger text without ANSI codes
		if !strings.Contains(result, "⏺ Bash(rm test-file)") {
			t.Error("Should extract trigger text correctly after stripping ANSI codes")
		}
	})

	t.Run("malformed dialog box", func(t *testing.T) {
		context := []string{
			"╭─────────────────────────╮",
			"│ Command without closing │",
			"│   some content          │",
			// Missing closing box
		}

		result := GetCleanDialogMessage("test", context, "Test", "test", "123", regexPatterns)
		// Should not panic and return something reasonable
		if !strings.Contains(result, "Do you want to proceed?") {
			t.Error("Should handle malformed dialog box gracefully")
		}
	})
}

func TestGetCleanDialogMessage_HelperFunctions(t *testing.T) {
	// Test individual aspects that could be extracted to helper functions
	
	t.Run("trigger text extraction", func(t *testing.T) {
		ansiEscape := regexp.MustCompile(`\x1b\[[0-9;?]*[mKHJhlABCDEFGPST]`)
		regexPatterns := &types.RegexPatterns{AnsiEscape: ansiEscape}
		
		context := []string{
			"some other line",
			"⏺ Bash(rm test-file)",
			"more context",
		}
		
		result := GetCleanDialogMessage("test", context, "Test", "test", "123", regexPatterns)
		
		if !strings.Contains(result, "Trigger text: ⏺ Bash(rm test-file)") {
			t.Error("Should extract trigger text from context correctly")
		}
	})
	
	t.Run("command type and details parsing", func(t *testing.T) {
		ansiEscape := regexp.MustCompile(`\x1b\[[0-9;?]*[mKHJhlABCDEFGPST]`)
		regexPatterns := &types.RegexPatterns{AnsiEscape: ansiEscape}
		
		context := []string{
			"╭─────────────────────────────────────────────────────────────────────────────╮",
			"│ Custom Command Type                                                         │",
			"│                                                                             │",
			"│   detail line 1                                                             │",
			"│   detail line 2                                                             │",
			"│                                                                             │",
			"│ Do you want to proceed?                                                     │",
			"╰─────────────────────────────────────────────────────────────────────────────╯",
		}
		
		result := GetCleanDialogMessage("test", context, "Test", "test", "123", regexPatterns)
		
		if !strings.Contains(result, "Custom Command Type") {
			t.Error("Should extract command type correctly")
		}
		
		if !strings.Contains(result, "  detail line 1") {
			t.Error("Should extract and indent command details correctly")
		}
		
		if !strings.Contains(result, "  detail line 2") {
			t.Error("Should extract and indent command details correctly")
		}
	})
}