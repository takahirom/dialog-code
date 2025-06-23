package dialog

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/takahirom/dialog-code/internal/choice"
	"github.com/takahirom/dialog-code/internal/types"
)

// Global variables for backward compatibility
var (
	dialogMutex       sync.Mutex
	dialogShowing     bool
	lastDialogTime    time.Time
	waitingForChoice  bool
	choiceResponse    string
	outputMutex       sync.Mutex
	ptmxGlobal        *os.File
	lastPromptLine    string
	collectedChoices  map[string]string
	promptStarted     bool
	processedPrompts  map[string]time.Time
	dialogJustShown   bool
	dialogCooldown    time.Time
)

// ColorStripWriter is a writer that strips ANSI colors before writing
type ColorStripWriter struct {
	Writer io.Writer
}

func (w *ColorStripWriter) Write(p []byte) (n int, err error) {
	// Create pattern inline for stripping
	ansiEscape := `\x1b\[[0-9;?]*[mKHJhlABCDEFGPST]`
	re := regexp.MustCompile(ansiEscape)
	stripped := re.ReplaceAllString(string(p), "")
	return w.Writer.Write([]byte(stripped))
}

// OSDialog implements DialogInterface for macOS dialogs
type OSDialog struct {
	State    *types.DialogState
	Patterns *types.RegexPatterns
}

// NewOSDialog creates a new OS dialog
func NewOSDialog(state *types.DialogState, patterns *types.RegexPatterns) *OSDialog {
	return &OSDialog{
		State:    state,
		Patterns: patterns,
	}
}

// AskWithChoices shows dialog on macOS with collected choices
func (d *OSDialog) AskWithChoices(msg string, choices map[string]string, debugFile *os.File) string {
	if debugFile != nil {
		fmt.Fprintf(debugFile, "[DEBUG] askWithChoices called with msg: %q\n", msg)
		fmt.Fprintf(debugFile, "[DEBUG] askWithChoices choices: %+v\n", choices)
	}
	
	d.State.Mutex.Lock()
	defer d.State.Mutex.Unlock()
	
	// Only prevent if dialog is currently showing
	if d.State.Showing {
		if debugFile != nil {
			fmt.Fprintf(debugFile, "[DEBUG] Dialog prevented: dialogShowing=%v\n", d.State.Showing)
		}
		return "3" // Default to No
	}
	
	if debugFile != nil {
		fmt.Fprintf(debugFile, "[DEBUG] Starting dialog display\n")
	}
	
	cleanMsg := d.Patterns.StripAnsi(msg)
	
	// Build buttons array from collected choices
	var buttons []string
	var buttonToChoice = make(map[string]string)
	
	if debugFile != nil {
		fmt.Fprintf(debugFile, "[DEBUG] Building buttons from %d choices\n", len(choices))
	}
	
	// Add choices in order (1, 2, 3...)
	for i := 1; i <= 3; i++ {
		numStr := fmt.Sprintf("%d", i)
		if choice, exists := choices[numStr]; exists {
			// Clean up the choice text - remove number prefix and formatting
			cleanChoice := strings.TrimSpace(choice)
			// Remove pipe characters and extra decorative characters
			cleanChoice = strings.Trim(cleanChoice, "│ \t")
			cleanChoice = strings.TrimRight(cleanChoice, "│ \t\r\n\u00A0\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200A\u200B\u202F\u205F\u3000◯○◉●>?─━┌┐└┘├┤┬┴┼╭╮╯╰╠╣╦╩╬⧉")
			cleanChoice = strings.TrimSpace(cleanChoice)
			cleanChoice = strings.ReplaceAll(cleanChoice, `"`, `\"`)
			// Extract just the meaningful part (after the number and dot)
			if parts := strings.SplitN(cleanChoice, ". ", 2); len(parts) > 1 {
				cleanChoice = parts[1]
			}
			buttons = append(buttons, cleanChoice)
			buttonToChoice[cleanChoice] = numStr
			if debugFile != nil {
				fmt.Fprintf(debugFile, "[DEBUG] Added button %s: %q -> %s\n", numStr, cleanChoice, numStr)
			}
		}
	}
	
	if len(buttons) == 0 {
		if debugFile != nil {
			fmt.Fprintf(debugFile, "[DEBUG] No buttons found, not showing dialog (waiting for choices)\n")
		}
		// Don't set dialogShowing or cooldown for empty dialogs
		return "" // Return empty string to indicate "don't respond yet"
	}
	
	// Now that we have choices, set dialog state and prepare cleanup
	d.State.Showing = true
	defer func() {
		d.State.Showing = false
		d.State.LastTime = time.Now()
		d.State.JustShown = true
		d.State.Cooldown = time.Now()
		if debugFile != nil {
			fmt.Fprintf(debugFile, "[DEBUG] Dialog finished, cooldown activated\n")
		}
	}()
	
	// Build AppleScript with dynamic buttons
	buttonsStr := `{"` + strings.Join(buttons, `","`) + `"}`
	script := `display dialog "` + strings.ReplaceAll(cleanMsg, `"`, `\"`) + `" with title "Claude Permission" buttons ` + buttonsStr + ` default button "` + buttons[0] + `"`
	
	if debugFile != nil {
		fmt.Fprintf(debugFile, "[DEBUG] Executing AppleScript: %s\n", script)
	}
	
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		if debugFile != nil {
			fmt.Fprintf(debugFile, "[DEBUG] AppleScript error: %v\n", err)
		}
		return "3" // Default to No on error
	}
	
	if debugFile != nil {
		fmt.Fprintf(debugFile, "[DEBUG] AppleScript output: %q\n", string(out))
	}
	
	// Parse which button was clicked
	outStr := string(out)
	for button, choiceNum := range buttonToChoice {
		if strings.Contains(outStr, "button returned:"+button) {
			if debugFile != nil {
				fmt.Fprintf(debugFile, "[DEBUG] Button %q clicked, returning %s\n", button, choiceNum)
			}
			return choiceNum
		}
	}
	
	if debugFile != nil {
		fmt.Fprintf(debugFile, "[DEBUG] No button match found, returning default\n")
	}
	return "3" // Default to No
}

