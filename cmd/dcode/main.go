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
	stripColors = flag.Bool("strip-colors", false, "Remove ANSI color codes from output")
	debug       = flag.Bool("debug", false, "Enable debug logging to debug_output.log")
)

func debugf(debugFile *os.File, format string, args ...interface{}) {
	if debugFile != nil {
		fmt.Fprintf(debugFile, format, args...)
	}
}

type PermissionHandler struct {
	ptmx         *os.File
	appState     *types.AppState
	patterns     *types.RegexPatterns
	contextLines []string
	debugFile    *os.File
}

func NewPermissionHandler(ptmx *os.File, debugFile *os.File) *PermissionHandler {
	return &PermissionHandler{
		ptmx:         ptmx,
		appState:     types.NewAppState(),
		patterns:     types.NewRegexPatterns(),
		contextLines: make([]string, 0, 10),
		debugFile:    debugFile,
	}
}

func (p *PermissionHandler) processLine(line string) {
	cleanLine := p.patterns.StripAnsi(line)
	
	// Log interesting lines
	if len(strings.TrimSpace(cleanLine)) > 0 && !strings.HasPrefix(cleanLine, "[") {
		debugf(p.debugFile, "[DEBUG] Line: %q\n", cleanLine)
	}
	if strings.Contains(cleanLine, "permission") || strings.Contains(cleanLine, "approval") || 
	   strings.Contains(cleanLine, "requires") || strings.Contains(cleanLine, "Write(") || 
	   strings.Contains(cleanLine, "rejected") {
		debugf(p.debugFile, "[DEBUG] Potential permission line: %q\n", cleanLine)
	}
	
	// Log permission prompts for debugging
	if strings.Contains(cleanLine, "Do you want") {
		debugf(p.debugFile, "[DEBUG] Permission prompt detected: %q\n", cleanLine)
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
				debugf(p.debugFile, "[DEBUG] Detected permission prompt: %q\n", p.patterns.StripAnsi(line))
				p.appState.StartPromptCollectionWithContext(line, contextIdentifier)
			} else {
				debugf(p.debugFile, "[DEBUG] Permission prompt was BLOCKED by shouldProcessPrompt: %q\n", p.patterns.StripAnsi(line))
			}
		} else {
			debugf(p.debugFile, "[DEBUG] Permission prompt SKIPPED due to same context: %q\n", p.patterns.StripAnsi(line))
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
	if !p.appState.ShouldProcessPrompt(line, p.patterns) {
		debugf(p.debugFile, "[DEBUG] Skipping prompt due to processing rules: %q\n", p.patterns.StripAnsi(line))
		return false
	}
	
	return true
}

func (p *PermissionHandler) processChoice(line, cleanLine string) {
	debugf(p.debugFile, "[DEBUG] Checking line for choices: %q\n", cleanLine)
	
	p.appState.AddChoice(line, p.patterns)
	
	// Check if this is the end of choices
	if strings.TrimSpace(cleanLine) == "" || strings.Contains(cleanLine, "╰") || strings.Contains(cleanLine, "Your choice:") {
		debugf(p.debugFile, "[DEBUG] End of choices detected, making decision\n")
		p.appState.Prompt.Started = false
		
		bestChoice := choice.GetBestChoiceFromState(p.appState, p.patterns, p.debugFile)
		p.handleUserChoice(bestChoice)
	}
}

func (p *PermissionHandler) handleUserChoice(bestChoice string) {
	if *autoApprove {
		p.sendAutoApprove(bestChoice)
	} else {
		p.showDialog(bestChoice)
	}
}

