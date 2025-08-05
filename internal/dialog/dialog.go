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
