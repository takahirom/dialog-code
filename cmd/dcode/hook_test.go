package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

// TestPermissionRequestHook tests the hook handler for both allow and deny scenarios
func TestPermissionRequestHook(t *testing.T) {
	tests := []struct {
		name           string
		dialogResponse string
		dialogMessage  string
		expectedOutput string
	}{
		{
			name:           "BashCommandAllow",
			dialogResponse: "1", // Allow button
			dialogMessage:  "",
			expectedOutput: `{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"allow"}}}`,
		},
		{
			name:           "BashCommandDeny",
			dialogResponse: "2", // Deny button
			dialogMessage:  "",
			expectedOutput: `{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","interrupt":false}}}`,
		},
		{
			name:           "BashCommandDenyWithMessage",
			dialogResponse: "2", // Deny button
			dialogMessage:  "Not safe to run",
			expectedOutput: `{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","message":"Not safe to run","interrupt":false}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: Create input JSON for a Bash command
			stdin := createTestInput(t)
			var stdout bytes.Buffer
			mockDialog := &MockDialog{
				response: tt.dialogResponse,
				message:  tt.dialogMessage,
			}

			// Act: Call the hook handler
			err := handlePermissionRequestHook(stdin, &stdout, mockDialog, 60)

			// Assert: No error occurred
			if err != nil {
				t.Fatalf("handlePermissionRequestHook returned error: %v", err)
			}

			// Assert: Output matches expected format
			assertJSONEqual(t, tt.expectedOutput, stdout.String())
		})
	}
}

// TestInvalidJSONReturnsError verifies that invalid JSON input causes an error (Requirement 3.1)
func TestInvalidJSONReturnsError(t *testing.T) {
	// Arrange: Create invalid JSON input
	invalidJSON := strings.NewReader("{ invalid json }")
	var stdout bytes.Buffer
	mockDialog := &MockDialog{
		response: "1", // Allow button (won't be used)
	}

	// Act: Call the hook handler with invalid JSON
	err := handlePermissionRequestHook(invalidJSON, &stdout, mockDialog, 60)

	// Assert: Error is returned
	if err == nil {
		t.Fatal("Expected error for invalid JSON, but got nil")
	}
}

// TestEmptyStdinReturnsEOF verifies that empty stdin causes an EOF error (Requirement 3.2)
// This allows the hook to gracefully exit with code 0 for empty input
func TestEmptyStdinReturnsEOF(t *testing.T) {
	// Arrange: Create empty stdin
	emptyStdin := strings.NewReader("")
	var stdout bytes.Buffer
	mockDialog := &MockDialog{
		response: "1", // Allow button (won't be used)
	}

	// Act: Call the hook handler with empty stdin
	err := handlePermissionRequestHook(emptyStdin, &stdout, mockDialog, 60)

	// Assert: EOF error is returned
	if err == nil {
		t.Fatal("Expected EOF error for empty stdin, but got nil")
	}

	// Assert: The error is EOF
	if !errors.Is(err, io.EOF) {
		t.Errorf("Expected EOF error, but got: %v", err)
	}
}

// TestDialogMessageContainsToolName verifies that the dialog message includes the tool name
func TestDialogMessageContainsToolName(t *testing.T) {
	// Arrange: Create input JSON with Bash tool
	stdin := createTestInput(t)
	var stdout bytes.Buffer
	mockDialog := &MockDialog{
		response: "1", // Allow button
	}

	// Act: Call the hook handler
	err := handlePermissionRequestHook(stdin, &stdout, mockDialog, 60)

	// Assert: No error occurred
	if err != nil {
		t.Fatalf("handlePermissionRequestHook returned error: %v", err)
	}

	// Assert: Dialog message contains the tool name "Bash"
	if !strings.Contains(mockDialog.capturedMessage, "Bash") {
		t.Errorf("Dialog message does not contain tool name 'Bash'.\nGot: %s", mockDialog.capturedMessage)
	}
}

// TestDialogMessageContainsCommandContent verifies that the dialog message includes the command content (Requirement 2.2)
func TestDialogMessageContainsCommandContent(t *testing.T) {
	// Arrange: Create input JSON with Bash tool and "npm run build" command
	stdin := createTestInput(t)
	var stdout bytes.Buffer
	mockDialog := &MockDialog{
		response: "1", // Allow button
	}

	// Act: Call the hook handler
	err := handlePermissionRequestHook(stdin, &stdout, mockDialog, 60)

	// Assert: No error occurred
	if err != nil {
		t.Fatalf("handlePermissionRequestHook returned error: %v", err)
	}

	// Assert: Dialog message contains the command content "npm run build"
	if !strings.Contains(mockDialog.capturedMessage, "npm run build") {
		t.Errorf("Dialog message does not contain command content 'npm run build'.\nGot: %s", mockDialog.capturedMessage)
	}
}

// TestDialogShowsAllowDenyButtons verifies that the dialog receives Allow and Deny buttons (Requirement 2.3)
func TestDialogShowsAllowDenyButtons(t *testing.T) {
	// Arrange: Create input JSON with Bash tool
	stdin := createTestInput(t)
	var stdout bytes.Buffer
	mockDialog := &MockDialog{
		response: "1", // Allow button
	}

	// Act: Call the hook handler
	err := handlePermissionRequestHook(stdin, &stdout, mockDialog, 60)

	// Assert: No error occurred
	if err != nil {
		t.Fatalf("handlePermissionRequestHook returned error: %v", err)
	}

	// Assert: Dialog receives exactly 2 buttons
	if len(mockDialog.capturedButtons) != 2 {
		t.Errorf("Expected 2 buttons, got %d: %v", len(mockDialog.capturedButtons), mockDialog.capturedButtons)
	}

	// Assert: First button is "Allow"
	if len(mockDialog.capturedButtons) > 0 && mockDialog.capturedButtons[0] != "Allow" {
		t.Errorf("Expected first button to be 'Allow', got '%s'", mockDialog.capturedButtons[0])
	}

	// Assert: Second button is "Deny"
	if len(mockDialog.capturedButtons) > 1 && mockDialog.capturedButtons[1] != "Deny" {
		t.Errorf("Expected second button to be 'Deny', got '%s'", mockDialog.capturedButtons[1])
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

// TestEditToolShowsFilePath verifies that the dialog message contains file_path for Edit tool (Requirement 4.1)
func TestEditToolShowsFilePath(t *testing.T) {
	// Arrange: Create input JSON with Edit tool
	stdin := createEditToolInput(t)
	var stdout bytes.Buffer
	mockDialog := &MockDialog{
		response: "1", // Allow button
	}

	// Act: Call the hook handler
	err := handlePermissionRequestHook(stdin, &stdout, mockDialog, 60)

	// Assert: No error occurred
	if err != nil {
		t.Fatalf("handlePermissionRequestHook returned error: %v", err)
	}

	// Assert: Dialog message contains the file_path
	if !strings.Contains(mockDialog.capturedMessage, "/path/to/file.go") {
		t.Errorf("Dialog message does not contain file_path '/path/to/file.go'.\nGot: %s", mockDialog.capturedMessage)
	}

	// Assert: Dialog message contains tool name "Edit"
	if !strings.Contains(mockDialog.capturedMessage, "Edit") {
		t.Errorf("Dialog message does not contain tool name 'Edit'.\nGot: %s", mockDialog.capturedMessage)
	}
}

// TestWriteToolShowsFilePath verifies that the dialog message contains file_path for Write tool (Requirement 4.2)
func TestWriteToolShowsFilePath(t *testing.T) {
	// Arrange: Create input JSON with Write tool
	stdin := createWriteToolInput(t)
	var stdout bytes.Buffer
	mockDialog := &MockDialog{
		response: "1", // Allow button
	}

	// Act: Call the hook handler
	err := handlePermissionRequestHook(stdin, &stdout, mockDialog, 60)

	// Assert: No error occurred
	if err != nil {
		t.Fatalf("handlePermissionRequestHook returned error: %v", err)
	}

	// Assert: Dialog message contains the file_path
	if !strings.Contains(mockDialog.capturedMessage, "/path/to/file.go") {
		t.Errorf("Dialog message does not contain file_path '/path/to/file.go'.\nGot: %s", mockDialog.capturedMessage)
	}

	// Assert: Dialog message contains tool name "Write"
	if !strings.Contains(mockDialog.capturedMessage, "Write") {
		t.Errorf("Dialog message does not contain tool name 'Write'.\nGot: %s", mockDialog.capturedMessage)
	}
}

// TestUnknownToolStillWorks verifies that unknown tools still show dialog with raw tool_input (Requirement 4.3)
func TestUnknownToolStillWorks(t *testing.T) {
	// Arrange: Create input JSON with an unknown tool
	stdin := createUnknownToolInput(t)
	var stdout bytes.Buffer
	mockDialog := &MockDialog{
		response: "1", // Allow button
	}

	// Act: Call the hook handler
	err := handlePermissionRequestHook(stdin, &stdout, mockDialog, 60)

	// Assert: No error occurred
	if err != nil {
		t.Fatalf("handlePermissionRequestHook returned error: %v", err)
	}

	// Assert: Dialog message contains tool name "UnknownTool"
	if !strings.Contains(mockDialog.capturedMessage, "UnknownTool") {
		t.Errorf("Dialog message does not contain tool name 'UnknownTool'.\nGot: %s", mockDialog.capturedMessage)
	}

	// Assert: Output matches expected format (allow decision)
	expectedOutput := `{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"allow"}}}`
	assertJSONEqual(t, expectedOutput, stdout.String())
}

// TestDialogTimeout verifies that when dialog times out, it returns deny with timeout message
func TestDialogTimeout(t *testing.T) {
	tests := []struct {
		name            string
		timeout         int
		expectedMessage string
	}{
		{
			name:            "Default 60 second timeout",
			timeout:         60,
			expectedMessage: "User did not respond within 60 seconds",
		},
		{
			name:            "Custom 30 second timeout",
			timeout:         30,
			expectedMessage: "User did not respond within 30 seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: Create input JSON with Bash tool
			stdin := createTestInput(t)
			var stdout bytes.Buffer
			mockDialog := &MockDialog{
				response: "", // Empty string indicates timeout
			}

			// Act: Call the hook handler
			err := handlePermissionRequestHook(stdin, &stdout, mockDialog, tt.timeout)

			// Assert: No error occurred
			if err != nil {
				t.Fatalf("handlePermissionRequestHook returned error: %v", err)
			}

			// Assert: Output is deny with correct timeout message
			expectedOutput := fmt.Sprintf(`{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","message":"%s","interrupt":false}}}`, tt.expectedMessage)
			assertJSONEqual(t, expectedOutput, stdout.String())
		})
	}
}

