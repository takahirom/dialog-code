package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestPermissionRequestHook tests the hook handler for both allow and deny scenarios
func TestPermissionRequestHook(t *testing.T) {
	tests := []struct {
		name           string
		dialogResponse string
		expectedOutput string
	}{
		{
			name:           "BashCommandAllow",
			dialogResponse: "1", // Allow button
			expectedOutput: `{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"allow"}}}`,
		},
		{
			name:           "BashCommandDeny",
			dialogResponse: "2", // Deny button
			expectedOutput: `{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","interrupt":false}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: Create input JSON for a Bash command
			stdin := createTestInput(t)
			var stdout bytes.Buffer
			mockDialog := &MockDialog{response: tt.dialogResponse}

			// Act: Call the hook handler
			err := handlePermissionRequestHook(stdin, &stdout, mockDialog)

			// Assert: No error occurred
			if err != nil {
				t.Fatalf("handlePermissionRequestHook returned error: %v", err)
			}

			// Assert: Output matches expected format
			assertJSONEqual(t, tt.expectedOutput, stdout.String())
		})
	}
}

// createTestInput creates a mock stdin reader with a Bash command JSON input
func createTestInput(t *testing.T) *bytes.Reader {
	t.Helper()
	input := map[string]interface{}{
		"hook_event_name": "PermissionRequest",
		"tool_name":       "Bash",
		"tool_input": map[string]interface{}{
			"command": "npm run build",
		},
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input JSON: %v", err)
	}
	return bytes.NewReader(inputJSON)
}

// assertJSONEqual compares two JSON strings for structural equality
func assertJSONEqual(t *testing.T, expected, actual string) {
	t.Helper()
	var expectedJSON, actualJSON map[string]interface{}
	if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(actual), &actualJSON); err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %v", err)
	}

	expectedBytes, _ := json.Marshal(expectedJSON)
	actualBytes, _ := json.Marshal(actualJSON)
	if string(expectedBytes) != string(actualBytes) {
		t.Errorf("Output JSON does not match expected format.\nExpected: %s\nGot: %s", expected, actual)
	}
}

// MockDialog is a mock implementation of the dialog interface for testing
type MockDialog struct {
	response string
}

// Show implements the dialog interface, returning the mocked response
func (m *MockDialog) Show(message string, buttons []string, defaultButton string) string {
	return m.response
}
