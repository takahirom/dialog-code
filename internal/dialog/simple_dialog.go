package dialog

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// SimpleOSDialog provides pure OS dialog functionality without message processing
type SimpleOSDialog struct{}

// NewSimpleOSDialog creates a new simple OS dialog
func NewSimpleOSDialog() *SimpleOSDialog {
	return &SimpleOSDialog{}
}

// Show displays a dialog with the given message and buttons, returns the selected button text
func (d *SimpleOSDialog) Show(message string, buttons []string, defaultButton string) string {
	if len(buttons) == 0 {
		buttons = []string{"OK"}
		defaultButton = "OK"
	}

	// Execute AppleScript dialog
	return d.executeAppleScriptDialog(message, buttons, defaultButton)
}

// executeAppleScriptDialog executes the actual AppleScript dialog
func (d *SimpleOSDialog) executeAppleScriptDialog(message string, buttons []string, defaultButton string) string {
	// Escape message for AppleScript
	escapedMessage := d.escapeForAppleScript(message)
	
	// Build buttons string for AppleScript
	var buttonStrings []string
	for _, button := range buttons {
		// Truncate very long button text for macOS dialog limits
		if len(button) > 50 {
			button = button[:47] + "..."
		}
		buttonStrings = append(buttonStrings, fmt.Sprintf(`"%s"`, d.escapeForAppleScript(button)))
	}
	buttonsStr := strings.Join(buttonStrings, ",")
	
	// Build AppleScript command
	script := fmt.Sprintf(`display dialog "%s" with title "Claude Permission" buttons {%s} default button "%s"`,
		escapedMessage, buttonsStr, d.escapeForAppleScript(defaultButton))
	
	
	// Execute AppleScript
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		// AppleScript execution failed, default to first button (safe choice)
		return "1"
	}
	
	
	// Parse the result to find which button was clicked
	return d.parseAppleScriptResult(string(output), buttons)
}

// escapeForAppleScript escapes special characters for AppleScript strings
func (d *SimpleOSDialog) escapeForAppleScript(text string) string {
	// Replace quotes and backslashes
	text = strings.ReplaceAll(text, `\`, `\\`)
	text = strings.ReplaceAll(text, `"`, `\"`)
	return text
}

// parseAppleScriptResult parses AppleScript output to determine which button was clicked
func (d *SimpleOSDialog) parseAppleScriptResult(output string, buttons []string) string {
	// AppleScript returns "button returned:ButtonName"
	re := regexp.MustCompile(`button returned:(.+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		buttonName := strings.TrimSpace(matches[1])
		
		// Find the matching button and return its index (1-based)
		for i, button := range buttons {
			if button == buttonName || (len(button) > 50 && strings.HasPrefix(button, buttonName[:47])) {
				return fmt.Sprintf("%d", i+1)
			}
		}
	}
	
	// Default to first button if parsing fails
	return "1"
}