// createEditToolInput creates a mock stdin reader with Edit tool JSON input
func createEditToolInput(t *testing.T) *bytes.Reader {
	t.Helper()
	input := map[string]interface{}{
		"hook_event_name": "PermissionRequest",
		"tool_name":       "Edit",
		"tool_input": map[string]interface{}{
			"file_path":  "/path/to/file.go",
			"old_string": "old",
			"new_string": "new",
		},
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input JSON: %v", err)
	}
	return bytes.NewReader(inputJSON)
}

// createWriteToolInput creates a mock stdin reader with Write tool JSON input
func createWriteToolInput(t *testing.T) *bytes.Reader {
	t.Helper()
	input := map[string]interface{}{
		"hook_event_name": "PermissionRequest",
		"tool_name":       "Write",
		"tool_input": map[string]interface{}{
			"file_path": "/path/to/file.go",
			"content":   "file content",
		},
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input JSON: %v", err)
	}
	return bytes.NewReader(inputJSON)
}

// createUnknownToolInput creates a mock stdin reader with an unknown tool JSON input
func createUnknownToolInput(t *testing.T) *bytes.Reader {
	t.Helper()
	input := map[string]interface{}{
		"hook_event_name": "PermissionRequest",
		"tool_name":       "UnknownTool",
		"tool_input": map[string]interface{}{
			"some_param": "some_value",
		},
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input JSON: %v", err)
	}
	return bytes.NewReader(inputJSON)
}

