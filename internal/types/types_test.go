package types

import (
	"testing"
)

func TestRegexPatterns(t *testing.T) {
	patterns := NewRegexPatterns()
	
	t.Run("Permit pattern", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected bool
		}{
			{"Do you want to proceed?", true},
			{"Do you want to continue?", true},
			{"Do you want to", true},
			{"Permissions:", false},
			{"Claude Code won't ask", false},
			{"requires permission", false},
			{"needs your approval", false},
			{"Write(file.txt)", false},
			{"Read(config.json)", false},
			{"Bash(ls -la)", false},
			{"regular text", false},
		}
		
		for _, tc := range testCases {
			result := patterns.Permit.MatchString(tc.input)
			if result != tc.expected {
				t.Errorf("Permit pattern for %q: expected %v, got %v", tc.input, tc.expected, result)
			}
		}
	})
	
	t.Run("ChoiceYes pattern", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected bool
		}{
			{"1. Allow", true},
			{"2. Yes, proceed", true},
			{"3. Approve this action", true},
			{"1. Deny", false},
			{"2. No", false},
		}
		
		for _, tc := range testCases {
			result := patterns.ChoiceYes.MatchString(tc.input)
			if result != tc.expected {
				t.Errorf("ChoiceYes pattern for %q: expected %v, got %v", tc.input, tc.expected, result)
			}
		}
	})
}

func TestStripAnsi(t *testing.T) {
	patterns := NewRegexPatterns()
	
	testCases := []struct {
		input    string
		expected string
	}{
		{"\x1b[31mRed text\x1b[0m", "Red text"},
		{"\x1b[1;32mBold Green\x1b[0m", "Bold Green"},
		{"No ANSI codes", "No ANSI codes"},
		{"\x1b[2K\x1b[1GParsing...", "Parsing..."},
	}
	
	for _, tc := range testCases {
		result := patterns.StripAnsi(tc.input)
		if result != tc.expected {
			t.Errorf("StripAnsi for %q: expected %q, got %q", tc.input, tc.expected, result)
		}
	}
}

func TestAppState(t *testing.T) {
	state := NewAppState()
	patterns := NewRegexPatterns()
	
	t.Run("Initial state", func(t *testing.T) {
		if state.Dialog == nil {
			t.Error("Dialog state should not be nil")
		}
		if state.Prompt == nil {
			t.Error("Prompt state should not be nil")
		}
		if state.Prompt.CollectedChoices == nil {
			t.Error("CollectedChoices should not be nil")
		}
		if state.Prompt.Processed == nil {
			t.Error("Processed should not be nil")
		}
	})
	
	t.Run("ShouldProcessPrompt", func(t *testing.T) {
		prompt := "Do you want to proceed?"
		
		// First occurrence should be processed
		if !state.ShouldProcessPrompt(prompt, patterns) {
			t.Error("First occurrence should be processed")
		}
		
		// Same prompt should be detected as duplicate
		if state.ShouldProcessPrompt(prompt, patterns) {
			t.Error("Duplicate prompt should not be processed")
		}
		
		// Different prompt should be processed
		if !state.ShouldProcessPrompt("Different prompt?", patterns) {
			t.Error("Different prompt should be processed")
		}
	})
}

func TestContextCollection(t *testing.T) {
	state := NewAppState()
	patterns := NewRegexPatterns()
	
	t.Run("Context line collection", func(t *testing.T) {
		// Add some context lines
		state.AddContextLine("Writing file /path/to/file.txt", patterns)
		state.AddContextLine("Read(/path/to/config.json)", patterns)
		state.AddContextLine("Bash(npm install)", patterns)
		state.AddContextLine("permission request incoming", patterns)
		
		// Check context was collected
		if len(state.Prompt.Context) != 4 {
			t.Errorf("Expected 4 context lines, got %d", len(state.Prompt.Context))
		}
	})
}

// MockDialog for testing dialog functionality
type MockDialog struct {
	ReturnValue string
	CallCount   int
	LastMsg     string
	LastChoices map[string]string
}

func (m *MockDialog) AskWithChoices(msg string, choices map[string]string) string {
	m.CallCount++
	m.LastMsg = msg
	m.LastChoices = make(map[string]string)
	for k, v := range choices {
		m.LastChoices[k] = v
	}
	return m.ReturnValue
}

func TestDialogInterface(t *testing.T) {
	mock := &MockDialog{ReturnValue: "1"}
	
	result := mock.AskWithChoices("Test message", map[string]string{"1": "Yes", "2": "No"})
	
	if result != "1" {
		t.Errorf("Expected '1', got %q", result)
	}
	
	if mock.CallCount != 1 {
		t.Errorf("Expected 1 call, got %d", mock.CallCount)
	}
	
	if mock.LastMsg != "Test message" {
		t.Errorf("Expected 'Test message', got %q", mock.LastMsg)
	}
}