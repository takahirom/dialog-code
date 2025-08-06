package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/takahirom/dialog-code/internal/choice"
	"github.com/takahirom/dialog-code/internal/dialog"
	"github.com/takahirom/dialog-code/internal/types"
)

// Constants for configuration
const (
	PTYBufferSize     = 1024 // Buffer size for PTY reading
	ContextBufferSize = 20   // Buffer size for context lines
)

// PermissionCallback defines the callback for permission requests
type PermissionCallback func(message string, buttons []string, defaultButton string) string

// App represents the main application
type App struct {
	ptmx               *os.File
	handler            *PermissionHandler
	displayWriter      io.Writer
	permissionCallback PermissionCallback
}

// NewApp creates a new App instance
func NewApp(ptmx *os.File, displayWriter io.Writer) *App {
	app := &App{
		ptmx:          ptmx,
		displayWriter: displayWriter,
	}
	app.handler = NewPermissionHandler(ptmx, app.requestPermission)
	return app
}

// SetPermissionCallback sets the callback for permission requests
func (a *App) SetPermissionCallback(callback PermissionCallback) {
	a.permissionCallback = callback
	// Update handler with new callback
	a.handler.permissionCallback = callback
}

// requestPermission is the internal method that calls the external callback
func (a *App) requestPermission(message string, buttons []string, defaultButton string) string {
	if a.permissionCallback != nil {
		return a.permissionCallback(message, buttons, defaultButton)
	}
	// Fallback behavior if no callback is set
	return "1" // Default to first button
}

// NewAppWithDialog creates a new App instance with custom dialog
// Deprecated: Use NewApp with SetPermissionCallback instead
func NewAppWithDialog(ptmx *os.File, displayWriter io.Writer, dialogInterface DialogInterface) *App {
	// Wrap dialog interface in callback
	callback := func(message string, buttons []string, defaultButton string) string {
		return dialogInterface.Show(message, buttons, defaultButton)
	}

	app := &App{
		ptmx:          ptmx,
		handler:       NewPermissionHandler(ptmx, callback),
		displayWriter: displayWriter,
	}
	return app
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
	mu       sync.RWMutex
	FakeTime time.Time
}

func (t *FakeTimeProvider) Now() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.FakeTime
}

// SetTime safely updates the fake time under a write lock
func (t *FakeTimeProvider) SetTime(tm time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.FakeTime = tm
}

// FakeDialog implements DialogInterface for testing
type FakeDialog struct {
	mu              sync.RWMutex
	CapturedMessage string
	CapturedButtons []string
	CapturedDefault string
	ReturnChoice    string
	TimeProvider    TimeProvider
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
	ptmx               *os.File
	appState           *types.AppState
	patterns           *types.RegexPatterns
	contextLines       []string
	waitingForInput    bool
	timeProvider       TimeProvider
	permissionCallback PermissionCallback
}

