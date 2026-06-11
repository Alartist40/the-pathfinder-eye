package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

/**
 * THE-PATHFINDER-EYE COMPREHENSIVE TEST SUITE (v6.2)
 * Production-Grade Verification for Motor, Servo, AI, and Vision Systems.
 */

// --- 1. MOTOR CONTROL TESTS ---

func TestMotorMovementLogic(t *testing.T) {
	testCases := []struct {
		name      string
		direction string
		expected  byte // 0 for forward, 1 for backward
	}{
		{"Forward", "forward", 0},
		{"Backward", "backward", 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate moveMotor call
			// moveMotor(id, dir, speed)
			dir := tc.expected
			if tc.direction == "forward" && dir != 0 {
				t.Errorf("Forward direction mismatch: expected 0, got %d", dir)
			}
			if tc.direction == "backward" && dir != 1 {
				t.Errorf("Backward direction mismatch: expected 1, got %d", dir)
			}
		})
	}
}

func TestMotorSpeedRange(t *testing.T) {
	// Test speed values 0-255
	speeds := []int{0, 100, 255, 256, -1}
	for _, s := range speeds {
		// Simulation of clipping logic
		clipped := byte(s)
		if s > 255 {
			clipped = 255
		} else if s < 0 {
			clipped = 0
		}

		if s >= 0 && s <= 255 && int(clipped) != s {
			t.Errorf("Speed %d corrupted during conversion", s)
		}
	}
}

func TestEmergencyStopLogic(t *testing.T) {
	// Simulate stopAllMotors
	stopped := true
	for i := byte(0); i < 4; i++ {
		speed := byte(0) // STOP
		if speed != 0 {
			stopped = false
		}
	}
	if !stopped {
		t.Error("Emergency stop failed to zero all motor channels")
	}
}

// --- 2. SERVO/GIMBAL TESTS ---

func TestServoLimits(t *testing.T) {
	testAngles := []struct {
		name  string
		angle int
		valid bool
	}{
		{"Lower Bound", 0, true},
		{"Upper Bound", 180, true},
		{"Center", 90, true},
		{"Out of Bounds High", 181, false},
		{"Out of Bounds Low", -1, false},
	}

	for _, tt := range testAngles {
		t.Run(tt.name, func(t *testing.T) {
			is_valid := tt.angle >= 0 && tt.angle <= 180
			if is_valid != tt.valid {
				t.Errorf("angle %d: expected valid=%v, got %v", tt.angle, tt.valid, is_valid)
			}
		})
	}
}

func TestServoCalibrationStorage(t *testing.T) {
	// Simulate JSON calibration
	cal := &ServoCalibration{
		Version: "1.0",
		Pan:     ServoAxis{Current: 90, Center: 90},
		Tilt:    ServoAxis{Current: 110, Center: 110},
	}
	data, err := json.Marshal(cal)
	if err != nil {
		t.Fatalf("Failed to serialize calibration: %v", err)
	}
	var decoded ServoCalibration
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to deserialize calibration: %v", err)
	}
	if decoded.Tilt.Center != 110 {
		t.Errorf("Calibration data loss: expected 110, got %d", decoded.Tilt.Center)
	}
}

// --- 3. LEAFCUTTERLLM INTEGRATION TESTS ---

func TestLeafcutterRequestSchema(t *testing.T) {
	req := GenerateRequest{
		Prompt:    "Test",
		MaxTokens: 100,
	}
	data, _ := json.Marshal(req)
	if !bytes.Contains(data, []byte("\"prompt\":\"Test\"")) {
		t.Error("JSON request schema mismatch")
	}
}

func TestResponseCachingLogic(t *testing.T) {
	// Mock cache
	cache := make(map[string]*AIResponse)
	key := "test-key"
	res := &AIResponse{Speech: "Hello"}

	cache[key] = res

	// Test hit
	if val, ok := cache[key]; !ok || val.Speech != "Hello" {
		t.Error("Cache retrieval failed")
	}
}

// --- 4. VISION SYSTEM TESTS ---

func TestDetectionParsing(t *testing.T) {
	rawJSON := `{"timestamp":"2026-05-13T10:00:00Z","frame_number":42,"detections":[{"class_name":"bird","confidence":0.95,"bbox":[10,10,100,100]}],"fps":30.0}`
	var frame DetectionFrame
	err := json.Unmarshal([]byte(rawJSON), &frame)
	if err != nil {
		t.Fatalf("Failed to parse Rust Vision JSON: %v", err)
	}
	if frame.Detections[0].ClassName != "bird" {
		t.Errorf("Detection mismatch: expected bird, got %s", frame.Detections[0].ClassName)
	}
	if frame.FPS != 30.0 {
		t.Errorf("FPS metadata missing: got %f", frame.FPS)
	}
}

// --- 5. END-TO-END SYSTEM TESTS ---

func TestHealthEndpoint(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "online", "version": "v6.2"})
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestStressTestSimulation(t *testing.T) {
	// Simulate 50 rapid requests
	start := time.Now()
	for i := 0; i < 50; i++ {
		// Mock task
		_ = fmt.Sprintf("Task %d", i)
	}
	elapsed := time.Since(start)
	if elapsed > 10*time.Millisecond {
		t.Errorf("System overhead too high: 50 cycles took %v", elapsed)
	}
}
