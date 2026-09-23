package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ToolRegistry struct {
	tools map[string]func(json.RawMessage) (string, error)
}

func newToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]func(json.RawMessage) (string, error)),
	}
}

func (r *ToolRegistry) Register(name string, fn func(json.RawMessage) (string, error)) {
	r.tools[name] = fn
}

func (r *ToolRegistry) Schemas() []Tool {
	return []Tool{
		{
			Name:        "move",
			Description: "Move the robot forward, back, left, right, or about_turn.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"direction": map[string]any{"type": "string", "enum": []string{"forward", "backward", "left", "right", "stop", "about_turn"}},
					"speed":     map[string]any{"type": "integer", "minimum": 0, "maximum": 255, "default": 150},
				},
				"required": []string{"direction"},
			},
		},
		{
			Name:        "stop",
			Description: "Stop all robot motors and halt movement immediately.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "look",
			Description: "Control the camera gimbal (pan/tilt).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"target": map[string]any{"type": "string", "enum": []string{"up", "down", "left", "right", "center"}},
				},
				"required": []string{"target"},
			},
		},
		{
			Name:        "light",
			Description: "Control the robot's WS2812 LEDs.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"color": map[string]any{"type": "string", "enum": []string{"red", "green", "blue", "yellow", "off"}},
				},
				"required": []string{"color"},
			},
		},
		{
			Name:        "play_resource",
			Description: "Play an audio file (song/soundtrack) from the resources folder.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"resource": map[string]any{"type": "string", "enum": []string{"pathfinder_song", "adventurer_song"}},
				},
				"required": []string{"resource"},
			},
		},
		{
			Name:        "read_document",
			Description: "Read a document (Law, Pledge, etc.) aloud.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"document": map[string]any{"type": "string", "enum": []string{"pathfinder_law", "pathfinder_pledge", "pathfinder_aim", "pathfinder_motto", "adventurer_law", "adventurer_pledge"}},
				},
				"required": []string{"document"},
			},
		},
		{
			Name:        "activate",
			Description: "Activate a robot mode.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"mode": map[string]any{"type": "string", "enum": []string{"birdwatch", "follow", "attention", "remote", "deep", "translate", "security"}},
				},
				"required": []string{"mode"},
			},
		},
		{
			Name:        "deactivate",
			Description: "Deactivate a robot mode or stop all activity.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"mode": map[string]any{"type": "string", "enum": []string{"birdwatch", "follow", "attention", "remote", "deep", "security", "translate"}},
				},
				"required": []string{"mode"},
			},
		},
	}
}

func (r *ToolRegistry) Execute(name string, args json.RawMessage) (string, error) {
	fn, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return fn(args)
}

var (
	sharedHardwareTools *ToolRegistry
	toolsOnce           sync.Once
)

func initHardwareTools() *ToolRegistry {
	toolsOnce.Do(func() {
		r := newToolRegistry()

		r.Register("move", func(args json.RawMessage) (string, error) {
			var a struct {
				Direction string `json:"direction"`
				Speed     int    `json:"speed"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", err
			}
			if a.Speed == 0 {
				a.Speed = 150
			}

			switch a.Direction {
			case "forward":
				for i := byte(0); i < 4; i++ {
					_ = moveMotor(i, 0, byte(a.Speed))
				}
			case "backward":
				for i := byte(0); i < 4; i++ {
					_ = moveMotor(i, 1, byte(a.Speed))
				}
			case "left":
				_ = moveMotor(0, 1, byte(a.Speed))
				_ = moveMotor(1, 0, byte(a.Speed))
				_ = moveMotor(2, 0, byte(a.Speed))
				_ = moveMotor(3, 1, byte(a.Speed))
			case "right":
				_ = moveMotor(0, 0, byte(a.Speed))
				_ = moveMotor(1, 1, byte(a.Speed))
				_ = moveMotor(2, 1, byte(a.Speed))
				_ = moveMotor(3, 0, byte(a.Speed))
			case "about_turn":
				_ = moveMotor(0, 1, byte(a.Speed))
				_ = moveMotor(1, 0, byte(a.Speed))
				_ = moveMotor(2, 1, byte(a.Speed))
				_ = moveMotor(3, 0, byte(a.Speed))
			case "stop":
				stopAllMotors()
			}
			return fmt.Sprintf("Moved %s at speed %d", a.Direction, a.Speed), nil
		})

		r.Register("stop", func(args json.RawMessage) (string, error) {
			stopAllMotors()
			return "Stopped all motors", nil
		})

		r.Register("look", func(args json.RawMessage) (string, error) {
			var a struct {
				Target string `json:"target"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", err
			}
			switch a.Target {
			case "up":
				_ = setServo(2, 170)
			case "down":
				_ = setServo(2, 30)
			case "left":
				_ = setServo(1, 150)
			case "right":
				_ = setServo(1, 30)
			case "center":
				_ = setServo(1, 90)
				_ = setServo(2, 75)
			}
			return fmt.Sprintf("Adjusted camera to %s", a.Target), nil
		})

		r.Register("light", func(args json.RawMessage) (string, error) {
			var a struct {
				Color string `json:"color"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", err
			}
			switch a.Color {
			case "red":
				_ = setLEDAll(1, LEDColorRed)
			case "green":
				_ = setLEDAll(1, LEDColorGreen)
			case "blue":
				_ = setLEDAll(1, LEDColorBlue)
			case "yellow":
				_ = setLEDAll(1, LEDColorYellow)
			case "off":
				_ = setLEDAll(0, 0)
			}
			return fmt.Sprintf("Set light to %s", a.Color), nil
		})

		r.Register("play_resource", func(args json.RawMessage) (string, error) {
			var a struct {
				Resource string `json:"resource"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", err
			}
			file := "Pathfinder Song.mp3"
			if strings.Contains(a.Resource, "adventurer") {
				file = "Adventurer Song.mp3"
			}
			go func() {
				_ = exec.Command("mpg123", "/home/pi/the-pathfinder-eye_ai/resources/"+file).Run()
			}()
			return "Playing " + a.Resource, nil
		})

		r.Register("read_document", func(args json.RawMessage) (string, error) {
			var a struct {
				Document string `json:"document"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", err
			}

			path := "/home/pi/the-pathfinder-eye_ai/resources/"
			switch a.Document {
			case "pathfinder_law":
				path += "Pathfinder Law.md"
			case "pathfinder_pledge":
				path += "Pathfinder Pledge.md"
			case "pathfinder_aim":
				path += "Pathfinder Aim.md"
			case "pathfinder_motto":
				path += "Pathfinder Motto.md"
			case "adventurer_law":
				path += "Adventurer Law.md"
			case "adventurer_pledge":
				path += "Adventurer Pledge.md"
			}
			go readDocument(path)
			return "Reading " + a.Document, nil
		})

		r.Register("activate", func(args json.RawMessage) (string, error) {
			var a struct {
				Mode string `json:"mode"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", err
			}
			cmd := ParsedCommand{Action: "activate", Target: a.Mode}
			if handleActivateAction(cmd) {
				return "Activated " + a.Mode, nil
			}
			return "Activated " + a.Mode, nil
		})

		r.Register("deactivate", func(args json.RawMessage) (string, error) {
			var a struct {
				Mode string `json:"mode"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", err
			}
			cmd := ParsedCommand{Action: "deactivate", Target: a.Mode}
			if handleDeactivateAction(cmd) {
				return "Deactivated " + a.Mode, nil
			}
			return "Deactivated " + a.Mode, nil
		})

		sharedHardwareTools = r
	})
	return sharedHardwareTools
}
