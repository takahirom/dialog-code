package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/takahirom/dialog-code/internal/choice"
	"github.com/takahirom/dialog-code/internal/debug"
	"github.com/takahirom/dialog-code/internal/dialog"
	"github.com/takahirom/dialog-code/internal/parser"
	"github.com/takahirom/dialog-code/internal/types"
)

// App represents the main application
type App struct {
	ptmx         *os.File
	handler      *PermissionHandler
	displayWriter io.Writer
}

// NewApp creates a new App instance
func NewApp(ptmx *os.File, displayWriter io.Writer) *App {
	return &App{
		ptmx:          ptmx,
		handler:       NewPermissionHandler(ptmx),
		displayWriter: displayWriter,
	}
}

// NewAppWithDialog creates a new App instance with custom dialog
func NewAppWithDialog(ptmx *os.File, displayWriter io.Writer, dialogInterface DialogInterface) *App {
	return &App{
		ptmx:          ptmx,
		handler:       NewPermissionHandlerWithDialog(ptmx, dialogInterface),
		displayWriter: displayWriter,
	}
}

// NewAppWithDialogAndTimeProvider creates a new App instance with custom dialog and time provider
func NewAppWithDialogAndTimeProvider(ptmx *os.File, displayWriter io.Writer, dialogInterface DialogInterface, timeProvider TimeProvider) *App {
	return &App{
		ptmx:          ptmx,
		handler:       NewPermissionHandlerWithDialogAndTimeProvider(ptmx, dialogInterface, timeProvider),
		displayWriter: displayWriter,
	}
}

// DialogInterface defines the interface for dialog interactions
type DialogInterface interface {
	Show(message string, buttons []string, defaultButton string) string
}

// TimeProvider defines the interface for time operations
type TimeProvider interface {
	Now() time.Time
}

// RealDialog implements DialogInterface using the actual dialog package
type RealDialog struct{}

func (d *RealDialog) Show(message string, buttons []string, defaultButton string) string {
	return dialog.Show(message, buttons, defaultButton)
}

// RealTimeProvider implements TimeProvider using the actual time package
type RealTimeProvider struct{}

func (t *RealTimeProvider) Now() time.Time {
	return time.Now()
}

// FakeTimeProvider implements TimeProvider for testing
type FakeTimeProvider struct {
	FakeTime time.Time
}

func (t *FakeTimeProvider) Now() time.Time {
	return t.FakeTime
}

// FakeDialog implements DialogInterface for testing
type FakeDialog struct {
	mu              sync.RWMutex
	CapturedMessage string
	CapturedButtons []string
	CapturedDefault string
	ReturnChoice    string
}

func (d *FakeDialog) Show(message string, buttons []string, defaultButton string) string {
	d.mu.Lock()
	d.CapturedMessage = message
	d.CapturedButtons = make([]string, len(buttons))
	copy(d.CapturedButtons, buttons)
	d.CapturedDefault = defaultButton
	returnChoice := d.ReturnChoice
	d.mu.Unlock()
	return returnChoice
}

// GetCapturedMessage returns the captured message thread-safely
func (d *FakeDialog) GetCapturedMessage() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.CapturedMessage
}

// GetCapturedButtons returns the captured buttons thread-safely
func (d *FakeDialog) GetCapturedButtons() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	// Return a copy to prevent concurrent slice access
	buttons := make([]string, len(d.CapturedButtons))
	copy(buttons, d.CapturedButtons)
	return buttons
}

// GetCapturedDefault returns the captured default button thread-safely
func (d *FakeDialog) GetCapturedDefault() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.CapturedDefault
}

type PermissionHandler struct {
	ptmx            *os.File
	appState        *types.AppState
	patterns        *types.RegexPatterns
	contextLines    []string
	waitingForInput bool
	dialog          DialogInterface
	timeProvider    TimeProvider
}

