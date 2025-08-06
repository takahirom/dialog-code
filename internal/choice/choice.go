package choice

import (
	"fmt"
	"strings"

	"github.com/takahirom/dialog-code/internal/types"
)

// cleanDialogText removes pipe characters, unicode whitespace, and dialog box decorations from text
func cleanDialogText(text string) string {
	cleanText := strings.Trim(text, "│ \t")
	cleanText = strings.TrimRight(cleanText, "│ \t\r\n\u00A0\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200A\u200B\u202F\u205F\u3000◯○◉●>─━┌┐└┘├┤┬┴┼╭╮╯╰╠╣╦╩╬⧉")
	return strings.TrimSpace(cleanText)
}

// GetBestChoice determines the best choice number based on collected choices
func GetBestChoice(choices map[string]string, regexPatterns *types.RegexPatterns) string {
	// For Claude permissions: Priority is "Allow" > first available choice
	for num, text := range choices {
		if regexPatterns.ChoiceYes.MatchString(text) {
			return num
		}
	}

	// Look for "Add a new rule" as second choice (often choice 1)
	for num, text := range choices {
		if strings.Contains(text, "Add a new rule") {
			return num
		}
	}

	// Fallback to the first available choice
	for num := 1; num <= 10; num++ {
		numStr := fmt.Sprintf("%d", num)
		if _, exists := choices[numStr]; exists {
			return numStr
		}
	}

	// Ultimate fallback
	return "1"
}

// GetBestChoiceFromState determines the best choice number based on app state
func GetBestChoiceFromState(state *types.AppState, regexPatterns *types.RegexPatterns) string {
	return GetBestChoice(state.Prompt.CollectedChoices, regexPatterns)
}

// GetContextualMessage builds a more informative dialog message with context
func GetContextualMessage(prompt string, context []string, regexPatterns *types.RegexPatterns) string {
	// Remove pipe characters and extra whitespace from the main prompt
	cleanPrompt := cleanDialogText(regexPatterns.StripAnsi(prompt))

	// Start with the main prompt
	message := cleanPrompt

	// Add context if available
	if len(context) > 0 {
		message += "\n\nContext:\n"
		for _, contextLine := range context {
			// Clean up the context line by removing pipe characters and extra whitespace
			cleanContextLine := cleanDialogText(contextLine)
			if len(cleanContextLine) > 0 {
				message += "• " + cleanContextLine + "\n"
			}
		}
	}

	return message
}

// GetContextualMessageWithReason builds a dialog message with context and reason information
func GetContextualMessageWithReason(prompt string, context []string, triggerReason string, triggerLine string, regexPatterns *types.RegexPatterns) string {
	// Remove pipe characters and extra whitespace from the main prompt
	cleanPrompt := cleanDialogText(regexPatterns.StripAnsi(prompt))

	// Start with reason information
	message := "🔒 " + triggerReason + "\n\n" + cleanPrompt

	// Add trigger line if different from prompt
	if triggerLine != prompt && strings.TrimSpace(regexPatterns.StripAnsi(triggerLine)) != strings.TrimSpace(cleanPrompt) {
		cleanTrigger := cleanDialogText(regexPatterns.StripAnsi(triggerLine))
		if len(cleanTrigger) > 0 {
			message += "\n\nTriggered by: " + cleanTrigger
		}
	}

	// Add context if available
	if len(context) > 0 {
		message += "\n\nContext:\n"
		for _, contextLine := range context {
			// Clean up the context line by removing pipe characters and extra whitespace
			cleanContextLine := cleanDialogText(contextLine)
			if len(cleanContextLine) > 0 {
				message += "• " + cleanContextLine + "\n"
			}
		}
	}

	return message
}

// safeStripAnsi safely strips ANSI codes with nil checks
func safeStripAnsi(text string, regexPatterns *types.RegexPatterns) string {
	if regexPatterns != nil && regexPatterns.AnsiEscape != nil {
		return regexPatterns.StripAnsi(text)
	}
	return text
}

// extractTriggerText finds the trigger text from context or fallback line
func extractTriggerText(context []string, triggerLine string, regexPatterns *types.RegexPatterns) string {
	// Extract trigger text from context (first line that looks like trigger)
	if len(context) > 0 {
		for _, line := range context {
			cleanLine := safeStripAnsi(line, regexPatterns)
			cleanLine = strings.TrimSpace(cleanLine)
			if strings.HasPrefix(cleanLine, "⏺") {
				return cleanLine
			}
		}
	}
	
	// Fallback to triggerLine if no trigger found in context
	if triggerLine != "" {
		triggerText := safeStripAnsi(triggerLine, regexPatterns)
		triggerText = cleanDialogText(triggerText) // Clean pipe characters and decorations
		return strings.TrimSpace(triggerText)
	}
	
	return ""
}

