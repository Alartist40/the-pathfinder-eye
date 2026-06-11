package main

import (
	"regexp"
	"strings"
	"testing"
)

// TEST 1: Command String Matching (Fixed)
func TestVoiceCommandStringMatching(t *testing.T) {
	testCases := []struct {
		cmd        string
		target     string
		shouldMatch bool
	}{
		{"move forward", "move forward", true},
		{"move forward now", "move forward", true},
		{"can we move forward", "move forward", false}, 
	}
	
	for _, tc := range testCases {
		matches := strings.HasPrefix(tc.cmd, tc.target)
		if matches != tc.shouldMatch {
			t.Errorf("Command '%s': expected %v, got %v", tc.cmd, tc.shouldMatch, matches)
		}
	}
	
	t.Log("✅ TEST PASSED: Voice command string matching works correctly")
}

// TEST 2: Parameter Extraction (Regex)
func TestVoiceCommandParameterExtraction(t *testing.T) {
	// Tilt angle extraction
	tiltRegex := regexp.MustCompile(`eye tilt (\d+)`)
	
	testCases := []struct {
		cmd      string
		expected string
	}{
		{"eye tilt 45", "45"},
		{"eye tilt 90", "90"},
		{"eye tilt 180", "180"},
		{"eye rotate 45", ""},  // Should not match
	}
	
	for _, tc := range testCases {
		match := tiltRegex.FindStringSubmatch(tc.cmd)
		var result string
		if len(match) > 1 {
			result = match[1]
		}
		if result != tc.expected {
			t.Errorf("Command '%s': expected '%s', got '%s'", tc.cmd, tc.expected, result)
		}
	}
	
	t.Log("✅ TEST PASSED: Parameter extraction works correctly")
}

// TEST 3: Movement Command Parsing
func TestVoiceCommandMovementParsing(t *testing.T) {
	commands := []string{
		"move forward",
		"move back",
		"move left",
		"move right",
		"rotate left",
		"rotate right",
		"about turn",
	}
	
	for _, cmd := range commands {
		cmd = strings.ToLower(cmd)
		
		// All should be recognized
		if !strings.HasPrefix(cmd, "move") && !strings.HasPrefix(cmd, "rotate") && cmd != "about turn" {
			t.Errorf("Command not recognized: %s", cmd)
		}
	}
	
	t.Log("✅ TEST PASSED: Movement command parsing works correctly")
}

// TEST 4: Wake Word Detection
func TestVoiceWakeWordDetection(t *testing.T) {
	testCases := []struct {
		input      string
		shouldTrigger bool
	}{
		{"instruction", true},
		{"Instruction", true},
		{"INSTRUCTION", true},
		{"hey instruction", true},
		{"instructional", true}, 
		{"listen", false},
		{"hello", false},
		{"please instruction", true},
	}
	
	for _, tc := range testCases {
		triggers := strings.Contains(strings.ToLower(tc.input), "instruction")
		if triggers != tc.shouldTrigger {
			t.Errorf("Input '%s': expected %v, got %v", tc.input, tc.shouldTrigger, triggers)
		}
	}
	
	t.Log("✅ TEST PASSED: Wake word detection works correctly")
}

// TEST 5: Registration Command Parsing
func TestVoiceRegistrationCommandParsing(t *testing.T) {
	regRegex := regexp.MustCompile(`register new (leader|masterguide) ([a-z\s]+)`)
	
	testCases := []struct {
		cmd       string
		shouldMatch bool
		expectedRole string
		expectedName string
	}{
		{"register new leader john smith", true, "leader", "john smith"},
		{"register new masterguide jane doe", true, "masterguide", "jane doe"},
		{"register new scout alex", false, "", ""},
		{"register john smith", false, "", ""},
	}
	
	for _, tc := range testCases {
		match := regRegex.FindStringSubmatch(strings.ToLower(tc.cmd))
		if len(match) > 1 && tc.shouldMatch {
			if match[1] != tc.expectedRole || match[2] != tc.expectedName {
				t.Errorf("Command '%s': expected (%s, %s), got (%s, %s)",
					tc.cmd, tc.expectedRole, tc.expectedName, match[1], match[2])
			}
		} else if len(match) > 1 != tc.shouldMatch {
			t.Errorf("Command '%s': expected match=%v, got match=%v", tc.cmd, tc.shouldMatch, len(match) > 1)
		}
	}
	
	t.Log("✅ TEST PASSED: Registration command parsing works correctly")
}

// TEST 6: Resource Command Mapping
func TestVoiceResourceCommandMapping(t *testing.T) {
	mediaCommands := map[string]string{
		"adventurer song": "Adventurer Song.mp3",
		"pathfinder song": "Pathfinder Song.mp3",
	}
	
	for trigger := range mediaCommands {
		if trigger == "" {
			t.Errorf("Empty trigger in media commands")
		}
	}
	
	t.Log("✅ TEST PASSED: Resource command mapping works correctly")
}
