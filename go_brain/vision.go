/**
 * THE-PATHFINDER-EYE : Vision Fusion Module (v3.1 - Textual Awareness + DB-backed)
 */

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
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

			if d.ClassName == "face" || strings.HasPrefix(d.ClassName, "face:") {
				faceDetectedThisFrame = true
				if visionDB != nil {
					_ = visionDB.storeDetection(d, frame.Timestamp)
				}
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

// seekGimbal state is protected by seekMu
var (
	seekMu            sync.Mutex
	seekDirectionPan  int = 15
	seekDirectionTilt int = 10
	seekPan           int = 90
	seekTilt          int = 75
)

func seekGimbal() {
	seekMu.Lock()
	defer seekMu.Unlock()

	seekPan += seekDirectionPan
	if seekPan > 150 {
		seekDirectionPan = -15
		seekTilt += seekDirectionTilt
		if seekTilt > 130 {
			seekDirectionTilt = -10
		} else if seekTilt < 30 {
			seekDirectionTilt = 10
		}
	} else if seekPan < 30 {
		seekDirectionPan = 15
		seekTilt += seekDirectionTilt
		if seekTilt > 130 {
			seekDirectionTilt = -10
		} else if seekTilt < 30 {
			seekDirectionTilt = 10
		}
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
	dbPath := "../db/vision.sqlite"
	db, err := sql.Open("sqlite3", dbPath+"?_journal=WAL")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS face_detections (
		id INTEGER PRIMARY KEY,
		face_id TEXT,
		person_name TEXT,
		x INTEGER, y INTEGER, w INTEGER, h INTEGER,
		detected_at TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_faces_recent ON face_detections(created_at DESC);
	`)
	if err != nil {
		return nil, err
	}
	return &VisionDB{db: db}, nil
}

type VisionDB struct {
	db *sql.DB
}

func (db *VisionDB) storeDetection(d Detection, ts string) error {
	personName := ""
	faceID := d.ClassName
	if strings.HasPrefix(d.ClassName, "face:") {
		personName = strings.TrimPrefix(d.ClassName, "face:")
		faceID = "face"
	}
	_, err := db.db.Exec(
		"INSERT INTO face_detections (face_id, person_name, x, y, w, h, detected_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		faceID, personName, d.X, d.Y, d.W, d.H, ts,
	)
	return err
}

func (db *VisionDB) GetCurrentSpeaker() (struct{ FaceID string }, error) {
	if db == nil || db.db == nil {
		return struct{ FaceID string }{FaceID: "unknown"}, nil
	}
	var personName string
	err := db.db.QueryRow(
		"SELECT COALESCE(person_name, 'unknown') FROM face_detections ORDER BY created_at DESC LIMIT 1",
	).Scan(&personName)
	if err != nil {
		return struct{ FaceID string }{FaceID: "unknown"}, nil
	}
	return struct{ FaceID string }{FaceID: personName}, nil
}

func (db *VisionDB) Close() {
	if db != nil && db.db != nil {
		db.db.Close()
	}
}
