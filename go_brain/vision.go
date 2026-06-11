/**
 * THE-PATHFINDER-EYE : Vision Fusion Module (v3.0 - Textual Awareness)
 */

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"
)

type Detection struct {
	X          int     `json:"x"`
	Y          int     `json:"y"`
	W          int     `json:"w"`
	H          int     `json:"h"`
	ClassName  string  `json:"class_name"`
	Species    string  `json:"species"` // for birdwatch
	Confidence float32 `json:"confidence"`
}

type DetectionFrame struct {
	Timestamp  string      `json:"timestamp"`
	Detections []Detection `json:"detections"`
}

type WorldState struct {
	SeenObjects []string
	Distances   map[string]float64
	FaceVisible bool
	BirdVisible bool
	SafetyRisk  bool
	mu          sync.Mutex
}

var currentState WorldState

func readLatestDetections() (*DetectionFrame, error) {
	data, err := ioutil.ReadFile("/tmp/detections.json")
	if err != nil {
		return nil, err
	}
	var frame DetectionFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		return nil, err
	}
	return &frame, nil
}

func visionPollerLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	for range ticker.C {
		frame, _ := readLatestDetections()
		if frame == nil {
			continue
		}

		birdDetectedThisFrame := false
		faceDetectedThisFrame := false
		var objects []string

		for _, d := range frame.Detections {
			objects = append(objects, d.ClassName)
			
			if d.ClassName == "face" {
				faceDetectedThisFrame = true
				if visionDB != nil { _ = visionDB.storeDetection(d, frame.Timestamp) }
			}

			if birdwatchActive && (d.ClassName == "bird" || d.ClassName == "animal") {
				birdDetectedThisFrame = true
				gimbal.TrackObject(d.X, d.Y, d.W, d.H, 640, 480)
			}
		}

		// Update global awareness
		currentState.mu.Lock()
		currentState.SeenObjects = objects
		currentState.FaceVisible = faceDetectedThisFrame
		currentState.BirdVisible = birdDetectedThisFrame
		currentState.mu.Unlock()

		// Active seeking if looking for something specific
		if birdwatchActive && !birdDetectedThisFrame {
			seekGimbal()
		}
	}
}

var seekDirectionPan int = 15
var seekDirectionTilt int = 10
var seekPan int = 90
var seekTilt int = 75

func seekGimbal() {
    seekPan += seekDirectionPan
    if seekPan > 150 {
        seekDirectionPan = -15
        seekTilt += seekDirectionTilt
        if seekTilt > 130 { seekDirectionTilt = -10 } else if seekTilt < 30 { seekDirectionTilt = 10 }
    } else if seekPan < 30 {
        seekDirectionPan = 15
        seekTilt += seekDirectionTilt
        if seekTilt > 130 { seekDirectionTilt = -10 } else if seekTilt < 30 { seekDirectionTilt = 10 }
    }
    _ = setServo(1, byte(seekPan))
    _ = setServo(2, byte(seekTilt))
}

func GetWorldStatePrompt() string {
	currentState.mu.Lock()
	defer currentState.mu.Unlock()
	
	state := fmt.Sprintf("Objects in view: %v. Face Visible: %v. Bird Visible: %v.", 
		currentState.SeenObjects, currentState.FaceVisible, currentState.BirdVisible)
	return state
}

func initVisionDB() (*VisionDB, error) {
    return &VisionDB{}, nil
}

type VisionDB struct{}
func (db *VisionDB) storeDetection(d Detection, ts string) error { return nil }
func (db *VisionDB) GetCurrentSpeaker() (struct{ FaceID string }, error) { 
	return struct{ FaceID string }{FaceID: "unknown"}, nil 
}