// buildDialogMessage constructs the dialog message from the permission prompt data using new clean format
func (p *PermissionHandler) buildDialogMessage(promptLine string, contextLines []string, triggerReason string) string {
	// Create timestamp for clean format
	var timestamp string
	if p.timeProvider != nil {
		timestamp = fmt.Sprintf("%d", p.timeProvider.Now().UnixNano())
	} else {
		timestamp = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Create regex patterns if not available
	regexPatterns := p.patterns
	if regexPatterns == nil {
		regexPatterns = &types.RegexPatterns{}
	}

	// Use the TriggerLine from appState if available, otherwise use promptLine
	triggerLine := promptLine
	if p.appState.Prompt.TriggerLine != "" {
		triggerLine = p.appState.Prompt.TriggerLine
	}

	// Use the new clean dialog message format
	return choice.GetCleanDialogMessage(promptLine, contextLines, triggerReason, triggerLine, timestamp, regexPatterns)
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

func NewPermissionHandler(ptmx *os.File, permissionCallback PermissionCallback) *PermissionHandler {
	return &PermissionHandler{
		ptmx:               ptmx,
		appState:           types.NewAppState(),
		patterns:           types.NewRegexPatterns(),
		contextLines:       make([]string, 0, 10),
		timeProvider:       &RealTimeProvider{},
		permissionCallback: permissionCallback,
	}
}

// NewPermissionHandlerWithDialog creates a handler that uses dialog interface via callback wrapper
// Deprecated: Use NewPermissionHandler with callback instead
func NewPermissionHandlerWithDialog(ptmx *os.File, dialogInterface DialogInterface) *PermissionHandler {
	// Wrap the dialog interface in a callback
	callback := func(message string, buttons []string, defaultButton string) string {
		return dialogInterface.Show(message, buttons, defaultButton)
	}

	return &PermissionHandler{
		ptmx:               ptmx,
		appState:           types.NewAppState(),
		patterns:           types.NewRegexPatterns(),
		contextLines:       make([]string, 0, 10),
		timeProvider:       &RealTimeProvider{},
		permissionCallback: callback,
	}
}

// NewPermissionHandlerWithDialogAndTimeProvider creates a handler with dialog interface and time provider
// Deprecated: Use NewPermissionHandler with callback instead
func NewPermissionHandlerWithDialogAndTimeProvider(ptmx *os.File, dialogInterface DialogInterface, timeProvider TimeProvider) *PermissionHandler {
	// Wrap the dialog interface in a callback
	callback := func(message string, buttons []string, defaultButton string) string {
		return dialogInterface.Show(message, buttons, defaultButton)
	}

	return &PermissionHandler{
		ptmx:               ptmx,
		appState:           types.NewAppState(),
		patterns:           types.NewRegexPatterns(),
		contextLines:       make([]string, 0, 10),
		timeProvider:       timeProvider,
		permissionCallback: callback,
	}
}

func (p *PermissionHandler) processLine(line string) {
	cleanLine := p.patterns.StripAnsi(line)

	// Collect context lines (always collect unless it's debug)
	if len(strings.TrimSpace(cleanLine)) > 0 && !strings.HasPrefix(cleanLine, "[DEBUG]") {
		p.contextLines = append(p.contextLines, cleanLine)
		if len(p.contextLines) > ContextBufferSize { // Increase buffer for dialog boxes
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
				p.appState.StartPromptCollectionWithContext(line, contextIdentifier, p.contextLines)
			}
		}
		return
	}

	// Process choices if in prompt
	if p.appState.Prompt.Started {
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
	return p.appState.ShouldProcessPrompt(line, p.patterns)
}

func (p *PermissionHandler) processChoice(line, cleanLine string) {
	p.appState.AddChoice(line, p.patterns)

	// Check if this is the end of choices
	if strings.Contains(cleanLine, "╰") {
		p.appState.Prompt.Started = false

		// Add a longer delay to ensure the prompt is fully rendered and processed
		time.Sleep(ChoiceProcessingDelayMs * time.Millisecond)

		bestChoice := choice.GetBestChoiceFromState(p.appState, p.patterns)
		p.handleUserChoice(bestChoice)
	}
}

func (p *PermissionHandler) handleUserChoice(bestChoice string) {
	if *autoApprove {
		errCh := p.sendAutoApprove(bestChoice)
		go func() {
			if err := <-errCh; err != nil {
				// Log error but continue operation
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}()
	} else if *autoReject {
		p.sendAutoReject()
	} else if *autoRejectWait > 0 {
		p.sendAutoRejectWithWait(bestChoice)
	} else {
		p.showDialog(bestChoice)
	}
}

func (p *PermissionHandler) sendAutoApprove(choice string) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		time.Sleep(AutoApproveDelayMs * time.Millisecond)
		if err := p.writeToTerminal(choice); err != nil {
			errCh <- fmt.Errorf("auto-approve failed: %w", err)
			return
		}
	}()
	return errCh
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

	go func() {
		time.Sleep(AutoRejectProcessDelayMs * time.Millisecond)
		// Send the max choice number without newline (like dialog mode)
		if err := p.writeToTerminal(maxChoice); err != nil {
			return
		}

		// Wait for the choice to be processed
		time.Sleep(AutoRejectChoiceDelayMs * time.Millisecond)

		// Now send the rejection message
		rejectMsg := p.buildAutoRejectMessage()
		if err := p.writeToTerminal(rejectMsg); err != nil {
			return
		}

		// Send carriage return separately
		time.Sleep(AutoRejectCRDelayMs * time.Millisecond)
		if err := p.writeToTerminal("\r"); err != nil {
			// Carriage return failed, continue silently
		}
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
			baseMessage := p.buildDialogMessage(p.appState.Prompt.LastLine, p.appState.Prompt.Context, p.appState.Prompt.TriggerReason)
			countdownMsg := fmt.Sprintf("This will auto-reject in %d seconds...\n\n%s", *autoRejectWait, baseMessage)
			buttons := p.extractButtons()
			defaultButton := ""
			if len(buttons) > 0 {
				defaultButton = buttons[0]
			}

			var userChoice string
			if p.permissionCallback != nil {
				userChoice = p.permissionCallback(countdownMsg, buttons, defaultButton)
			} else {
				// No permission callback set, cannot show dialog
				userChoice = ""
			}

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
			if err := p.writeToTerminal(userChoice); err != nil {
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

// Dialog parsing constants
const (
	DialogQuestionPattern = "Do you want to proceed"
	DialogChoicePattern   = "❯"
	DialogCommandPattern  = "command"
)

// isValidCommandLine checks if a line contains valid command information
func isValidCommandLine(line string) bool {
	cleanLine := strings.Trim(line, "│ \t")
	cleanLine = strings.TrimSpace(cleanLine)
	
	if cleanLine == "" {
		return false
	}
	
	// Skip dialog UI elements and decorations
	excludePatterns := []string{
		DialogQuestionPattern,
		DialogChoicePattern,
		DialogCommandPattern,
	}
	
	// Check for patterns that should be filtered anywhere in the line
	for _, pattern := range excludePatterns {
		if strings.Contains(cleanLine, pattern) {
			return false
		}
	}
	
	// Check for patterns that should be filtered at line start
	if strings.HasPrefix(cleanLine, ">") || strings.HasPrefix(cleanLine, ".") {
		return false
	}
	
	return true
}

// buildAutoRejectMessage creates auto-reject message with command details
func (p *PermissionHandler) buildAutoRejectMessage() string {
	// Get command details from dialog context
	if len(p.appState.Prompt.Context) > 0 {
		var builder strings.Builder
		
		for _, line := range p.appState.Prompt.Context {
			// Look for command information (skip dialog box decorations)
			if strings.Contains(line, "│") && isValidCommandLine(line) {
				cleanLine := strings.Trim(line, "│ \t")
				cleanLine = strings.TrimSpace(cleanLine)
				
				if builder.Len() > 0 {
					builder.WriteString("\n")
				}
				builder.WriteString(cleanLine)
			}
		}
		
		if builder.Len() > 0 {
			return fmt.Sprintf("Rejected command:\n%s\n\n%s", builder.String(), AutoRejectBaseMessage)
		}
	}
	
	return AutoRejectBaseMessage
}

func (p *PermissionHandler) writeAutoRejectChoice(maxChoice string) {
	// Send the max choice number without newline (like dialog mode)
	if err := p.writeToTerminal(maxChoice); err != nil {
		return
	}

	// Wait for the choice to be processed
	time.Sleep(AutoRejectChoiceDelayMs * time.Millisecond)

	// Now send the rejection message
	rejectMsg := p.buildAutoRejectMessage()
	if err := p.writeToTerminal(rejectMsg); err != nil {
		return
	}

	// Send carriage return separately
	time.Sleep(AutoRejectCRDelayMs * time.Millisecond)
	if err := p.writeToTerminal("\r"); err != nil {
		// Carriage return failed, continue silently
	}
}

func (p *PermissionHandler) writeToTerminal(text string) error {
	_, err := p.ptmx.WriteString(text)
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
	}()
}

func (p *PermissionHandler) showDialog(bestChoice string) {
	go func() {
		message := p.buildDialogMessage(p.appState.Prompt.LastLine, p.appState.Prompt.Context, p.appState.Prompt.TriggerReason)
		buttons := p.extractButtons()
		defaultButton := ""
		if len(buttons) > 0 {
			defaultButton = buttons[0]
		}

		var userChoice string
		if p.permissionCallback != nil {
			userChoice = p.permissionCallback(message, buttons, defaultButton)
		} else {
			// No permission callback set, cannot show dialog
			userChoice = ""
		}

		if userChoice != "" {
			if err := p.writeToTerminal(userChoice); err != nil {
				return
			}

			p.handleDialogCooldown()
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

// isUserInputPattern checks if the output contains patterns indicating user input
func isUserInputPattern(output string) bool {
	return strings.Contains(output, "1") ||
		strings.Contains(output, "2") ||
		strings.Contains(output, "3") ||
		strings.Contains(output, "\n") ||
		strings.Contains(output, "\r\n")
}

// Run starts the application
func (a *App) Run() error {
	// Initialize dialog globals
	dialog.SetPtmxGlobal(a.ptmx)
	dialog.InitGlobals()

	// Single read loop that handles both output and permission detection
	buffer := make([]byte, PTYBufferSize)
	var lineBuffer []byte

	// Create a pipe to process data
	pipeReader, pipeWriter := io.Pipe()
	defer pipeWriter.Close()

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
			return fmt.Errorf("PTY read error: %w", err)
		}

		// Write to pipe for output
		pipeWriter.Write(buffer[:n])

		// Check for user input during wait period by monitoring PTY output changes
		if a.handler.waitingForInput && n > 0 {
			// Look for patterns that indicate actual user choice input
			outputStr := string(buffer[:n])

			// Detect specific user input patterns (choice numbers, enter key)
			if isUserInputPattern(outputStr) {
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

	return nil
}
