package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"

	"github.com/takahirom/dialog-code/internal/choice"
	"github.com/takahirom/dialog-code/internal/debug"
	"github.com/takahirom/dialog-code/internal/dialog"
	"github.com/takahirom/dialog-code/internal/types"
)

const (
	// Timing constants for cooldowns and delays
	DialogCooldownMs     = 500
	AutoApproveDelayMs   = 100
	DialogResetDelayMs   = 3000
	CharDelayMs          = 10
	LineProcessDelayMs   = 100
	FinalDelayMs         = 500
	PromptDuplicationSec = 5
)

var (
	autoApprove = flag.Bool("auto-approve", false, "Automatically approve all prompts without showing dialogs")
	autoReject  = flag.Bool("auto-reject", false, "Automatically reject unauthorized commands without showing dialogs")
	stripColors = flag.Bool("strip-colors", false, "Remove ANSI color codes from output")
	debugFlag   = flag.Bool("debug", false, "Enable debug logging to debug_output.log")
)

func debugf(format string, args ...interface{}) {
	debug.Printf(format, args...)
}

type PermissionHandler struct {
	ptmx         *os.File
	appState     *types.AppState
	patterns     *types.RegexPatterns
	contextLines []string
}

func NewPermissionHandler(ptmx *os.File) *PermissionHandler {
	return &PermissionHandler{
		ptmx:         ptmx,
		appState:     types.NewAppState(),
		patterns:     types.NewRegexPatterns(),
		contextLines: make([]string, 0, 10),
	}
}

func (p *PermissionHandler) processLine(line string) {
	cleanLine := p.patterns.StripAnsi(line)

	// Log interesting lines
	// Removed Line: logging to reduce log noise
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
		contextIdentifier += "|" + fmt.Sprintf("%d", time.Now().UnixNano())

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
		rejectMsg := "The command was automatically rejected. Please try a different command. This may occur due to pipes or redirections."
		n, err = p.ptmx.WriteString(rejectMsg)
		debugf("[DEBUG] Auto-reject message WriteString(%q) returned n=%d, err=%v\n", rejectMsg, n, err)
		p.ptmx.Sync()

		// Send carriage return separately
		time.Sleep(300 * time.Millisecond)
		n, err = p.ptmx.WriteString("\r")
		debugf("[DEBUG] Auto-reject CR WriteString(%q) returned n=%d, err=%v\n", "\r", n, err)
		p.ptmx.Sync()
		debugf("[DEBUG] Auto-reject complete\n")
	}()
}

func (p *PermissionHandler) showDialog(bestChoice string) {
	go func() {
		userChoice := dialog.AskWithChoicesContextAndReason(p.appState.Prompt.LastLine, p.appState.Prompt.CollectedChoices, p.contextLines, p.appState.Prompt.TriggerReason, p.appState.Prompt.TriggerLine)

		debugf("[DEBUG] User selected choice %s\n", userChoice)

		if userChoice != "" {
			n, err := p.ptmx.WriteString(userChoice)
			debugf("[DEBUG] User choice WriteString(%q) returned n=%d, err=%v\n", userChoice, n, err)

			// Set cooldown in deduplication manager
			p.appState.Deduplicator.SetDialogCooldown("main_dialog")

			go func() {
				time.Sleep(DialogResetDelayMs * time.Millisecond)
				p.appState.Prompt.JustShown = false
				p.appState.Deduplicator.ClearCooldown("main_dialog")
				debugf("[DEBUG] Dialog cooldown reset\n")
			}()

			p.ptmx.Sync()
		} else {
			debugf("[DEBUG] No choice to send (dialog not shown yet)\n")
		}
	}()
}