// buildDialogMessage constructs the dialog message from the permission prompt data
func (p *PermissionHandler) buildDialogMessage(promptLine string, contextLines []string, triggerReason string) string {
	var message strings.Builder
	
	// Add context if available  
	if len(contextLines) > 0 {
		message.WriteString("Context:\n")
		for _, line := range contextLines {
			if strings.TrimSpace(line) != "" {
				message.WriteString(line + "\n")
			}
		}
		message.WriteString("\n")
	}
	
	// Add the main prompt
	message.WriteString(promptLine)
	
	// Add trigger reason if available
	if triggerReason != "" {
		message.WriteString("\n\nReason: " + triggerReason)
	}
	
	return message.String()
}

// extractButtons extracts button labels from collected choices
func (p *PermissionHandler) extractButtons() []string {
	var buttons []string
	for i := 1; i <= len(p.appState.Prompt.CollectedChoices); i++ {
		key := fmt.Sprintf("%d", i)
		if choice, exists := p.appState.Prompt.CollectedChoices[key]; exists {
			// Extract button text after the number and period
			parts := strings.SplitN(choice, ". ", 2)
			if len(parts) > 1 {
				buttons = append(buttons, parts[1])
			} else {
				buttons = append(buttons, choice)
			}
		}
	}
	return buttons
}

func NewPermissionHandler(ptmx *os.File) *PermissionHandler {
	return &PermissionHandler{
		ptmx:         ptmx,
		appState:     types.NewAppState(),
		patterns:     types.NewRegexPatterns(),
		contextLines: make([]string, 0, 10),
		dialog:       &RealDialog{},
		timeProvider: &RealTimeProvider{},
	}
}

func NewPermissionHandlerWithDialog(ptmx *os.File, dialogInterface DialogInterface) *PermissionHandler {
	return &PermissionHandler{
		ptmx:         ptmx,
		appState:     types.NewAppState(),
		patterns:     types.NewRegexPatterns(),
		contextLines: make([]string, 0, 10),
		dialog:       dialogInterface,
		timeProvider: &RealTimeProvider{},
	}
}

func NewPermissionHandlerWithDialogAndTimeProvider(ptmx *os.File, dialogInterface DialogInterface, timeProvider TimeProvider) *PermissionHandler {
	return &PermissionHandler{
		ptmx:         ptmx,
		appState:     types.NewAppState(),
		patterns:     types.NewRegexPatterns(),
		contextLines: make([]string, 0, 10),
		dialog:       dialogInterface,
		timeProvider: timeProvider,
	}
}

// ProcessWithParser processes context lines through parser before showing dialog
func (p *PermissionHandler) ProcessWithParser(contextLinesStr string) (*parser.DialogInfo, error) {
	return parser.ParseDialog(contextLinesStr)
}

func (p *PermissionHandler) processLine(line string) {
	cleanLine := p.patterns.StripAnsi(line)

	// Log interesting lines for debugging
	if strings.Contains(cleanLine, "permission") || strings.Contains(cleanLine, "approval") ||
		strings.Contains(cleanLine, "requires") || strings.Contains(cleanLine, "Write(") ||
		strings.Contains(cleanLine, "rejected") {
		debugf("[DEBUG] Potential permission line: %q\n", cleanLine)
	}

	// Log permission prompts for debugging
	if strings.Contains(cleanLine, "Do you want") {
		debugf("[DEBUG] Permission prompt detected: %q\n", cleanLine)
	}

	// Collect context lines
	if !p.appState.Prompt.Started && len(strings.TrimSpace(cleanLine)) > 0 && !strings.HasPrefix(cleanLine, "[DEBUG]") {
		p.contextLines = append(p.contextLines, cleanLine)
		if len(p.contextLines) > 10 {
			p.contextLines = p.contextLines[1:]
		}
	}

	// Skip certain types of lines
	if p.shouldSkipLine(cleanLine) {
		return
	}

	// Check for permission prompt start
	if p.patterns.Permit.MatchString(line) {
		// Create a context-aware identifier for this prompt
		// Include recent context lines to distinguish between different commands
		contextIdentifier := ""
		if len(p.contextLines) > 0 {
			// Use the last few context lines to create a unique identifier
			contextLinesToInclude := 3
			for i := len(p.contextLines) - contextLinesToInclude; i < len(p.contextLines) && i >= 0; i++ {
				contextIdentifier += p.contextLines[i] + "|"
			}
		}
		contextIdentifier += p.patterns.StripAnsi(line)

		// Add timestamp to make each prompt unique
		contextIdentifier += "|" + fmt.Sprintf("%d", p.timeProvider.Now().UnixNano())

		if contextIdentifier != p.appState.Prompt.LastLine {
			if p.shouldProcessPrompt(line) {
				debugf("[DEBUG] Detected permission prompt: %q\n", p.patterns.StripAnsi(line))
				p.appState.StartPromptCollectionWithContext(line, contextIdentifier)
			} else {
				debugf("[DEBUG] Permission prompt was BLOCKED by shouldProcessPrompt: %q\n", p.patterns.StripAnsi(line))
			}
		} else {
			debugf("[DEBUG] Permission prompt SKIPPED due to same context: %q\n", p.patterns.StripAnsi(line))
		}
		return
	}

	// Process choices if in prompt
	if p.appState.Prompt.Started {
		debugf("[DEBUG] Processing choice (prompt started): %q\n", cleanLine)
		p.processChoice(line, cleanLine)
	}
}

