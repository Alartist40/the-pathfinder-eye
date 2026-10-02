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
	"right": "move", "spin": "move", "drive": "move", "rotate": "move",
	// stop variants
	"stop": "stop", "halt": "stop", "freeze": "stop", "brake": "stop",
	// look/scan/view variants
	"look": "look", "scan": "look", "see": "look", "view": "look", "gaze": "look",
	"tilt": "look", "pan": "look", "camera": "look",
	// light variants
	"light": "light", "lights": "light", "headlight": "light", "headlights": "light", "lamp": "light",
	// play/audio variants
	"play": "play", "song": "play", "music": "play", "audio": "play", "sound": "play", "sing": "play",
	// read/recite variants
	"read": "read", "recite": "read", "say": "read", "tell": "read",
	"instructions": "read", "instruction": "read",
	// activate/enable variants
	"activate": "activate", "enable": "activate", "start": "activate",
	"enter": "activate", "mode": "activate", "initiate": "activate",
	// deactivate/disable variants
	"deactivate": "deactivate", "disable": "deactivate",
	"exit": "deactivate", "sleep": "deactivate",
	// status & help
	"status": "status", "health": "status", "stats": "status",
	"help": "help", "commands": "help",
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
	"birdwatch": "birdwatch", "bird watch": "birdwatch",
	"bird": "birdwatch", "bired": "birdwatch", "beard": "birdwatch",
	"word": "birdwatch", "third": "birdwatch",
	// movement directions
	"forward": "forward", "ahead": "forward", "straight": "forward", "front": "forward",
	"back": "backward", "backward": "backward", "backwards": "backward", "reverse": "backward",
	"left": "left", "turn left": "left", "strafe left": "left",
	"right": "right", "turn right": "right", "strafe right": "right",
	"stop": "stop",
	// look targets
	"up":     "up",
	"down":   "down",
	"center": "center", "middle": "center",
	"look left":  "left",
	"look right": "right",
	// light targets
	"on": "on", "off": "off", "white": "white", "red": "red", "green": "green", "blue": "blue", "yellow": "yellow", "strobe": "strobe",
	// documents
	"law": "law", "pathfinder law": "law", "adventurer law": "adventurer_law",
	"pledge": "pledge", "pathfinder pledge": "pledge", "adventurer pledge": "adventurer_pledge",
	"aim": "aim", "pathfinder aim": "aim", "adventurer aim": "adventurer_aim",
	"motto": "motto", "pathfinder motto": "motto", "adventurer motto": "adventurer_motto",
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
	"about turn": "about_turn", "turn about": "about_turn", "turn around": "about_turn", "u turn": "about_turn",
}

