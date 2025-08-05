package parser

import (
	"errors"
	"strings"
)

// DialogInfo represents parsed dialog information
type DialogInfo struct {
	RawContent string // Raw dialog content as text
	ToolType   string // "Bash", "Edit", "Task", etc.
}

// ParseDialog parses a dialog box string and extracts structured information
func ParseDialog(input string) (*DialogInfo, error) {
	content := extractDialogContent(input)
	if len(content) == 0 {
		return nil, errors.New("no dialog content found")
	}

	result := &DialogInfo{}
	
	// Set raw content as the primary result  
	result.RawContent = strings.Join(content, "\n")
	
	// Optional: still extract some basic structured info for backward compatibility
	if len(content) > 0 {
		firstLine := content[0]
		if strings.Contains(firstLine, "Bash command") {
			result.ToolType = "Bash"
		} else if strings.Contains(firstLine, "Edit command") {
			result.ToolType = "Edit"
		} else if strings.Contains(firstLine, "Task") {
			result.ToolType = "Task"
		}
	}
	
	return result, nil
}


// Helper function to extract lines between dialog borders
func extractDialogContent(input string) []string {
	lines := strings.Split(input, "\n")
	var content []string
	
	inDialog := false
	hasStart := false
	hasEnd := false
	
	for _, line := range lines {
		if strings.Contains(line, "╭") {
			inDialog = true
			hasStart = true
			continue
		}
		if strings.Contains(line, "╰") {
			inDialog = false
			hasEnd = true
			break
		}
		if inDialog && strings.Contains(line, "│") {
			// Find first and last │ characters (Unicode pipe)
			firstPipe := strings.Index(line, "│")
			lastPipe := strings.LastIndex(line, "│")
			
			if firstPipe >= 0 && lastPipe > firstPipe {
				// Extract content between the pipes
				cleaned := line[firstPipe+len("│") : lastPipe]
				// Remove trailing whitespace but preserve leading spaces for indentation
				cleaned = strings.TrimRightFunc(cleaned, func(r rune) bool {
					return r == ' ' || r == '\t'
				})
				content = append(content, cleaned)
			}
		}
	}
	
	// Return empty if dialog is malformed (missing start or end)
	if hasStart && !hasEnd && len(content) > 0 {
		return []string{} // Malformed dialog
	}
	
	return content
}

