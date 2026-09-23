package main

import (
	"testing"
)

func TestNeedle2ToolCalling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantTool string
	}{
		{"move forward", "move forward", "move"},
		{"turn left", "turn left", "move"},
		{"about turn", "turn around", "move"},
		{"stop", "stop", "stop"},
		{"look up", "look up", "look"},
		{"set red light", "set light red", "light"},
		{"play song", "play pathfinder song", "play_resource"},
		{"read pledge", "recite pathfinder pledge", "read_document"},
		{"activate follow", "activate follow mode", "activate"},
		{"deactivate security", "disable security mode", "deactivate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools := getCachedToolSchemas()
			found := false
			for _, tool := range tools {
				if tool.Name == tt.wantTool {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Tool %q not found in Needle 2 schema", tt.wantTool)
			}
		})
	}
}