func main() {
	// Parse only known flags, pass everything else to claude
	var args []string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "-auto-approve" || arg == "--auto-approve" {
			*autoApprove = true
		} else if arg == "-auto-reject" || arg == "--auto-reject" {
			*autoReject = true
		} else if arg == "-strip-colors" || arg == "--strip-colors" {
			*stripColors = true
		} else if arg == "-debug" || arg == "--debug" {
			*debugFlag = true
		} else {
			args = append(args, arg)
		}
	}

	// Check if stdin is a pipe/file vs interactive terminal
	stat, _ := os.Stdin.Stat()
	isPipe := (stat.Mode() & os.ModeCharDevice) == 0

	// Enable debug logging if debug flag is set
	if *debugFlag {
		debug.Enable()
	}

	cmd := exec.Command("claude", args...)

	// Allocate PTY for Claude
	ptmx, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start PTY: %v\n", err)
		os.Exit(1)
	}
	defer ptmx.Close()

	// Set initial terminal size and handle resize
	if !isPipe {
		if size, err := pty.GetsizeFull(os.Stdin); err == nil {
			pty.Setsize(ptmx, size)
			debugf("[DEBUG] Set initial terminal size: %dx%d\n", size.Cols, size.Rows)
		}

		// Handle terminal resize
		sigwinch := make(chan os.Signal, 1)
		signal.Notify(sigwinch, syscall.SIGWINCH)
		go func() {
			for range sigwinch {
				if size, err := pty.GetsizeFull(os.Stdin); err == nil {
					pty.Setsize(ptmx, size)
					debugf("[DEBUG] Terminal resized: %dx%d\n", size.Cols, size.Rows)
				}
			}
		}()
	}
	debugf("[DEBUG] Main: isPipe=%v, stat.Mode()=%v\n", isPipe, stat.Mode())

	var oldState *term.State
	if !isPipe {
		// Set terminal to raw mode only for interactive input
		oldState, _ = term.MakeRaw(int(os.Stdin.Fd()))
	}

	// Restore terminal state only if it was set
	defer func() {
		if oldState != nil {
			term.Restore(int(os.Stdin.Fd()), oldState)
		}
	}()

	// Forward stdin to Claude
	if isPipe {
		// For piped input, read line by line and send with proper termination
		go func() {
			debugf("[DEBUG] Starting piped input processing\n")
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := scanner.Text()
				debugf("[DEBUG] Processing line: %q\n", line)

				// Send the text character by character
				for _, char := range line {
					ptmx.WriteString(string(char))
					time.Sleep(CharDelayMs * time.Millisecond)
				}
				debugf("[DEBUG] Sent text, now sending Enter\n")
				// Then send Enter key - try different approaches
				time.Sleep(LineProcessDelayMs * time.Millisecond)
				ptmx.WriteString("\n")
				ptmx.Sync()
				debugf("[DEBUG] Sent Enter, done with line\n")
				time.Sleep(FinalDelayMs * time.Millisecond)
			}
			debugf("[DEBUG] Finished piped input processing\n")
		}()
	} else {
		// For interactive input, use direct copy
		go func() {
			_, _ = io.Copy(ptmx, os.Stdin)
		}()
	}

	// Initialize dialog globals
	dialog.SetPtmxGlobal(ptmx)
	dialog.InitGlobals()

	// Create display writer
	var displayWriter io.Writer
	if *stripColors {
		displayWriter = &dialog.ColorStripWriter{Writer: os.Stdout}
	} else {
		displayWriter = os.Stdout
	}

	// Create permission handler
	permHandler := NewPermissionHandler(ptmx)

	// Use TeeReader to handle both output and permission detection in single read
	var lineBuffer []byte

	// Create a pipe to process data
	pipeReader, pipeWriter := io.Pipe()

	// Start output handling from pipe
	go func() {
		defer pipeReader.Close()
		_, _ = io.Copy(displayWriter, pipeReader)
	}()

	// Single read loop that handles both output and permission detection
	buffer := make([]byte, 1024)
	for {
		n, err := ptmx.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			debugf("[ERROR] Reading from PTY: %v\n", err)
			break
		}

		// Write to pipe for output
		pipeWriter.Write(buffer[:n])

		// Process data for permission detection
		for i := 0; i < n; i++ {
			if buffer[i] == '\n' {
				line := string(lineBuffer)
				lineBuffer = nil
				permHandler.processLine(line)
			} else {
				lineBuffer = append(lineBuffer, buffer[i])
			}
		}
	}

	pipeWriter.Close()
}
