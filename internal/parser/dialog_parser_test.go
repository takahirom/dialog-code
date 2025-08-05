package parser

import (
	"strings"
	"testing"
)

func TestParseSimpleBashDialog(t *testing.T) {
	input := `Previous output from terminal...
Processing user request...

╭─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ Bash command                                                                                                                                            │
│                                                                                                                                                         │
│   rm not-found-file                                                                                                                                     │
│   Test dialog message for data collection                                                                                                               │
│                                                                                                                                                         │
│ Do you want to proceed?                                                                                                                                 │
│ ❯ 1. Yes                                                                                                                                                │
│   2. Yes, and don't ask again for rm commands in /Users/test/git/dialog-code                                                                          │
│   3. No, and tell Claude what to do differently (esc)                                                                                                   │
╰─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯

Some trailing text after dialog...`

	result, err := ParseDialog(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test that raw content is extracted
	if result.RawContent == "" {
		t.Error("Expected RawContent to be non-empty")
	}

	// Test that it contains the expected content using standard library
	if !strings.Contains(result.RawContent, "Bash command") {
		t.Error("Expected RawContent to contain 'Bash command'")
	}
	if !strings.Contains(result.RawContent, "rm not-found-file") {
		t.Error("Expected RawContent to contain 'rm not-found-file'")
	}
	if !strings.Contains(result.RawContent, "Do you want to proceed?") {
		t.Error("Expected RawContent to contain 'Do you want to proceed?'")
	}

	// Test tool type extraction
	if result.ToolType != "Bash" {
		t.Errorf("Expected ToolType 'Bash', got %q", result.ToolType)
	}
}

func TestParseDialogNoContent(t *testing.T) {
	input := `No dialog box here
Just some text without borders`

	result, err := ParseDialog(input)
	if err == nil {
		t.Fatal("Expected error for input without dialog borders")
	}
	if result != nil {
		t.Error("Expected nil result when parsing fails")
	}
}

func TestParseEditDialog(t *testing.T) {
	input := `╭─────────────────────────────────────────────────────────────────╮
│ Edit command                                                    │
│                                                                 │
│   nano config.yaml                                              │
│                                                                 │
│ Do you want to proceed?                                         │
│ ❯ 1. Yes                                                        │
│   2. No                                                         │
╰─────────────────────────────────────────────────────────────────╯`

	result, err := ParseDialog(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.ToolType != "Edit" {
		t.Errorf("Expected ToolType 'Edit', got %q", result.ToolType)
	}
	if !strings.Contains(result.RawContent, "nano config.yaml") {
		t.Error("Expected RawContent to contain 'nano config.yaml'")
	}
}

func TestParseEmptyInput(t *testing.T) {
	result, err := ParseDialog("")
	if err == nil {
		t.Fatal("Expected error for empty input")
	}
	if result != nil {
		t.Error("Expected nil result for empty input")
	}
}

func TestParseMalformedDialog(t *testing.T) {
	input := `╭─────────────────────────────────────────────────────────────────╮
│ Bash command                                                    │
│   rm test-file                                                  │
│ Missing closing border`

	result, err := ParseDialog(input)
	if err == nil {
		t.Fatal("Expected error for malformed dialog without closing border")
	}
	if result != nil {
		t.Error("Expected nil result for malformed dialog")
	}
}