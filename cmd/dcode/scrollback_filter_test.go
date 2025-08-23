package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/takahirom/dialog-code/internal/dialog"
)

func TestScrollbackClearFilterWriter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no control sequences",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "scrollback clear sequence",
			input:    "\x1b[3JHello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "multiple scrollback clear sequences",
			input:    "\x1b[3JHello\x1b[3J, World!\x1b[3J",
			expected: "Hello, World!",
		},
		{
			name:     "mixed with other sequences",
			input:    "\x1b[31mRed\x1b[3JText\x1b[0m",
			expected: "\x1b[31mRedText\x1b[0m",
		},
		{
			name:     "scrollback clear in middle",
			input:    "Before\x1b[3JAfter",
			expected: "BeforeAfter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := dialog.NewScrollbackClearFilterWriter(&buf)
			
			_, err := writer.Write([]byte(tt.input))
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}
			
			result := buf.String()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestScrollbackClearFilterWithColorStrip(t *testing.T) {
	var buf bytes.Buffer
	// Chain the writers: first filter scrollback clear, then strip colors
	scrollbackFilter := dialog.NewScrollbackClearFilterWriter(&buf)
	colorStripWriter := dialog.NewColorStripWriter(scrollbackFilter)
	
	input := "\x1b[31m\x1b[3JRed Text\x1b[0m\x1b[3J"
	expected := "Red Text"
	
	_, err := colorStripWriter.Write([]byte(input))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	
	result := strings.TrimSpace(buf.String())
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}