func (p *PermissionHandler) sendAutoApprove(choice string) {
	debugf(p.debugFile, "[DEBUG] Auto-approve mode, will send %s\n", choice)
	go func() {
		time.Sleep(AutoApproveDelayMs * time.Millisecond)
		var debugFile *os.File
		if *debug {
			debugFile, _ = os.OpenFile("debug_output.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		}
		n, err := p.ptmx.WriteString(choice)
		debugf(debugFile, "[DEBUG] Auto-approve WriteString(%q) returned n=%d, err=%v\n", choice, n, err)
		if debugFile != nil {
			debugFile.Close()
		}
		p.ptmx.Sync()
	}()
}

func (p *PermissionHandler) showDialog(bestChoice string) {
	go func() {
		userChoice := dialog.AskWithChoicesContextAndReason(p.appState.Prompt.LastLine, p.appState.Prompt.CollectedChoices, p.contextLines, p.appState.Prompt.TriggerReason, p.appState.Prompt.TriggerLine)
		
		var debugFile *os.File
		if *debug {
			debugFile, _ = os.OpenFile("debug_output.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		}
		debugf(debugFile, "[DEBUG] User selected choice %s\n", userChoice)
		
		if userChoice != "" {
			n, err := p.ptmx.WriteString(userChoice)
			debugf(debugFile, "[DEBUG] User choice WriteString(%q) returned n=%d, err=%v\n", userChoice, n, err)
			
			// Set cooldown in deduplication manager
			p.appState.Deduplicator.SetDialogCooldown("main_dialog")
			
			go func() {
				time.Sleep(DialogResetDelayMs * time.Millisecond)
				p.appState.Prompt.JustShown = false
				p.appState.Deduplicator.ClearCooldown("main_dialog")
				var debugFile2 *os.File
				if *debug {
					debugFile2, _ = os.OpenFile("debug_output.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
				}
				debugf(debugFile2, "[DEBUG] Dialog cooldown reset\n")
				if debugFile2 != nil {
					debugFile2.Close()
				}
			}()
			
			p.ptmx.Sync()
		} else {
			debugf(debugFile, "[DEBUG] No choice to send (dialog not shown yet)\n")
		}
		
		if debugFile != nil {
			debugFile.Close()
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
		} else if arg == "-strip-colors" || arg == "--strip-colors" {
			*stripColors = true
		} else if arg == "-debug" || arg == "--debug" {
			*debug = true
		} else {
			args = append(args, arg)
		}
	}
	
	// Check if stdin is a pipe/file vs interactive terminal
	stat, _ := os.Stdin.Stat()
	isPipe := (stat.Mode() & os.ModeCharDevice) == 0
	
	// Debug logging - only create file if debug flag is enabled
	var debugFile *os.File
	if *debug {
		debugFile, _ = os.OpenFile("debug_output.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		debugFile = nil
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
			debugf(debugFile, "[DEBUG] Set initial terminal size: %dx%d\n", size.Cols, size.Rows)
		}
		
		// Handle terminal resize
		sigwinch := make(chan os.Signal, 1)
		signal.Notify(sigwinch, syscall.SIGWINCH)
		go func() {
			for range sigwinch {
				if size, err := pty.GetsizeFull(os.Stdin); err == nil {
					pty.Setsize(ptmx, size)
					var debugFile2 *os.File
					if *debug {
						debugFile2, _ = os.OpenFile("debug_output.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
					}
					debugf(debugFile2, "[DEBUG] Terminal resized: %dx%d\n", size.Cols, size.Rows)
					if debugFile2 != nil {
						debugFile2.Close()
					}
				}
			}
		}()
	}
	debugf(debugFile, "[DEBUG] Main: isPipe=%v, stat.Mode()=%v\n", isPipe, stat.Mode())
	if debugFile != nil {
		debugFile.Close()
	}
	
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
			var debugFile *os.File
			if *debug {
				debugFile, _ = os.OpenFile("debug_output.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			}
			defer func() {
				if debugFile != nil {
					debugFile.Close()
				}
			}()
			
			debugf(debugFile, "[DEBUG] Starting piped input processing\n")
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := scanner.Text()
				debugf(debugFile, "[DEBUG] Processing line: %q\n", line)
				
				// Send the text character by character
				for _, char := range line {
					ptmx.WriteString(string(char))
					time.Sleep(CharDelayMs * time.Millisecond)
				}
				debugf(debugFile, "[DEBUG] Sent text, now sending Enter\n")
				// Then send Enter key - try different approaches
				time.Sleep(LineProcessDelayMs * time.Millisecond)
				ptmx.WriteString("\n")
				ptmx.Sync()
				debugf(debugFile, "[DEBUG] Sent Enter, done with line\n")
				time.Sleep(FinalDelayMs * time.Millisecond)
			}
			debugf(debugFile, "[DEBUG] Finished piped input processing\n")
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
	
	// Open debug file
	var debugFile2 *os.File
	if *debug {
		debugFile2, err = os.OpenFile("debug_output.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			debugFile2 = os.Stderr
		}
	} else {
		debugFile2 = nil
	}
	defer func() {
		if debugFile2 != nil {
			debugFile2.Close()
		}
	}()

	// Create display writer
	var displayWriter io.Writer
	if *stripColors {
		displayWriter = &dialog.ColorStripWriter{Writer: os.Stdout}
	} else {
		displayWriter = os.Stdout
	}
	
	// Create permission handler
	permHandler := NewPermissionHandler(ptmx, debugFile2)
	
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
			debugf(debugFile2, "[ERROR] Reading from PTY: %v\n", err)
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