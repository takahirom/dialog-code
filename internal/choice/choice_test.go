package choice

import (
	"strings"
	"testing"

	"github.com/takahirom/dialog-code/internal/types"
)

func TestGetBestChoice(t *testing.T) {
	patterns := types.NewRegexPatterns()
	
	t.Run("Prefer Allow/Yes choices", func(t *testing.T) {
		choices := map[string]string{
			"1": "1. Allow this action",
			"2": "2. Deny this action",
		}
		
		result := GetBestChoice(choices, patterns)
		if result != "1" {
			t.Errorf("Expected choice 1 (Allow), got %q", result)
		}
	})
	
	t.Run("Prefer Add a new rule", func(t *testing.T) {
		choices := map[string]string{
			"1": "1. Add a new rule",
			"2": "2. Deny this action",
		}
		
		result := GetBestChoice(choices, patterns)
		if result != "1" {
			t.Errorf("Expected choice 1 (Add a new rule), got %q", result)
		}
	})
	
	t.Run("Fallback to first available", func(t *testing.T) {
		choices := map[string]string{
			"3": "3. Some other option",
		}
		
		result := GetBestChoice(choices, patterns)
		if result != "3" {
			t.Errorf("Expected choice 3 (fallback), got %q", result)
		}
	})
	
	t.Run("Ultimate fallback", func(t *testing.T) {
		choices := map[string]string{}
		
		result := GetBestChoice(choices, patterns)
		if result != "1" {
			t.Errorf("Expected choice 1 (ultimate fallback), got %q", result)
		}
	})
}

func TestGetBestChoiceFromState(t *testing.T) {
	state := types.NewAppState()
	patterns := types.NewRegexPatterns()
	
	state.Prompt.CollectedChoices["1"] = "1. Allow this action"
	state.Prompt.CollectedChoices["2"] = "2. Deny this action"
	
	result := GetBestChoiceFromState(state, patterns)
	if result != "1" {
		t.Errorf("Expected choice 1, got %q", result)
	}
}

func TestGetContextualMessage(t *testing.T) {
	patterns := types.NewRegexPatterns()
	
	context := []string{
		"Write(/path/to/file.txt)",
		"Read(/path/to/config.json)",
		"Bash(npm install)",
		"Some random text that should be filtered",
		"permission needed for operation",
	}
	
	msg := GetContextualMessage("Do you want to proceed?", context, patterns)
	
	// Should contain the main prompt
	if !strings.Contains(msg, "Do you want to proceed?") {
		t.Error("Message should contain main prompt")
	}
	
	// Should contain context section
	if !strings.Contains(msg, "Context:") {
		t.Error("Message should contain context section")
	}
	
	// Should contain relevant context lines
	if !strings.Contains(msg, "Write(/path/to/file.txt)") {
		t.Error("Message should contain Write operation context")
	}
	
	if !strings.Contains(msg, "Read(/path/to/config.json)") {
		t.Error("Message should contain Read operation context")
	}
	
	// Should contain permission-related context
	if !strings.Contains(msg, "permission needed for operation") {
		t.Error("Message should contain permission-related context")
	}
}