func (p *PermissionHandler) shouldSkipLine(cleanLine string) bool {
	return strings.HasPrefix(strings.TrimSpace(cleanLine), "+") ||
		strings.HasPrefix(strings.TrimSpace(cleanLine), "-") ||
		strings.Contains(cleanLine, "⎿") ||
		strings.Contains(cleanLine, "☒") ||
		strings.Contains(cleanLine, "Context:") ||
		len(strings.TrimSpace(cleanLine)) <= 10
}

func (p *PermissionHandler) shouldProcessPrompt(line string) bool {
	if !p.appState.ShouldProcessPrompt(line, p.patterns) {
		debugf("[DEBUG] Skipping prompt due to processing rules: %q\n", p.patterns.StripAnsi(line))
		return false
	}

	return true
}

func (p *PermissionHandler) processChoice(line, cleanLine string) {
	debugf("[DEBUG] Checking line for choices: %q\n", cleanLine)

	p.appState.AddChoice(line, p.patterns)

	// Check if this is the end of choices
	if strings.Contains(cleanLine, "╰") {
		debugf("[DEBUG] End of choices detected (found ╰), making decision\n")
		debugf("[DEBUG] Collected choices: %v\n", p.appState.Prompt.CollectedChoices)
		p.appState.Prompt.Started = false

		// Add a longer delay to ensure the prompt is fully rendered and processed
		time.Sleep(300 * time.Millisecond)

		bestChoice := choice.GetBestChoiceFromState(p.appState, p.patterns)
		debugf("[DEBUG] Best choice: %s, autoReject: %v\n", bestChoice, *autoReject)
		p.handleUserChoice(bestChoice)
	}
}

func (p *PermissionHandler) handleUserChoice(bestChoice string) {
	if *autoApprove {
		p.sendAutoApprove(bestChoice)
	} else if *autoReject {
		p.sendAutoReject()
	} else if *autoRejectWait > 0 {
		p.sendAutoRejectWithWait(bestChoice)
	} else {
		p.showDialog(bestChoice)
	}
}

func (p *PermissionHandler) sendAutoApprove(choice string) {
	debugf("[DEBUG] Auto-approve mode, will send %s\n", choice)
	go func() {
		time.Sleep(AutoApproveDelayMs * time.Millisecond)
		n, err := p.ptmx.WriteString(choice)
		debugf("[DEBUG] Auto-approve WriteString(%q) returned n=%d, err=%v\n", choice, n, err)
		p.ptmx.Sync()
	}()
}

