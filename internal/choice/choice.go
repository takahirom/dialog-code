package choice

import (
	"fmt"
	"os"
	"strings"

	"github.com/takahirom/dialog-code/internal/types"
)

// GetBestChoice determines the best choice number based on collected choices
func GetBestChoice(choices map[string]string, regexPatterns *types.RegexPatterns, debugFile *os.File) string {
	// For Claude permissions: Priority is "Allow" > first available choice
	for num, text := range choices {
		if regexPatterns.ChoiceYes.MatchString(text) {
			if debugFile != nil {
				fmt.Fprintf(debugFile, "[DEBUG] Found 'Allow/Yes' at choice %s: %s\n", num, text)
			}
			return num
		}
	}
	
	// Look for "Add a new rule" as second choice (often choice 1)
	for num, text := range choices {
		if strings.Contains(text, "Add a new rule") {
			if debugFile != nil {
				fmt.Fprintf(debugFile, "[DEBUG] Found 'Add a new rule' at choice %s: %s\n", num, text)
			}
			return num
		}
	}
	
	// Fallback to the first available choice
	for num := 1; num <= 10; num++ {
		numStr := fmt.Sprintf("%d", num)
		if _, exists := choices[numStr]; exists {
			if debugFile != nil {
				fmt.Fprintf(debugFile, "[DEBUG] Fallback to first available choice %s\n", numStr)
			}
			return numStr
		}
	}
	
	// Ultimate fallback
	if debugFile != nil {
		fmt.Fprintf(debugFile, "[DEBUG] No choices found, defaulting to 1\n")
	}
	return "1"
}

// GetBestChoiceFromState determines the best choice number based on app state
func GetBestChoiceFromState(state *types.AppState, regexPatterns *types.RegexPatterns, debugFile *os.File) string {
	return GetBestChoice(state.Prompt.CollectedChoices, regexPatterns, debugFile)
}

// GetContextualMessage builds a more informative dialog message with context
func GetContextualMessage(prompt string, context []string, regexPatterns *types.RegexPatterns) string {
	cleanPrompt := regexPatterns.StripAnsi(prompt)
	// Remove pipe characters and extra whitespace from the main prompt
	cleanPrompt = strings.Trim(cleanPrompt, "│ \t")
	cleanPrompt = strings.TrimRight(cleanPrompt, "│ \t\r\n\u00A0\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200A\u200B\u202F\u205F\u3000◯○◉●>─━┌┐└┘├┤┬┴┼╭╮╯╰╠╣╦╩╬⧉")
	cleanPrompt = strings.TrimSpace(cleanPrompt)
	
	// Start with the main prompt
	message := cleanPrompt
	
	// Add context if available
	if len(context) > 0 {
		message += "\n\nContext:\n"
		for _, contextLine := range context {
			// Include meaningful lines - be more inclusive to show more context
			if strings.Contains(contextLine, "Write(") || 
			   strings.Contains(contextLine, "Read(") || 
			   strings.Contains(contextLine, "Bash(") ||
			   strings.Contains(contextLine, "file") ||
			   strings.Contains(contextLine, "permission") ||
			   strings.Contains(contextLine, "Error") ||
			   strings.Contains(contextLine, "Failed") ||
			   strings.Contains(contextLine, "Success") ||
			   strings.Contains(contextLine, "Search(") ||
			   strings.Contains(contextLine, "dependencies") ||
			   strings.Contains(contextLine, "module") ||
			   strings.Contains(contextLine, "Check") ||
			   len(contextLine) > 25 { // Include lines over 25 chars as they're likely meaningful
				// Clean up the context line by removing pipe characters and extra whitespace
				cleanContextLine := strings.Trim(contextLine, "│ \t")
				cleanContextLine = strings.TrimRight(cleanContextLine, "│ \t\r\n\u00A0\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200A\u200B\u202F\u205F\u3000◯○◉●>?─━┌┐└┘├┤┬┴┼╭╮╯╰╠╣╦╩╬⧉")
				cleanContextLine = strings.TrimSpace(cleanContextLine)
				message += "• " + cleanContextLine + "\n"
			}
		}
	}
	
	return message
}

// GetContextualMessageWithReason builds a dialog message with context and reason information
func GetContextualMessageWithReason(prompt string, context []string, triggerReason string, triggerLine string, regexPatterns *types.RegexPatterns) string {
	cleanPrompt := regexPatterns.StripAnsi(prompt)
	// Remove pipe characters and extra whitespace from the main prompt
	cleanPrompt = strings.Trim(cleanPrompt, "│ \t")
	cleanPrompt = strings.TrimRight(cleanPrompt, "│ \t\r\n\u00A0\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200A\u200B\u202F\u205F\u3000◯○◉●>─━┌┐└┘├┤┬┴┼╭╮╯╰╠╣╦╩╬⧉")
	cleanPrompt = strings.TrimSpace(cleanPrompt)
	
	// Start with reason information
	message := "🔒 " + triggerReason + "\n\n" + cleanPrompt
	
	// Add trigger line if different from prompt
	if triggerLine != prompt && strings.TrimSpace(regexPatterns.StripAnsi(triggerLine)) != strings.TrimSpace(cleanPrompt) {
		cleanTrigger := regexPatterns.StripAnsi(triggerLine)
		cleanTrigger = strings.Trim(cleanTrigger, "│ \t")
		cleanTrigger = strings.TrimSpace(cleanTrigger)
		if len(cleanTrigger) > 0 {
			message += "\n\nTriggered by: " + cleanTrigger
		}
	}
	
	// Add context if available
	if len(context) > 0 {
		message += "\n\nContext:\n"
		for _, contextLine := range context {
			// Include meaningful lines - be more inclusive to show more context
			if strings.Contains(contextLine, "Write(") || 
			   strings.Contains(contextLine, "Read(") || 
			   strings.Contains(contextLine, "Bash(") ||
			   strings.Contains(contextLine, "file") ||
			   strings.Contains(contextLine, "permission") ||
			   strings.Contains(contextLine, "Error") ||
			   strings.Contains(contextLine, "Failed") ||
			   strings.Contains(contextLine, "Success") ||
			   strings.Contains(contextLine, "Search(") ||
			   strings.Contains(contextLine, "dependencies") ||
			   strings.Contains(contextLine, "module") ||
			   strings.Contains(contextLine, "Check") ||
			   len(contextLine) > 25 { // Include lines over 25 chars as they're likely meaningful
				// Clean up the context line by removing pipe characters and extra whitespace
				cleanContextLine := strings.Trim(contextLine, "│ \t")
				cleanContextLine = strings.TrimRight(cleanContextLine, "│ \t\r\n\u00A0\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200A\u200B\u202F\u205F\u3000◯○◉●>?─━┌┐└┘├┤┬┴┼╭╮╯╰╠╣╦╩╬⧉")
				cleanContextLine = strings.TrimSpace(cleanContextLine)
				message += "• " + cleanContextLine + "\n"
			}
		}
	}
	
	return message
}