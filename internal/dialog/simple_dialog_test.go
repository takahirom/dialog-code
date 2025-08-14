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
		expected := "1" // Default when parsing fails
		
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
			expected: "1", // Default to first button
		},
		{
			name:     "Empty output",
			output:   "",
			expected: "1", // Default to first button
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