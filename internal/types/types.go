package types

import (
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	// Timing constants for types package
	PromptProcessingCooldownMs = 500
	PromptDuplicationSeconds   = 5
	DefaultContextLines        = 10
)

// DialogState holds the state for permission dialogs
type DialogState struct {
	Mutex     sync.Mutex
	Showing   bool
	LastTime  time.Time
	JustShown bool
	Cooldown  time.Time
}

// PromptState holds the state for prompt processing
type PromptState struct {
	LastLine         string
	Started          bool
	CollectedChoices map[string]string
	Processed        map[string]time.Time
	JustShown        bool
	Cooldown         time.Time
	Context          []string // Store context lines before the prompt
	ContextLines     int      // Number of context lines to collect
	TriggerReason    string   // What triggered this dialog (e.g., "Write()", "Bash()", etc.)
	TriggerLine      string   // The exact line that triggered the dialog
}

// AppState holds the global application state
type AppState struct {
	Dialog           *DialogState
	Prompt           *PromptState
	WaitingForChoice bool
	ChoiceResponse   string
	OutputTimer      *time.Timer
	OutputMutex      sync.Mutex
	Ptmx             *os.File
	AutoApprove      bool
	StripColors      bool
}

// NewAppState creates a new application state
func NewAppState() *AppState {
	return &AppState{
		Dialog: &DialogState{},
		Prompt: &PromptState{
			CollectedChoices: make(map[string]string),
			Processed:        make(map[string]time.Time),
			Context:          make([]string, 0),
			ContextLines:     DefaultContextLines,
		},
	}
}

// RegexPatterns is needed for method signatures
type RegexPatterns struct {
	Permit              *regexp.Regexp
	ChoiceYes           *regexp.Regexp
	ChoiceYesAndDontAsk *regexp.Regexp
	ChoiceNo            *regexp.Regexp
	ChoiceAny           *regexp.Regexp
	AnsiEscape          *regexp.Regexp
}

// NewRegexPatterns creates a new instance of regex patterns
func NewRegexPatterns() *RegexPatterns {
	return &RegexPatterns{
		Permit: regexp.MustCompile(
			`Do you want to`),
		ChoiceYes:           regexp.MustCompile(`.*?([0-9]+)\.\s+(.*(Allow|Yes|Approve).*)`),
		ChoiceYesAndDontAsk: regexp.MustCompile(`.*?([0-9]+)\.\s+(.*(Allow|Yes).*don't ask.*)`),
		ChoiceNo:            regexp.MustCompile(`.*?([0-9]+)\.\s+(.*(Deny|No|Cancel).*)`),
		ChoiceAny:           regexp.MustCompile(`[│\s]*[❯\s]*([0-9]+)\.\s+(.+?)(?:\s*│)?$`),
		AnsiEscape:          regexp.MustCompile(`\x1b\[[0-9;?]*[mKHJhlABCDEFGPST]`),
	}
}

// StripAnsi removes ANSI escape sequences from a string
func (r *RegexPatterns) StripAnsi(s string) string {
	return r.AnsiEscape.ReplaceAllString(s, "")
}


// ShouldProcessPrompt determines if a prompt should be processed based on cooldown and duplicate detection
func (state *AppState) ShouldProcessPrompt(prompt string, regexPatterns *RegexPatterns) bool {
	cleanPrompt := regexPatterns.StripAnsi(prompt)
	
	// Check if we've already processed this exact prompt recently
	if lastProcessed, exists := state.Prompt.Processed[cleanPrompt]; exists {
		if time.Since(lastProcessed) < PromptDuplicationSeconds*time.Second {
			return false
		}
		// If more than specified time has passed, allow reprocessing
		delete(state.Prompt.Processed, cleanPrompt)
	}
	
	// Check cooldown - but only if we actually showed a dialog recently
	if state.Prompt.JustShown && time.Since(state.Prompt.Cooldown) < PromptProcessingCooldownMs*time.Millisecond {
		return false
	}
	
	// Mark this prompt as processed with current timestamp
	state.Prompt.Processed[cleanPrompt] = time.Now()
	return true
}

