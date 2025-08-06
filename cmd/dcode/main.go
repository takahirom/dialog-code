package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"

	"github.com/takahirom/dialog-code/internal/debug"
	"github.com/takahirom/dialog-code/internal/dialog"
)

const (
	// Timing constants for cooldowns and delays
	DialogCooldownMs        = 500
	AutoApproveDelayMs      = 100
	DialogResetDelayMs      = 3000
	CharDelayMs             = 10
	LineProcessDelayMs      = 100
	FinalDelayMs            = 500
	PromptDuplicationSec    = 5
	ChoiceProcessingDelayMs = 300

	// Auto-reject timing constants
	AutoRejectChoiceDelayMs  = 500
	AutoRejectCRDelayMs      = 400
	AutoRejectProcessDelayMs = 500

	// Auto-reject message
	AutoRejectMessage = "The command was automatically rejected. If using Task tools, please restart them. Otherwise, try a different command."
)

var (
	autoApprove    = flag.Bool("auto-approve", false, "Automatically approve all prompts without showing dialogs")
	autoReject     = flag.Bool("auto-reject", false, "Automatically reject unauthorized commands without showing dialogs")
	autoRejectWait = flag.Int("auto-reject-wait", 0, "Auto-reject with N seconds wait for user intervention (0 = disabled)")
	stripColors    = flag.Bool("strip-colors", false, "Remove ANSI color codes from output")
	debugFlag      = flag.Bool("debug", false, "Enable debug logging to debug_output.log")
)

func main() {
	// Parse only known flags, pass everything else to claude
	var args []string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "-auto-approve" || arg == "--auto-approve" {
			*autoApprove = true
		} else if arg == "-auto-reject" || arg == "--auto-reject" {
			*autoReject = true
		} else if strings.HasPrefix(arg, "-auto-reject-wait=") || strings.HasPrefix(arg, "--auto-reject-wait=") {
			// Parse --auto-reject-wait=N format
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				if waitTime, err := strconv.Atoi(parts[1]); err == nil && waitTime >= 0 {
					*autoRejectWait = waitTime
				} else {
					fmt.Fprintf(os.Stderr, "Invalid auto-reject-wait value: %s\n", parts[1])
					os.Exit(1)
				}
			}
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
		}

		// Handle terminal resize
		sigwinch := make(chan os.Signal, 1)
		signal.Notify(sigwinch, syscall.SIGWINCH)
		go func() {
			for range sigwinch {
				if size, err := pty.GetsizeFull(os.Stdin); err == nil {
					pty.Setsize(ptmx, size)
				}
			}
		}()
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
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := scanner.Text()

				// Send the text character by character
				for _, char := range line {
					ptmx.WriteString(string(char))
					time.Sleep(CharDelayMs * time.Millisecond)
				}
				// Then send Enter key - try different approaches
				time.Sleep(LineProcessDelayMs * time.Millisecond)
				ptmx.WriteString("\n")
				ptmx.Sync()
				time.Sleep(FinalDelayMs * time.Millisecond)
			}
		}()
	} else {
		// For interactive input, use direct copy
		go func() {
			_, _ = io.Copy(ptmx, os.Stdin)
		}()
	}

	// Create display writer
	var displayWriter io.Writer
	if *stripColors {
		displayWriter = &dialog.ColorStripWriter{Writer: os.Stdout}
	} else {
		displayWriter = os.Stdout
	}

	// Create and run the app
	app := NewApp(ptmx, displayWriter)

	// Initialize dialog at application level (outside of app core)
	simpleDialog := dialog.NewSimpleOSDialog()

	// Set up permission callback to use the simple dialog
	app.SetPermissionCallback(func(message string, buttons []string, defaultButton string) string {
		return simpleDialog.Show(message, buttons, defaultButton)
	})

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "App error: %v\n", err)
		os.Exit(1)
	}
}
