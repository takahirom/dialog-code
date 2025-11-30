package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	// Button indices returned by dialog
	buttonIndexAllow = "1"
	buttonIndexDeny  = "2"

	// Behavior values for hook response
	behaviorAllow = "allow"
	behaviorDeny  = "deny"

	// Hook event name
	hookEventPermissionRequest = "PermissionRequest"
)

// DialogInterface defines the contract for showing dialogs
type DialogInterface interface {
	Show(message string, buttons []string, defaultButton string) string
}

// handlePermissionRequestHook processes a PermissionRequest hook event
// It reads JSON input from stdin, shows a dialog to the user, and outputs
// a JSON response to stdout based on the user's decision
func handlePermissionRequestHook(stdin io.Reader, stdout io.Writer, dialog DialogInterface) error {
	// Read input JSON
	var input map[string]interface{}
	decoder := json.NewDecoder(stdin)
	if err := decoder.Decode(&input); err != nil {
		return err
	}

	// Extract tool_name and tool_input from input
	toolName, ok := input["tool_name"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid tool_name in input")
	}
	toolInput, ok := input["tool_input"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing or invalid tool_input in input")
	}

	// Format message for dialog
	message := formatDialogMessage(toolName, toolInput)

	// Show dialog with buttons
	response := dialog.Show(message, []string{"Allow", "Deny"}, "Allow")

	// Parse response for behavior and optional message
	behavior, msg := parseDialogResponse(response)

	// Create output based on user's decision
	output := createHookResponse(behavior, msg)

	return json.NewEncoder(stdout).Encode(output)
}

// parseDialogResponse parses the dialog response which may contain an optional message
// Format: "buttonIndex" or "buttonIndex|message"
// Returns: (behavior, message)
func parseDialogResponse(response string) (string, string) {
	parts := strings.SplitN(response, "|", 2)
	buttonIndex := parts[0]
	message := ""
	if len(parts) > 1 {
		message = parts[1]
	}

	behavior := behaviorDeny
	if buttonIndex == buttonIndexAllow {
		behavior = behaviorAllow
	}

	return behavior, message
}

// createHookResponse creates the JSON response structure for the hook
func createHookResponse(behavior string, message string) map[string]interface{} {
	decision := map[string]interface{}{
		"behavior": behavior,
	}

	// Add interrupt:false and optional message when denying
	if behavior == behaviorDeny {
		decision["interrupt"] = false
		if message != "" {
			decision["message"] = message
		}
	}

	return map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName": hookEventPermissionRequest,
			"decision":      decision,
		},
	}
}

// formatDialogMessage creates a user-friendly message from tool information
func formatDialogMessage(toolName string, toolInput map[string]interface{}) string {
	// For now, just show basic info
	message := "Tool: " + toolName
	if command, ok := toolInput["command"].(string); ok {
		message += "\nCommand: " + command
	}
	return message
}