func (p *PermissionHandler) sendAutoReject() {
	// Find the highest numbered choice (typically 2 or 3 for reject)
	maxChoice := "2"
	for num := 3; num >= 2; num-- {
		numStr := fmt.Sprintf("%d", num)
		if _, exists := p.appState.Prompt.CollectedChoices[numStr]; exists {
			maxChoice = numStr
			break
		}
	}

	debugf("[DEBUG] Auto-reject mode, will send %s followed by rejection message\n", maxChoice)
	go func() {
		time.Sleep(500 * time.Millisecond)
		// Send the max choice number without newline (like dialog mode)
		debugf("[DEBUG] About to send choice: %s\n", maxChoice)
		n, err := p.ptmx.WriteString(maxChoice)
		debugf("[DEBUG] Auto-reject WriteString(%q) returned n=%d, err=%v\n", maxChoice, n, err)
		p.ptmx.Sync()

		// Wait for the choice to be processed
		time.Sleep(500 * time.Millisecond)

		// Now send the rejection message
		rejectMsg := "The command was automatically rejected. If using Task tools, please restart them. Otherwise, try a different command. This may occur due to pipes or redirections."
		n, err = p.ptmx.WriteString(rejectMsg)
		debugf("[DEBUG] Auto-reject message WriteString(%q) returned n=%d, err=%v\n", rejectMsg, n, err)
		p.ptmx.Sync()

		// Send carriage return separately
		time.Sleep(400 * time.Millisecond)
		n, err = p.ptmx.WriteString("\r")
		debugf("[DEBUG] Auto-reject CR WriteString(%q) returned n=%d, err=%v\n", "\r", n, err)
		p.ptmx.Sync()
		debugf("[DEBUG] Auto-reject complete\n")
	}()
}

func (p *PermissionHandler) sendAutoRejectWithWait(bestChoice string) {
	maxChoice := findMaxRejectChoice(p.appState.Prompt.CollectedChoices)
	waitDuration := time.Duration(*autoRejectWait) * time.Second

	go func() {
		userChoiceChan := make(chan string, 1)
		done := make(chan bool, 1)

		// Show dialog with countdown in a separate goroutine
		go func() {
			baseMessage := p.buildDialogMessage(p.appState.Prompt.LastLine, p.contextLines, p.appState.Prompt.TriggerReason)
			countdownMsg := fmt.Sprintf("%s\n\nThis will auto-reject in %d seconds...", baseMessage, *autoRejectWait)
			buttons := p.extractButtons()
			defaultButton := ""
			if len(buttons) > 0 {
				defaultButton = buttons[0]
			}
			userChoice := p.dialog.Show(countdownMsg, buttons, defaultButton)
			
			select {
			case userChoiceChan <- userChoice:
			case <-done:
				// Timeout already occurred, don't send
			}
		}()

		// Wait for either user choice or timeout
		select {
		case userChoice := <-userChoiceChan:
			// User made a choice before timeout
			close(done)
			debugf("[DEBUG] User selected choice %s before timeout\n", userChoice)
			if err := p.writeToTerminal(userChoice); err != nil {
				debugf("[ERROR] Failed to write user choice: %v\n", err)
				return
			}
			p.handleDialogCooldown()

		case <-time.After(waitDuration):
			// Timeout expired, proceed with auto-reject
			close(done)
			p.writeAutoRejectChoice(maxChoice)
		}
	}()
}

func (p *PermissionHandler) writeAutoRejectChoice(maxChoice string) {
	// Send the max choice number without newline (like dialog mode)
	if err := p.writeToTerminal(maxChoice); err != nil {
		debugf("[ERROR] Failed to write auto-reject choice: %v\n", err)
		return
	}

	// Wait for the choice to be processed
	time.Sleep(AutoRejectChoiceDelayMs * time.Millisecond)

	// Now send the rejection message
	rejectMsg := "The command was automatically rejected after wait period. If using Task tools, please restart them. Otherwise, try a different command."
	if err := p.writeToTerminal(rejectMsg); err != nil {
		debugf("[ERROR] Failed to write auto-reject message: %v\n", err)
		return
	}

	// Send carriage return separately
	time.Sleep(AutoRejectCRDelayMs * time.Millisecond)
	if err := p.writeToTerminal("\r"); err != nil {
		debugf("[ERROR] Failed to write carriage return: %v\n", err)
	}
}