// DialogOptions configures how the dialog should be displayed
type DialogOptions struct {
	Message       string
	Choices       map[string]string
	Context       []string
	TriggerReason string
	TriggerLine   string
}

// Show dialog on macOS with collected choices and return selected choice number
func AskWithChoices(msg string, choices map[string]string) string {
	return askWithOptions(DialogOptions{
		Message: msg,
		Choices: choices,
	})
}

// Show dialog on macOS with collected choices and context information
func AskWithChoicesAndContext(msg string, choices map[string]string, context []string) string {
	return askWithOptions(DialogOptions{
		Message: msg,
		Choices: choices,
		Context: context,
	})
}

// Show dialog on macOS with collected choices, context information, and reason details
func AskWithChoicesContextAndReason(msg string, choices map[string]string, context []string, triggerReason string, triggerLine string) string {
	return askWithOptions(DialogOptions{
		Message:       msg,
		Choices:       choices,
		Context:       context,
		TriggerReason: triggerReason,
		TriggerLine:   triggerLine,
	})
}

// askWithOptions is the unified function that handles all dialog variations
func askWithOptions(opts DialogOptions) string {
	debugFile, _ := os.OpenFile("debug_output.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	defer debugFile.Close()
	
	fmt.Fprintf(debugFile, "[DEBUG] askWithOptions called with msg: %q\n", opts.Message)
	if opts.TriggerReason != "" {
		fmt.Fprintf(debugFile, "[DEBUG] triggerReason: %q, triggerLine: %q\n", opts.TriggerReason, opts.TriggerLine)
	}
	fmt.Fprintf(debugFile, "[DEBUG] choices: %+v\n", opts.Choices)
	fmt.Fprintf(debugFile, "[DEBUG] context: %+v\n", opts.Context)
	
	dialogMutex.Lock()
	defer dialogMutex.Unlock()
	
	if dialogShowing {
		fmt.Fprintf(debugFile, "[DEBUG] Dialog prevented: dialogShowing=%v\n", dialogShowing)
		return "3" // Default to No
	}
	
	fmt.Fprintf(debugFile, "[DEBUG] Starting dialog display\n")
	
	// Create patterns inline
	ansiEscape := regexp.MustCompile(`\x1b\[[0-9;?]*[mKHJhlABCDEFGPST]`)
	regexPatterns := &types.RegexPatterns{AnsiEscape: ansiEscape}
	
	// Choose appropriate message formatting based on available options
	var cleanMsg string
	if opts.TriggerReason != "" {
		cleanMsg = choice.GetContextualMessageWithReason(opts.Message, opts.Context, opts.TriggerReason, opts.TriggerLine, regexPatterns)
	} else {
		cleanMsg = choice.GetContextualMessage(opts.Message, opts.Context, regexPatterns)
	}
	
	// Build buttons array from collected choices
	var buttons []string
	var buttonToChoice = make(map[string]string)
	
	fmt.Fprintf(debugFile, "[DEBUG] Building buttons from %d choices\n", len(opts.Choices))
	
	// Add choices in order (1, 2, 3...)
	for i := 1; i <= 3; i++ {
		numStr := fmt.Sprintf("%d", i)
		if choice, exists := opts.Choices[numStr]; exists {
			cleanChoice := cleanChoiceText(choice)
			buttons = append(buttons, cleanChoice)
			buttonToChoice[cleanChoice] = numStr
			fmt.Fprintf(debugFile, "[DEBUG] Added button %s: %q -> %s\n", numStr, cleanChoice, numStr)
		}
	}
	
	if len(buttons) == 0 {
		fmt.Fprintf(debugFile, "[DEBUG] No buttons found, not showing dialog (waiting for choices)\n")
		return ""
	}
	
	// Set dialog state and prepare cleanup
	dialogShowing = true
	defer func() {
		dialogShowing = false
		lastDialogTime = time.Now()
		dialogJustShown = true
		dialogCooldown = time.Now()
		fmt.Fprintf(debugFile, "[DEBUG] Dialog finished, cooldown activated\n")
	}()
	
	// Execute AppleScript dialog
	return executeAppleScriptDialog(cleanMsg, buttons, buttonToChoice, debugFile)
}

// cleanChoiceText removes formatting from choice text
func cleanChoiceText(choice string) string {
	cleanChoice := strings.TrimSpace(choice)
	cleanChoice = strings.Trim(cleanChoice, "│ \t")
	cleanChoice = strings.TrimRight(cleanChoice, "│ \t\r\n\u00A0\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200A\u200B\u202F\u205F\u3000◯○◉●>?─━┌┐└┘├┤┬┴┼╭╮╯╰╠╣╦╩╬⧉")
	cleanChoice = strings.TrimSpace(cleanChoice)
	cleanChoice = strings.ReplaceAll(cleanChoice, `"`, `\"`)
	// Extract just the meaningful part (after the number and dot)
	if parts := strings.SplitN(cleanChoice, ". ", 2); len(parts) > 1 {
		cleanChoice = parts[1]
	}
	return cleanChoice
}

// executeAppleScriptDialog executes the AppleScript and returns the choice
func executeAppleScriptDialog(cleanMsg string, buttons []string, buttonToChoice map[string]string, debugFile *os.File) string {
	// Clean the message to avoid AppleScript parsing issues
	cleanMsg = sanitizeMessageForAppleScript(cleanMsg)
	
	// Ensure buttons don't contain problematic characters
	var cleanButtons []string
	cleanButtonToChoice := make(map[string]string)
	for _, button := range buttons {
		cleanButton := sanitizeButtonForAppleScript(button)
		cleanButtons = append(cleanButtons, cleanButton)
		cleanButtonToChoice[cleanButton] = buttonToChoice[button]
	}
	
	buttonsStr := `{"` + strings.Join(cleanButtons, `","`) + `"}`
	script := `display dialog "` + cleanMsg + `" with title "Claude Permission" buttons ` + buttonsStr + ` default button "` + cleanButtons[0] + `"`
	
	fmt.Fprintf(debugFile, "[DEBUG] Executing AppleScript: %s\n", script)
	
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		fmt.Fprintf(debugFile, "[DEBUG] AppleScript error: %v, output: %s\n", err, string(out))
		return "3" // Default to No on error
	}
	
	fmt.Fprintf(debugFile, "[DEBUG] AppleScript output: %q\n", string(out))
	
	// Parse which button was clicked
	outStr := string(out)
	for button, choiceNum := range cleanButtonToChoice {
		if strings.Contains(outStr, "button returned:"+button) {
			fmt.Fprintf(debugFile, "[DEBUG] Button %q clicked, returning %s\n", button, choiceNum)
			return choiceNum
		}
	}
	
	fmt.Fprintf(debugFile, "[DEBUG] No button match found, returning default\n")
	return "3" // Default to No
}

