package main

import (
	"strings"
)

// ParsedCommand represents a parsed voice command extracted from STT output.
type ParsedCommand struct {
	Action    string            // "move", "look", "play", "read", "activate", "deactivate", "test", "translate", "attention", "remote", "deep", "stop"
	Target    string            // "forward", "birdwatch", "law", "security", etc.
	Modifiers map[string]string // extra params: "speed", "direction", etc.
}

// actionAliases map canonical action verbs to the ParsedCommand.Action field.
var actionAliases = map[string]string{
	// move/turn variants
	"go": "move", "move": "move", "forward": "move", "back": "move",
	"backward": "move", "backwards": "move", "turn": "move", "left": "move",
	"right": "move", "spin": "move",
	// look/scan/view variants
	"look": "look", "scan": "look", "see": "look", "view": "look", "gaze": "look",
	// play/audio variants
	"play": "play", "song": "play", "music": "play", "audio": "play", "sound": "play",
	// read/recite variants
	"read": "read", "recite": "read", "say": "read", "tell": "read",
	// activate/enable variants
	"activate": "activate", "enable": "activate", "start": "activate",
	"enter": "activate", "mode": "activate", "initiate": "activate",
	// deactivate/disable variants
	"deactivate": "deactivate", "disable": "deactivate", "stop": "deactivate",
	"exit": "deactivate", "sleep": "deactivate",
	// diagnostic
	"test": "test", "diagnostic": "test", "check": "test", "run": "test",
	// translation
	"translate": "translate", "japanese": "translate",
	// AI activation
	"attention": "attention", "awaken": "attention", "wake": "attention",
	// modes
	"remote": "remote", "remote control": "remote",
	"deep": "deep", "thinking": "deep",
}

// targetAliases normalize STT variations to canonical target names.
var targetAliases = map[string]string{
	// birdwatch STT variations
	"birdwatch": "birdwatch", "bired": "birdwatch", "beard": "birdwatch",
	"word": "birdwatch", "third": "birdwatch",
	// movement directions
	"forward": "forward", "ahead": "forward", "straight": "forward",
	"back": "backward", "backward": "backward", "backwards": "backward",
	"left": "left", "turn left": "left",
	"right": "right", "turn right": "right",
	// look targets
	"up":     "up",
	"down":   "down",
	"center": "center", "middle": "center",
	"look left":  "left",
	"look right": "right",
	// documents
	"law": "law", "pathfinder law": "law", "adventurer law": "law",
	"pledge": "pledge", "pathfinder pledge": "pledge", "adventurer pledge": "pledge",
	"aim": "aim", "pathfinder aim": "aim", "adventurer aim": "aim",
	"motto": "motto", "pathfinder motto": "motto", "adventurer motto": "motto",
	// songs
	"pathfinder song": "pathfinder_song", "pathfinder soundtrack": "pathfinder_song",
	"adventurer song": "adventurer_song", "adventurer soundtrack": "adventurer_song",
	"adventure": "adventurer_song",
	// security
	"security": "security", "camera": "security", "pir": "security", "cameras": "security",
	"security mode": "security",
	// follow
	"follow": "follow", "track": "follow", "tracking": "follow", "follow mode": "follow",
	// translation
	"japanese": "japanese", "japan": "japanese",
	"remote control": "remote",
	"deep thought":   "deep", "thinking mode": "deep",
	"about turn": "about_turn", "turn about": "about_turn",
}

// ExtractCommand parses STT output into a structured command.
// Uses word-boundary tokenization for STT robustness.
func ExtractCommand(text string) ParsedCommand {
	cmd := ParsedCommand{Modifiers: make(map[string]string)}
	lower := strings.ToLower(text)
	tokens := tokenize(lower)

	// 1. Detect canonical action from first token or keyword match.
	if len(tokens) > 0 {
		if alias, ok := actionAliases[tokens[0]]; ok {
			cmd.Action = alias
		}
	}
	// Fallback: scan all tokens for an action verb.
	if cmd.Action == "" {
		for _, tok := range tokens {
			if alias, ok := actionAliases[tok]; ok {
				cmd.Action = alias
				break
			}
		}
	}

	// 2. Detect target by scanning for target keywords.
	bestTarget := ""
	for _, tok := range tokens {
		if alias, ok := targetAliases[tok]; ok {
			bestTarget = alias
			break // first match is fine
		}
	}

	// Special case: "about turn" and "turn about" need both tokens.
	if strings.Contains(lower, "about turn") || strings.Contains(lower, "turn about") {
		bestTarget = "about_turn"
	}
	if strings.Contains(lower, "deep thought") || strings.Contains(lower, "thinking mode") {
		bestTarget = "deep"
		cmd.Action = "deep"
	}

	cmd.Target = bestTarget

	// 3. Extract numeric modifiers (speed, duration, angle).
	for i, tok := range tokens {
		if isNumber(tok) && i > 0 {
			prev := tokens[i-1]
			if prev == "speed" || prev == "fast" || prev == "slow" {
				cmd.Modifiers["speed"] = tok
			} else if prev == "angle" || prev == "degrees" {
				cmd.Modifiers["angle"] = tok
			}
		}
		if tok == "fast" {
			cmd.Modifiers["speed"] = "200"
		} else if tok == "slow" {
			cmd.Modifiers["speed"] = "80"
		}
	}

	return cmd
}

// tokenize splits on whitespace and strips punctuation.
func tokenize(s string) []string {
	var out []string
	var current []rune
	for _, r := range s {
		if unicodeIsSpace(r) || unicodeIsPunct(r) {
			if len(current) > 0 {
				out = append(out, string(current))
				current = nil
			}
		} else {
			current = append(current, r)
		}
	}
	if len(current) > 0 {
		out = append(out, string(current))
	}
	return out
}

func unicodeIsSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func unicodeIsPunct(r rune) bool {
	return strings.ContainsRune(".,!?;:'\"-()[]{}@#$%^&*+=<>|\\/~`", r)
}

func isNumber(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			if c != '.' {
				return false
			}
		}
	}
	return len(s) > 0
}