// AddContextLine adds a line to the context buffer
func (state *AppState) AddContextLine(line string, regexPatterns *RegexPatterns) {
	cleanLine := regexPatterns.StripAnsi(line)
	// Skip empty lines and debug lines
	if len(strings.TrimSpace(cleanLine)) == 0 || strings.HasPrefix(cleanLine, "[DEBUG]") {
		return
	}
	
	// Add to context buffer
	state.Prompt.Context = append(state.Prompt.Context, cleanLine)
	
	// Keep only the last N lines
	if len(state.Prompt.Context) > state.Prompt.ContextLines {
		state.Prompt.Context = state.Prompt.Context[1:]  // Remove first element
	}
}

// StartPromptCollection starts collecting choices for a new prompt
func (state *AppState) StartPromptCollection(prompt string) {
	state.Prompt.LastLine = prompt
	state.Prompt.Started = true
	state.Prompt.CollectedChoices = make(map[string]string) // Reset choices
	state.Prompt.TriggerLine = prompt
	state.Prompt.TriggerReason = state.identifyTriggerReason(prompt, state.Prompt.Context)
}

// identifyTriggerReason determines what triggered the dialog based on the prompt line and context
func (state *AppState) identifyTriggerReason(prompt string, context []string) string {
	// Combine prompt and context for analysis
	fullContext := prompt
	for _, line := range context {
		fullContext += " " + line
	}
	
	// Check for specific function call patterns first
	if strings.Contains(fullContext, "Write(") {
		return "Write() function call"
	}
	if strings.Contains(fullContext, "Bash(") || (strings.Contains(fullContext, "⏺") && strings.Contains(fullContext, "Bash")) {
		// Extract command from Bash() call if present
		for _, line := range context {
			if strings.Contains(line, "⏺") && strings.Contains(line, "Bash(") {
				return "Bash command execution"
			}
		}
		return "Bash() function call"
	}
	// Check for operation permission patterns with record symbol
	if strings.Contains(fullContext, "⏺") && strings.Contains(fullContext, "Write") {
		return "Write operation permission"
	}
	// Check for general permission patterns
	if strings.Contains(fullContext, "requires permission") {
		return "General permission requirement"
	}
	if strings.Contains(fullContext, "needs your approval") {
		return "Approval request"
	}
	if strings.Contains(fullContext, "Permissions:") {
		return "Permission list dialog"
	}
	if strings.Contains(prompt, "Do you want to proceed") {
		return "Proceed confirmation"
	}
	return "Unknown trigger"
}

// AddChoice adds a choice to the current prompt collection
func (state *AppState) AddChoice(choiceLine string, regexPatterns *RegexPatterns) {
	if !state.Prompt.Started {
		return
	}
	
	cleanLine := regexPatterns.StripAnsi(choiceLine)
	
	// Check for any numbered choice (1., 2., 3.)
	if matches := regexPatterns.ChoiceAny.FindStringSubmatch(cleanLine); len(matches) > 2 {
		num := matches[1]
		choiceText := matches[2]
		// Strip pipe characters and extra whitespace from choice text
		choiceText = strings.Trim(choiceText, "│ \t")
		choiceText = strings.TrimRight(choiceText, "│ \t\r\n\u00A0\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200A\u200B\u202F\u205F\u3000◯○◉●>?─━┌┐└┘├┤┬┴┼╭╮╯╰╠╣╦╩╬⧉")
		choiceText = strings.TrimSpace(choiceText)
		// Reconstruct the choice line with cleaned text
		cleanedChoice := num + ". " + choiceText
		state.Prompt.CollectedChoices[num] = cleanedChoice
	}
}

// DialogInterface defines the interface for showing permission dialogs
type DialogInterface interface {
	AskWithChoices(msg string, choices map[string]string, debugFile *os.File) string
}