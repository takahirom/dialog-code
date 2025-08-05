package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

// AppRobot provides a fluent interface for testing app functionality
type AppRobot struct {
	t            *testing.T
	app          *App
	dialog       *FakeDialog
	timeProvider *FakeTimeProvider
	tmpFile      *os.File
}

// NewAppRobot creates a new test robot for app testing
func NewAppRobot(t *testing.T) *AppRobot {
	tmpFile, err := os.CreateTemp("", "fake_ptmx")
	if err != nil {
		t.Fatalf("Failed to create fake PTY: %v", err)
	}

	fakeDialog := &FakeDialog{
		ReturnChoice: "1",
	}

	// Use fixed time for consistent testing
	fakeTimeProvider := &FakeTimeProvider{
		FakeTime: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	app := NewAppWithDialogAndTimeProvider(tmpFile, os.Stdout, fakeDialog, fakeTimeProvider)

	robot := &AppRobot{
		t:            t,
		app:          app,
		dialog:       fakeDialog,
		timeProvider: fakeTimeProvider,
		tmpFile:      tmpFile,
	}

	// Setup cleanup
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
		tmpFile.Close()
	})

	return robot
}

// ReceiveClaudeText simulates receiving Claude output lines
func (r *AppRobot) ReceiveClaudeText(lines ...string) *AppRobot {
	for _, line := range lines {
		r.app.handler.processLine(line)
	}
	// Give goroutines time to complete
	time.Sleep(200 * time.Millisecond)
	return r
}

// AssertDialogCaptured verifies that dialog was triggered
func (r *AppRobot) AssertDialogCaptured() *AppRobot {
	if r.dialog.GetCapturedMessage() == "" {
		r.t.Error("Expected dialog to be captured, but CapturedMessage is empty")
	}
	return r
}

// AssertDialogTextContains verifies dialog message contains expected text
func (r *AppRobot) AssertDialogTextContains(expectedText string) *AppRobot {
	capturedMessage := r.dialog.GetCapturedMessage()
	if !strings.Contains(capturedMessage, expectedText) {
		r.t.Errorf("Expected dialog to contain '%s', got: %q", expectedText, capturedMessage)
	}
	return r
}

// AssertDialogText verifies dialog message matches exactly
func (r *AppRobot) AssertDialogText(expectedText string) *AppRobot {
	capturedMessage := r.dialog.GetCapturedMessage()
	if capturedMessage != expectedText {
		r.t.Errorf("Expected dialog text to be exactly '%s', got: %q", expectedText, capturedMessage)
	}
	return r
}

// AssertDialogTextExact verifies a custom message matches exactly (for timestamp-stripped comparisons)
func (r *AppRobot) AssertDialogTextExact(actualMessage, expectedText string) *AppRobot {
	if actualMessage != expectedText {
		r.t.Errorf("Expected dialog text to be exactly '%s', got: %q", expectedText, actualMessage)
	}
	return r
}

// AssertExactFormatSnapshotTest verifies dialog message matches exactly, similar to snapshot testing
func (r *AppRobot) AssertExactFormatSnapshotTest(expectedText string) *AppRobot {
	capturedMessage := r.dialog.GetCapturedMessage()
	if capturedMessage != expectedText {
		r.t.Errorf("Snapshot test failed - Expected dialog text to be exactly:\n%s\n\nGot:\n%q", expectedText, capturedMessage)
	}
	return r
}

// AssertButton verifies a specific button was captured
func (r *AppRobot) AssertButton(index int, expectedText string) *AppRobot {
	capturedButtons := r.dialog.GetCapturedButtons()
	if index >= len(capturedButtons) {
		r.t.Errorf("Expected button at index %d, but only %d buttons captured", index, len(capturedButtons))
	} else if capturedButtons[index] != expectedText {
		r.t.Errorf("Button %d: expected '%s', got '%s'", index, expectedText, capturedButtons[index])
	}
	return r
}

// AssertButtonCount verifies the number of buttons captured
func (r *AppRobot) AssertButtonCount(expected int) *AppRobot {
	actual := len(r.dialog.GetCapturedButtons())
	if actual != expected {
		r.t.Errorf("Expected %d buttons, got %d", expected, actual)
	}
	return r
}

// AssertMessageContains verifies the dialog message contains expected text
func (r *AppRobot) AssertMessageContains(expectedText string) *AppRobot {
	capturedMessage := r.dialog.GetCapturedMessage()
	if !strings.Contains(capturedMessage, expectedText) {
		r.t.Errorf("Expected dialog message to contain '%s', got: %q", expectedText, capturedMessage)
	}
	return r
}

// AssertParserExtractsToolTypeAndContent tests parser integration with captured context
func (r *AppRobot) AssertParserExtractsToolTypeAndContent(completeDialog string, expectedToolType string, expectedContent string) *AppRobot {
	parsedInfo, err := r.app.handler.ProcessWithParser(completeDialog)
	if err != nil {
		r.t.Errorf("Parser should handle dialog, got error: %v", err)
		return r
	}

	if parsedInfo == nil {
		r.t.Error("Parser should return valid result")
		return r
	}

	if parsedInfo.ToolType != expectedToolType {
		r.t.Errorf("Expected ToolType '%s', got '%s'", expectedToolType, parsedInfo.ToolType)
	}

	if !strings.Contains(parsedInfo.RawContent, expectedContent) {
		r.t.Errorf("Expected parser content to contain '%s', got: %s", expectedContent, parsedInfo.RawContent)
	}

	return r
}

// SetDialogChoice sets the choice that FakeDialog will return
func (r *AppRobot) SetDialogChoice(choice string) *AppRobot {
	r.dialog.mu.Lock()
	r.dialog.ReturnChoice = choice
	r.dialog.mu.Unlock()
	return r
}

// GetCapturedMessage returns the captured dialog message for custom assertions
func (r *AppRobot) GetCapturedMessage() string {
	return r.dialog.GetCapturedMessage()
}

// GetCapturedButtons returns the captured buttons for custom assertions
func (r *AppRobot) GetCapturedButtons() []string {
	return r.dialog.GetCapturedButtons()
}

// LogDebugInfo logs debug information about the current state
func (r *AppRobot) LogDebugInfo() *AppRobot {
	r.t.Logf("Dialog captured: %d buttons", len(r.dialog.GetCapturedButtons()))
	r.t.Logf("Default button: %s", r.dialog.GetCapturedDefault())
	r.t.Logf("Captured message: %q", r.dialog.GetCapturedMessage())
	return r
}

// PrintCapturedMessage prints the captured message for debugging
func (r *AppRobot) PrintCapturedMessage() *AppRobot {
	r.t.Logf("=== CAPTURED MESSAGE START ===")
	r.t.Logf("%s", r.dialog.GetCapturedMessage())
	r.t.Logf("=== CAPTURED MESSAGE END ===")
	return r
}

// SetFakeTime sets the fake time for the time provider
func (r *AppRobot) SetFakeTime(fakeTime time.Time) *AppRobot {
	r.timeProvider.FakeTime = fakeTime
	return r
}

// GetCapturedMessageWithoutTimestamp returns the captured message with timestamp stripped
func (r *AppRobot) GetCapturedMessageWithoutTimestamp() string {
	actualMessage := r.dialog.GetCapturedMessage()
	parts := strings.Split(actualMessage, "|")
	if len(parts) >= 2 {
		// Remove the timestamp (last part) for stable comparison
		return strings.Join(parts[:len(parts)-1], "|")
	}
	return actualMessage
}