// sanitizeMessageForAppleScript cleans up message text to avoid AppleScript parsing issues
func sanitizeMessageForAppleScript(msg string) string {
	// Remove or replace problematic characters that cause AppleScript parsing errors
	msg = strings.ReplaceAll(msg, `"`, `'`)  // Replace double quotes with single quotes
	msg = strings.ReplaceAll(msg, `\`, ``)   // Remove backslashes
	msg = strings.ReplaceAll(msg, "\r", " ") // Replace carriage returns
	msg = strings.ReplaceAll(msg, "\t", " ") // Replace tabs
	
	// Remove pipe characters and other special formatting that cause issues
	msg = strings.ReplaceAll(msg, "|", " ")
	msg = strings.ReplaceAll(msg, "│", " ")
	msg = strings.ReplaceAll(msg, "\\|", " ") // Escaped pipes
	
	// Remove emojis and special Unicode characters that might cause issues
	msg = strings.ReplaceAll(msg, "🔒", "")
	msg = strings.ReplaceAll(msg, "⏺", "")
	
	// Handle Japanese characters and other special regex characters that cause AppleScript issues
	msg = strings.ReplaceAll(msg, "シーク", "seek")
	msg = strings.ReplaceAll(msg, "\\;", ";")
	msg = strings.ReplaceAll(msg, "\\{", "{")
	msg = strings.ReplaceAll(msg, "\\}", "}")
	
	// Simplify complex file paths and commands to avoid AppleScript parsing issues
	if strings.Contains(msg, "find ") && strings.Contains(msg, "-exec") {
		// Simplify find commands to just show the essential info
		msg = "Execute find command in project directory?"
	}
	
	// Clean up multiple spaces
	for strings.Contains(msg, "  ") {
		msg = strings.ReplaceAll(msg, "  ", " ")
	}
	
	return strings.TrimSpace(msg)
}

// sanitizeButtonForAppleScript cleans up button text to avoid AppleScript parsing issues
func sanitizeButtonForAppleScript(button string) string {
	// Replace problematic characters in button text
	button = strings.ReplaceAll(button, `"`, `'`)
	button = strings.ReplaceAll(button, `\`, ``)
	button = strings.ReplaceAll(button, "\n", " ")
	button = strings.ReplaceAll(button, "\r", " ")
	button = strings.ReplaceAll(button, "\t", " ")
	
	// Limit button text length
	if len(button) > 50 {
		button = button[:47] + "..."
	}
	
	return strings.TrimSpace(button)
}

// Send input after output stabilizes
func SendDelayedInput() {
	outputMutex.Lock()
	defer outputMutex.Unlock()
	
	if waitingForChoice && choiceResponse != "" {
		_, _ = ptmxGlobal.WriteString(choiceResponse)
		waitingForChoice = false
		choiceResponse = ""
	}
}

// Initialize global variables
func InitGlobals() {
	collectedChoices = make(map[string]string)
	processedPrompts = make(map[string]time.Time)
}

// SetPtmxGlobal sets the global ptmx file
func SetPtmxGlobal(ptmx *os.File) {
	ptmxGlobal = ptmx
}