// ExtractCommand parses STT output into a structured command.
// Uses word-boundary tokenization for STT robustness.
func ExtractCommand(text string) ParsedCommand {
	cmd := ParsedCommand{Modifiers: make(map[string]string)}
	lower := strings.ToLower(text)
	tokens := tokenize(lower)

	// Direct phrase checks
	if strings.Contains(lower, "turn around") || strings.Contains(lower, "about turn") || strings.Contains(lower, "turn about") {
		return ParsedCommand{Action: "move", Target: "about_turn", Modifiers: map[string]string{}}
	}
	if strings.Contains(lower, "stop") || strings.Contains(lower, "halt") || strings.Contains(lower, "freeze") {
		return ParsedCommand{Action: "stop", Target: "stop", Modifiers: map[string]string{}}
	}
	if strings.Contains(lower, "help") || strings.Contains(lower, "what can you do") || strings.Contains(lower, "commands") {
		return ParsedCommand{Action: "help", Target: "help", Modifiers: map[string]string{}}
	}
	if strings.Contains(lower, "status") || strings.Contains(lower, "how are you") || strings.Contains(lower, "system health") {
		return ParsedCommand{Action: "status", Target: "status", Modifiers: map[string]string{}}
	}
	if strings.Contains(lower, "lights on") || strings.Contains(lower, "turn on lights") || strings.Contains(lower, "headlights on") {
		return ParsedCommand{Action: "light", Target: "on", Modifiers: map[string]string{}}
	}
	if strings.Contains(lower, "lights off") || strings.Contains(lower, "turn off lights") || strings.Contains(lower, "headlights off") {
		return ParsedCommand{Action: "light", Target: "off", Modifiers: map[string]string{}}
	}
	if strings.Contains(lower, "lights red") || strings.Contains(lower, "red light") {
		return ParsedCommand{Action: "light", Target: "red", Modifiers: map[string]string{}}
	}
	if strings.Contains(lower, "lights green") || strings.Contains(lower, "green light") {
		return ParsedCommand{Action: "light", Target: "green", Modifiers: map[string]string{}}
	}
	if strings.Contains(lower, "lights blue") || strings.Contains(lower, "blue light") {
		return ParsedCommand{Action: "light", Target: "blue", Modifiers: map[string]string{}}
	}
	if strings.Contains(lower, "lights yellow") || strings.Contains(lower, "yellow light") {
		return ParsedCommand{Action: "light", Target: "yellow", Modifiers: map[string]string{}}
	}
	if strings.Contains(lower, "pathfinder law") || strings.Contains(lower, "read law") || strings.Contains(lower, "the law") {
		return ParsedCommand{Action: "read", Target: "law", Modifiers: map[string]string{}}
	}
	if strings.Contains(lower, "pathfinder pledge") || strings.Contains(lower, "read pledge") || strings.Contains(lower, "the pledge") {
		return ParsedCommand{Action: "read", Target: "pledge", Modifiers: map[string]string{}}
	}
	if strings.Contains(lower, "pathfinder aim") || strings.Contains(lower, "read aim") || strings.Contains(lower, "the aim") {
		return ParsedCommand{Action: "read", Target: "aim", Modifiers: map[string]string{}}
	}
	if strings.Contains(lower, "pathfinder motto") || strings.Contains(lower, "read motto") || strings.Contains(lower, "the motto") {
		return ParsedCommand{Action: "read", Target: "motto", Modifiers: map[string]string{}}
	}

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

	// Special case: multi-word phrases need both tokens.
	if strings.Contains(lower, "about turn") || strings.Contains(lower, "turn about") {
		bestTarget = "about_turn"
	}
	if strings.Contains(lower, "adventurer song") || strings.Contains(lower, "adventurer soundtrack") {
		bestTarget = "adventurer_song"
		cmd.Action = "play"
	}
	if strings.Contains(lower, "pathfinder song") || strings.Contains(lower, "pathfinder soundtrack") {
		if bestTarget == "" {
			bestTarget = "pathfinder_song"
			cmd.Action = "play"
		}
	}
	if strings.Contains(lower, "deep thought") || strings.Contains(lower, "thinking mode") {
		bestTarget = "deep"
		cmd.Action = "deep"
	}
	if strings.Contains(lower, "adventurer instructions") || strings.Contains(lower, "adventurer instruction") {
		bestTarget = "adventurer_law"
		cmd.Action = "read"
	}
	if strings.Contains(lower, "adventurer pledge") {
		bestTarget = "adventurer_pledge"
		cmd.Action = "read"
	}
	if strings.Contains(lower, "adventurer law") {
		bestTarget = "adventurer_law"
		cmd.Action = "read"
	}
	if strings.Contains(lower, "adventurer aim") {
		bestTarget = "adventurer_aim"
		cmd.Action = "read"
	}
	if strings.Contains(lower, "adventurer motto") {
		bestTarget = "adventurer_motto"
		cmd.Action = "read"
	}
	if strings.Contains(lower, "pathfinder instructions") || strings.Contains(lower, "pathfinder instruction") {
		bestTarget = "law"
		cmd.Action = "read"
	}
	if strings.Contains(lower, "instructions") || strings.Contains(lower, "instruction") {
		if bestTarget == "" {
			bestTarget = "law"
			cmd.Action = "read"
		}
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