// DialogBoxInfo holds parsed dialog box information
type DialogBoxInfo struct {
	CommandType    string
	CommandDetails []string
	QuestionLine   string
}

// parseDialogBox extracts command information from dialog box context
func parseDialogBox(context []string, regexPatterns *types.RegexPatterns) DialogBoxInfo {
	// Extract command information from context (contains the full dialog box)
	dialogText := ""
	for _, line := range context {
		if strings.Contains(line, "╭") || strings.Contains(line, "│") || strings.Contains(line, "╰") {
			dialogText += line + "\n"
		}
	}
	
	// Parse the dialog box content to extract command type and details
	info := DialogBoxInfo{
		CommandDetails: []string{},
	}
	
	lines := strings.Split(dialogText, "\n")
	inDialog := false
	
	for _, line := range lines {
		cleanLine := safeStripAnsi(line, regexPatterns)
		cleanLine = strings.Trim(cleanLine, "│ \t╭╮╰╯─")
		cleanLine = strings.TrimSpace(cleanLine)
		
		if strings.Contains(line, "╭") || strings.Contains(line, "┌") {
			inDialog = true
			continue
		}
		if strings.Contains(line, "╰") || strings.Contains(line, "└") {
			inDialog = false
			continue
		}
		
		if !inDialog || cleanLine == "" {
			continue
		}
		
		// Detect command type (first non-empty line in dialog)
		if info.CommandType == "" && cleanLine != "" {
			info.CommandType = cleanDialogText(cleanLine) // Additional cleaning
			continue
		}
		
		// Check if this line contains the question
		if strings.Contains(cleanLine, "Do you want to proceed?") || 
		   strings.Contains(cleanLine, "proceed?") ||
		   strings.Contains(cleanLine, "continue?") {
			info.QuestionLine = cleanDialogText(cleanLine) // Additional cleaning
			continue
		}
		
		// Skip choice lines (starting with numbers or bullets)
		if strings.HasPrefix(cleanLine, "1.") || strings.HasPrefix(cleanLine, "2.") || 
		   strings.HasPrefix(cleanLine, "3.") || strings.HasPrefix(cleanLine, "❯") ||
		   strings.HasPrefix(cleanLine, "•") {
			continue
		}
		
		// Collect command details
		if cleanLine != "" {
			info.CommandDetails = append(info.CommandDetails, cleanDialogText(cleanLine)) // Additional cleaning
		}
	}
	
	return info
}

// formatCleanMessage builds the final clean dialog message format
func formatCleanMessage(triggerText, timestamp, triggerReason string, dialogInfo DialogBoxInfo) string {
	var messageParts []string
	
	// Add trigger information
	if triggerText != "" {
		messageParts = append(messageParts, "Trigger text: "+triggerText)
	}
	
	// Add timestamp
	if timestamp != "" {
		messageParts = append(messageParts, "Trigger timestamp: "+timestamp)
	}
	
	// Add reason
	if triggerReason != "" {
		messageParts = append(messageParts, "Reason: "+triggerReason)
	}
	
	// Add separator
	messageParts = append(messageParts, "───────────────────────────────────")
	
	// Add command type
	if dialogInfo.CommandType != "" {
		messageParts = append(messageParts, dialogInfo.CommandType)
		messageParts = append(messageParts, "") // Empty line
	}
	
	// Add command details with proper indentation
	for _, detail := range dialogInfo.CommandDetails {
		messageParts = append(messageParts, "  "+detail)
	}
	
	if len(dialogInfo.CommandDetails) > 0 {
		messageParts = append(messageParts, "") // Empty line after details
	}
	
	// Add the question
	questionLine := dialogInfo.QuestionLine
	if questionLine == "" {
		questionLine = "Do you want to proceed?"
	}
	messageParts = append(messageParts, questionLine)
	
	return strings.Join(messageParts, "\n")
}

// GetCleanDialogMessage creates a clean, organized dialog message format
// This function extracts context information and presents it in a structured way
// without the "Context:" header for dialog display
func GetCleanDialogMessage(prompt string, context []string, triggerReason string, triggerLine string, timestamp string, regexPatterns *types.RegexPatterns) string {
	triggerText := extractTriggerText(context, triggerLine, regexPatterns)
	dialogInfo := parseDialogBox(context, regexPatterns)
	return formatCleanMessage(triggerText, timestamp, triggerReason, dialogInfo)
}

// ParseDialogBox extracts command information from dialog box context (public wrapper)
func ParseDialogBox(context []string, regexPatterns *types.RegexPatterns) DialogBoxInfo {
	return parseDialogBox(context, regexPatterns)
}