// TestParseTimeoutFlag verifies that --timeout=N flag is parsed correctly
func TestParseTimeoutFlag(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedTimeout int
	}{
		{
			name:            "Default timeout is 60 seconds",
			args:            []string{},
			expectedTimeout: 60,
		},
		{
			name:            "Custom timeout 30 seconds",
			args:            []string{"--timeout=30"},
			expectedTimeout: 30,
		},
		{
			name:            "Custom timeout 120 seconds",
			args:            []string{"--timeout=120"},
			expectedTimeout: 120,
		},
		{
			name:            "Minimum timeout 5 seconds",
			args:            []string{"--timeout=5"},
			expectedTimeout: 5,
		},
		{
			name:            "Maximum timeout 3600 seconds",
			args:            []string{"--timeout=3600"},
			expectedTimeout: 3600,
		},
		{
			name:            "Below minimum defaults to 60",
			args:            []string{"--timeout=4"},
			expectedTimeout: 60,
		},
		{
			name:            "Above maximum defaults to 60",
			args:            []string{"--timeout=3601"},
			expectedTimeout: 60,
		},
		{
			name:            "Invalid value defaults to 60",
			args:            []string{"--timeout=abc"},
			expectedTimeout: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout := parseTimeoutFlag(tt.args)
			if timeout != tt.expectedTimeout {
				t.Errorf("parseTimeoutFlag(%v) = %d, want %d", tt.args, timeout, tt.expectedTimeout)
			}
		})
	}
}

// MockDialog is a mock implementation of the dialog interface for testing
type MockDialog struct {
	response        string
	message         string
	capturedMessage string   // Captures the message passed to Show()
	capturedButtons []string // Captures the buttons passed to Show()
}

// Show implements the dialog interface, returning the mocked response
// For deny responses with a message, it appends the message after a pipe separator
func (m *MockDialog) Show(message string, buttons []string, defaultButton string) string {
	m.capturedMessage = message   // Capture the message for verification
	m.capturedButtons = buttons    // Capture the buttons for verification
	if m.message != "" {
		return m.response + "|" + m.message
	}
	return m.response
}
