package dialog

import (
	"io"
	"os"
	"regexp"
	"sync"
	"time"
)

// Global variables for backward compatibility
var (
	dialogMutex      sync.Mutex
	dialogShowing    bool
	lastDialogTime   time.Time
	waitingForChoice bool
	choiceResponse   string
	outputMutex      sync.Mutex
	ptmxGlobal       *os.File
	lastPromptLine   string
	collectedChoices map[string]string
	promptStarted    bool
	processedPrompts map[string]time.Time
	dialogJustShown  bool
	dialogCooldown   time.Time
)

// writeAll writes b completely or returns an error.
func writeAll(w io.Writer, b []byte) error {
	for len(b) > 0 {
		n, err := w.Write(b)
		if n > 0 {
			b = b[n:]
		}
		if err != nil {
			return err
		}
		// Prevent infinite loop on zero-write, no-error
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

// ColorStripWriter is a writer that strips ANSI colors before writing
type ColorStripWriter struct {
	Writer io.Writer
	regex  *regexp.Regexp
}

// NewColorStripWriter creates a new ColorStripWriter
func NewColorStripWriter(writer io.Writer) *ColorStripWriter {
	// Pattern for stripping ANSI escape sequences
	ansiEscape := `\x1b\[[0-9;?]*[mKHJhlABCDEFGPST]`
	return &ColorStripWriter{
		Writer: writer,
		regex:  regexp.MustCompile(ansiEscape),
	}
}

func (w *ColorStripWriter) Write(p []byte) (n int, err error) {
	// Work on bytes to avoid string allocs.
	filtered := w.regex.ReplaceAll(p, []byte{})
	if len(filtered) == 0 {
		// All bytes were filtered; we still "consumed" p.
		return len(p), nil
	}
	if err := writeAll(w.Writer, filtered); err != nil {
		// All-or-nothing: if we couldn't flush filtered bytes, report failure.
		return 0, err
	}
	return len(p), nil
}

// ScrollbackClearFilterWriter is a writer that filters out scrollback clear control sequences
type ScrollbackClearFilterWriter struct {
	Writer io.Writer
	regex  *regexp.Regexp
}

// NewScrollbackClearFilterWriter creates a new ScrollbackClearFilterWriter
func NewScrollbackClearFilterWriter(writer io.Writer) *ScrollbackClearFilterWriter {
	// \x1b[3J - Clear entire scrollback buffer (ED - Erase Display with parameter 3)
	scrollbackClearPattern := `\x1b\[3J`
	return &ScrollbackClearFilterWriter{
		Writer: writer,
		regex:  regexp.MustCompile(scrollbackClearPattern),
	}
}

func (w *ScrollbackClearFilterWriter) Write(p []byte) (n int, err error) {
	// Use regex filtering like ColorStripWriter for consistency and simplicity
	// Filters complete ESC[3J sequences in normal usage
	filtered := w.regex.ReplaceAll(p, []byte{})
	if len(filtered) == 0 {
		// All bytes were filtered; we still "consumed" p.
		return len(p), nil
	}
	if err := writeAll(w.Writer, filtered); err != nil {
		// All-or-nothing: if we couldn't flush filtered bytes, report failure.
		return 0, err
	}
	return len(p), nil
}


// TimeProvider interface for testable time  
type TimeProvider interface {
	Now() time.Time
}

// RealTimeProvider provides real time
type RealTimeProvider struct{}

func (r *RealTimeProvider) Now() time.Time {
	return time.Now()
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

// Show displays a simple dialog with message, buttons and default button
func Show(message string, buttons []string, defaultButton string) string {
	// Use SimpleOSDialog instead of the old complex system
	simpleDialog := NewSimpleOSDialog()
	return simpleDialog.Show(message, buttons, defaultButton)
}