func (p *PermissionHandler) writeToTerminal(text string) error {
	n, err := p.ptmx.WriteString(text)
	debugf("[DEBUG] WriteString(%q) returned n=%d, err=%v\n", text, n, err)
	if err != nil {
		return fmt.Errorf("failed to write to terminal: %w", err)
	}
	p.ptmx.Sync()
	return nil
}

func (p *PermissionHandler) handleDialogCooldown() {
	// Set cooldown in deduplication manager
	p.appState.Deduplicator.SetDialogCooldown("main_dialog")

	go func() {
		time.Sleep(DialogResetDelayMs * time.Millisecond)
		p.appState.Prompt.JustShown = false
		p.appState.Deduplicator.ClearCooldown("main_dialog")
		debugf("[DEBUG] Dialog cooldown reset\n")
	}()
}

func (p *PermissionHandler) showDialog(bestChoice string) {
	go func() {
		message := p.buildDialogMessage(p.appState.Prompt.LastLine, p.contextLines, p.appState.Prompt.TriggerReason)
		buttons := p.extractButtons()
		defaultButton := ""
		if len(buttons) > 0 {
			defaultButton = buttons[0]
		}
		userChoice := p.dialog.Show(message, buttons, defaultButton)

		debugf("[DEBUG] User selected choice %s\n", userChoice)

		if userChoice != "" {
			if err := p.writeToTerminal(userChoice); err != nil {
				debugf("[ERROR] Failed to write user choice: %v\n", err)
				return
			}

			p.handleDialogCooldown()
		} else {
			debugf("[DEBUG] No choice to send (dialog not shown yet)\n")
		}
	}()
}

// findMaxRejectChoice finds the highest numbered choice for auto-reject (typically 2 or 3)
func findMaxRejectChoice(choices map[string]string) string {
	maxChoice := "2"
	for num := 3; num >= 2; num-- {
		numStr := fmt.Sprintf("%d", num)
		if _, exists := choices[numStr]; exists {
			maxChoice = numStr
			break
		}
	}
	return maxChoice
}

// Run starts the application
func (a *App) Run() error {
	// Initialize dialog globals
	dialog.SetPtmxGlobal(a.ptmx)
	dialog.InitGlobals()

	// Single read loop that handles both output and permission detection
	buffer := make([]byte, 1024)
	var lineBuffer []byte

	// Create a pipe to process data
	pipeReader, pipeWriter := io.Pipe()

	// Start output handling from pipe
	go func() {
		defer pipeReader.Close()
		_, _ = io.Copy(a.displayWriter, pipeReader)
	}()

	for {
		n, err := a.ptmx.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			debugf("[ERROR] Reading from PTY: %v\n", err)
			break
		}

		// Write to pipe for output
		pipeWriter.Write(buffer[:n])

		// Check for user input during wait period by monitoring PTY output changes
		if a.handler.waitingForInput && n > 0 {
			// Look for patterns that indicate actual user choice input
			outputStr := string(buffer[:n])

			// Detect specific user input patterns (choice numbers, enter key)
			if strings.Contains(outputStr, "1") || // Choice 1
				strings.Contains(outputStr, "2") || // Choice 2
				strings.Contains(outputStr, "3") || // Choice 3
				strings.Contains(outputStr, "\n") || // Enter key
				strings.Contains(outputStr, "\r\n") { // Enter key (CRLF)
				debugf("[DEBUG] User choice input detected in PTY output during wait period: %q\n", outputStr)
				a.handler.waitingForInput = false
			}
		}

		// Process data for permission detection
		for i := 0; i < n; i++ {
			if buffer[i] == '\n' {
				line := string(lineBuffer)
				lineBuffer = nil
				a.handler.processLine(line)
			} else {
				lineBuffer = append(lineBuffer, buffer[i])
			}
		}
	}

	pipeWriter.Close()
	return nil
}

func debugf(format string, args ...interface{}) {
	debug.Printf(format, args...)
}