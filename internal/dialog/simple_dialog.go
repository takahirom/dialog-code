package dialog

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/takahirom/dialog-code/internal/debug"
)

// SimpleOSDialog provides pure OS dialog functionality without message processing
type SimpleOSDialog struct {
	timeout int // Timeout in seconds (default 60)
}

// NewSimpleOSDialog creates a new simple OS dialog with default 60 second timeout
func NewSimpleOSDialog() *SimpleOSDialog {
	return &SimpleOSDialog{timeout: 60}
}

// SetTimeout sets the dialog timeout in seconds
func (d *SimpleOSDialog) SetTimeout(seconds int) {
	if seconds > 0 {
		d.timeout = seconds
	}
}

// Show displays a dialog with the given message and buttons, returns the selected button text
func (d *SimpleOSDialog) Show(message string, buttons []string, defaultButton string) string {
	if len(buttons) == 0 {
		buttons = []string{"OK"}
		defaultButton = "OK"
	}

	// Choose between dialog types based on button count
	if len(buttons) > 3 {
		debug.Printf("[DEBUG] SimpleOSDialog: Using choose from list for %d buttons\n", len(buttons))
		return d.executeChooseFromListDialog(message, buttons, defaultButton)
	} else {
		debug.Printf("[DEBUG] SimpleOSDialog: Using display dialog for %d buttons\n", len(buttons))
		return d.executeAppleScriptDialog(message, buttons, defaultButton)
	}
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

	// Build AppleScript command with configurable timeout
	script := fmt.Sprintf(`display dialog "%s" with title "Claude Permission" buttons {%s} default button "%s" giving up after %d`,
		escapedMessage, buttonsStr, d.escapeForAppleScript(defaultButton), d.timeout)

	debug.Printf("[DEBUG] SimpleOSDialog: Executing AppleScript: %s\n", script)

	// Execute AppleScript
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		// AppleScript execution failed, default to last button (most restrictive choice)
		maxChoice := fmt.Sprintf("%d", len(buttons))
		debug.Printf("[DEBUG] SimpleOSDialog: AppleScript error: %v, returning \"%s\"\n", err, maxChoice)
		return maxChoice
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
	// Check if dialog timed out (gave up:true)
	if strings.Contains(output, "gave up:true") {
		debug.Printf("[DEBUG] SimpleOSDialog: Dialog timed out, returning empty string\n")
		return ""
	}

	// AppleScript returns "button returned:ButtonName, gave up:false"
	re := regexp.MustCompile(`button returned:([^,]+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		buttonName := strings.TrimSpace(matches[1])

		// Find the matching button and return its index (1-based)
		for i, button := range buttons {
			if button == buttonName || (len(button) > 50 && len(buttonName) >= 47 && strings.HasPrefix(button, buttonName[:47])) {
				return fmt.Sprintf("%d", i+1)
			}
		}
	}

	// Default to last button if parsing fails (most restrictive choice)
	maxChoice := fmt.Sprintf("%d", len(buttons))
	debug.Printf("[DEBUG] SimpleOSDialog: No button match found, returning last button \"%s\"\n", maxChoice)
	return maxChoice
}

// executeChooseFromListDialog executes AppleScript choose from list for many buttons
func (d *SimpleOSDialog) executeChooseFromListDialog(message string, buttons []string, defaultButton string) string {
	// Build button list for AppleScript
	var buttonStrings []string
	for _, button := range buttons {
		buttonStrings = append(buttonStrings, fmt.Sprintf(`"%s"`, d.escapeForAppleScript(button)))
	}
	buttonsStr := strings.Join(buttonStrings, ",")
	
	// Build default selection
	defaultSelection := ""
	if defaultButton != "" {
		valid := false
		for _, b := range buttons {
			if b == defaultButton {
				valid = true
				break
			}
		}
		if valid {
			defaultSelection = fmt.Sprintf(` default items {"%s"}`, d.escapeForAppleScript(defaultButton))
		} else {
			debug.Printf("[DEBUG] SimpleOSDialog: defaultButton %q not in list; omitting default items\n", defaultButton)
		}
	}
	
	// Build AppleScript command for choose from list
	script := fmt.Sprintf(`choose from list {%s} with title "Claude Permission" with prompt "%s"%s`,
		buttonsStr, d.escapeForAppleScript(message), defaultSelection)
	
	debug.Printf("[DEBUG] SimpleOSDialog: Executing choose from list: %s\n", script)
	
	// Execute AppleScript
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		// Choose from list execution failed, default to last button (most restrictive choice)
		maxChoice := fmt.Sprintf("%d", len(buttons))
		debug.Printf("[DEBUG] SimpleOSDialog: Choose from list error: %v, returning \"%s\"\n", err, maxChoice)
		return maxChoice
	}
	
	// Parse the result to find which button was selected
	return d.parseChooseFromListResult(string(output), buttons)
}

// parseChooseFromListResult parses choose from list output to determine which button was selected
func (d *SimpleOSDialog) parseChooseFromListResult(output string, buttons []string) string {
	// choose from list returns selected items (often {"Label"}) or "false" if cancelled
	output = strings.TrimSpace(output)
	
	if output == "false" {
		// User cancelled, return last button (most restrictive)
		debug.Printf("[DEBUG] SimpleOSDialog: User cancelled choose from list, returning last button\n")
		return fmt.Sprintf("%d", len(buttons))
	}
	
	// Normalize: strip surrounding braces, pick first item if multiple, strip quotes
	normalized := output
	if strings.HasPrefix(normalized, "{") && strings.HasSuffix(normalized, "}") {
		normalized = strings.TrimSpace(normalized[1 : len(normalized)-1])
	}
	// If multiple items (shouldn't happen with default settings), take the first
	if idx := strings.Index(normalized, ","); idx >= 0 {
		normalized = normalized[:idx]
	}
	normalized = strings.TrimSpace(normalized)
	normalized = strings.Trim(normalized, `"`)
	
	// Find the matching button and return its index (1-based)
	for i, button := range buttons {
		if button == normalized {
			return fmt.Sprintf("%d", i+1)
		}
	}
	
	// Default to last button if no match found (most restrictive)
	debug.Printf("[DEBUG] SimpleOSDialog: No button match found in choose from list, returning last button\n")
	return fmt.Sprintf("%d", len(buttons))
}