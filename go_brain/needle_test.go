package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
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
		{"look up", "look up", "look"},
		{"set red light", "set light red", "light"},
		{"play song", "play pathfinder song", "play_resource"},
		{"read pledge", "recite pathfinder pledge", "read_document"},
		{"activate follow", "activate follow mode", "activate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify that Needle 2 tool schemas include the expected tools.
			// This is a schema-level test — the actual inference requires
			// the Needle 2 service to be running.
			tools := loadToolSchemas()
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

func loadToolSchemas() []Tool {
	path := "../config/needle_tools.json"
	if _, err := os.Stat(path); err != nil {
		path = "config/needle_tools.json"
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}
	var rawTools []struct {
		Function struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"function"`
	}
	_ = json.Unmarshal(data, &rawTools)
	var tools []Tool
	for _, t := range rawTools {
		tools = append(tools, Tool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
		})
	}
	return tools
}
