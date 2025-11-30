package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestPermissionRequestHook_BashCommandAllow tests that when user approves a Bash command,
// the hook handler outputs the correct allow JSON format
func TestPermissionRequestHook_BashCommandAllow(t *testing.T) {
	// Arrange: Create input JSON for a Bash command
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

	// Mock stdin with the input JSON
	stdin := bytes.NewReader(inputJSON)

	// Mock stdout to capture output
	var stdout bytes.Buffer

	// Mock dialog to return "Allow" (user approves)
	mockDialog := &MockDialog{
		response: "1", // "1" represents first button (Allow)
	}

	// Act: Call the hook handler
	err = handlePermissionRequestHook(stdin, &stdout, mockDialog)

	// Assert: No error occurred
	if err != nil {
		t.Fatalf("handlePermissionRequestHook returned error: %v", err)
	}

	// Assert: Output JSON is correct allow format
	var output map[string]interface{}
	err = json.Unmarshal(stdout.Bytes(), &output)
	if err != nil {
		t.Fatalf("Failed to unmarshal output JSON: %v", err)
	}

	// Verify the output structure
	hookOutput, ok := output["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Output missing hookSpecificOutput field")
	}

	hookEventName, ok := hookOutput["hookEventName"].(string)
	if !ok || hookEventName != "PermissionRequest" {
		t.Errorf("Expected hookEventName to be 'PermissionRequest', got: %v", hookEventName)
	}

	decision, ok := hookOutput["decision"].(map[string]interface{})
	if !ok {
		t.Fatal("Output missing decision field")
	}

	behavior, ok := decision["behavior"].(string)
	if !ok || behavior != "allow" {
		t.Errorf("Expected behavior to be 'allow', got: %v", behavior)
	}

	// Verify the exact expected output format
	expectedOutput := `{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"allow"}}}`
	actualOutput := stdout.String()

	// Compare JSON structures (order-independent)
	var expectedJSON, actualJSON map[string]interface{}
	json.Unmarshal([]byte(expectedOutput), &expectedJSON)
	json.Unmarshal([]byte(actualOutput), &actualJSON)

	if !jsonEqual(expectedJSON, actualJSON) {
		t.Errorf("Output JSON does not match expected format.\nExpected: %s\nGot: %s", expectedOutput, actualOutput)
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

// jsonEqual compares two JSON objects for structural equality
func jsonEqual(a, b map[string]interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}
