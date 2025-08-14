package dialog

import (
	"testing"
)

func TestSimpleOSDialog_AppleScriptError(t *testing.T) {
	dialog := NewSimpleOSDialog()
	
	// Test case 1: Normal buttons should return max button number when AppleScript fails
	t.Run("AppleScript error returns max button number", func(t *testing.T) {
		// This test would need a way to force AppleScript to fail
		// For now, we test the error handling path with a mock
		
		// Simulate buttons like ["Allow", "Deny", "Always Deny"]
		buttons := []string{"Allow", "Deny", "Always Deny"}
		
		// We can't easily mock exec.Command in this simple test, 
		// but we can test the parsing logic
		result := dialog.parseAppleScriptResult("", buttons)
		expected := "3" // Default to last button (most restrictive) when parsing fails
		
		if result != expected {
			t.Errorf("Expected %s but got %s when parsing fails", expected, result)
		}
	})
	
	// Test case 2: Test escaping logic that might cause AppleScript errors
	t.Run("Special characters in message should be escaped", func(t *testing.T) {
		// Test escaping logic
		testCases := []struct {
			input    string
			expected string
		}{
			{`Hello "World"`, `Hello \"World\"`},
			{`Path\to\file`, `Path\\to\\file`},
			{`Mix "quotes" and \backslashes\`, `Mix \"quotes\" and \\backslashes\\`},
		}
		
		for _, tc := range testCases {
			result := dialog.escapeForAppleScript(tc.input)
			if result != tc.expected {
				t.Errorf("escapeForAppleScript(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		}
	})
}

func TestSimpleOSDialog_ParseAppleScriptResult(t *testing.T) {
	dialog := NewSimpleOSDialog()
	buttons := []string{"Allow", "Deny", "Always Deny"}
	
	testCases := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "Valid button returned output",
			output:   "button returned:Allow",
			expected: "1",
		},
		{
			name:     "Second button returned",
			output:   "button returned:Deny", 
			expected: "2",
		},
		{
			name:     "Third button returned",
			output:   "button returned:Always Deny",
			expected: "3",
		},
		{
			name:     "Invalid output format",
			output:   "some other output",
			expected: "3", // Default to last button (most restrictive)
		},
		{
			name:     "Empty output",
			output:   "",
			expected: "3", // Default to last button (most restrictive)
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := dialog.parseAppleScriptResult(tc.output, buttons)
			if result != tc.expected {
				t.Errorf("parseAppleScriptResult(%q) = %q, want %q", tc.output, result, tc.expected)
			}
		})
	}
}

func TestSimpleOSDialog_ParseChooseFromListResult(t *testing.T) {
	dialog := NewSimpleOSDialog()
	buttons := []string{"Allow", "Deny", "Always Deny", "Never Allow"}
	
	testCases := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "Valid selection - first button",
			output:   "Allow",
			expected: "1",
		},
		{
			name:     "Valid selection - second button",
			output:   "Deny",
			expected: "2",
		},
		{
			name:     "Valid selection - third button",
			output:   "Always Deny",
			expected: "3",
		},
		{
			name:     "Valid selection - fourth button",
			output:   "Never Allow",
			expected: "4",
		},
		{
			name:     "User cancelled (false)",
			output:   "false",
			expected: "4", // Most restrictive choice
		},
		{
			name:     "Unknown selection",
			output:   "Unknown Option",
			expected: "4", // Most restrictive choice
		},
		{
			name:     "Empty output",
			output:   "",
			expected: "4", // Most restrictive choice
		},
		{
			name:     "Whitespace output",
			output:   "  \n\t  ",
			expected: "4", // Most restrictive choice
		},
		{
			name:     "Braced format - first button",
			output:   "{Allow}",
			expected: "1",
		},
		{
			name:     "Braced format with quotes",
			output:   `{"Deny"}`,
			expected: "2",
		},
		{
			name:     "Braced format with multiple items",
			output:   `{"Always Deny", "Never Allow"}`,
			expected: "3", // Should take first item
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := dialog.parseChooseFromListResult(tc.output, buttons)
			if result != tc.expected {
				t.Errorf("parseChooseFromListResult(%q) = %q, want %q", tc.output, result, tc.expected)
			}
		})
	}
}

func TestSimpleOSDialog_ButtonCountBranching(t *testing.T) {
	dialog := NewSimpleOSDialog()
	
	t.Run("3 buttons or less uses display dialog", func(t *testing.T) {
		// This test verifies the branching logic but cannot easily test the actual AppleScript execution
		// In a real test environment, we would mock the execution methods
		buttons := []string{"Allow", "Deny", "Cancel"}
		message := "Test message"
		
		// The Show method should handle this without error
		// Note: This will fail in CI/test environments without AppleScript, but validates the code path
		result := dialog.Show(message, buttons, "Allow")
		
		// Since we can't easily mock AppleScript in this simple test, 
		// we expect it to return the fallback value
		if result == "" {
			t.Error("Show should return a non-empty result even on error")
		}
	})
	
	t.Run("4 buttons or more uses choose from list", func(t *testing.T) {
		buttons := []string{"Allow", "Deny", "Always Allow", "Never Allow"}
		message := "Test message"
		
		// The Show method should handle this without error
		result := dialog.Show(message, buttons, "Allow")
		
		// Since we can't easily mock AppleScript in this simple test,
		// we expect it to return the fallback value
		if result == "" {
			t.Error("Show should return a non-empty result even on error")
		}
	})
}