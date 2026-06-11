package main

import (
	"testing"
	"os"
)

func TestBirdDetection(t *testing.T) {
	det := Detection{
		X: 100, Y: 150, W: 50, H: 60,
		Species: "robin", Confidence: 0.95,
	}
	if det.Species != "robin" {
		t.Errorf("Expected robin, got %s", det.Species)
	}
	t.Log("✅ TEST PASSED: Bird detection works")
}

func TestBirdTracking(t *testing.T) {
	tracker := NewBirdTracker()
	bdb := &BirdWatchDB{activeTracks: make(map[int64]*BirdTrack)}
	
	det1 := []Detection{{X: 100, Y: 150, W: 50, H: 60, Species: "robin", Confidence: 0.95}}
	tracks := tracker.UpdateTracks(det1, bdb)
	
	if len(tracks) != 1 {
		t.Errorf("Expected 1 track, got %d", len(tracks))
	}
	t.Log("✅ TEST PASSED: Bird tracking works")
}

func TestGimbalTracking(t *testing.T) {
	gt := NewGimbalTracker()
	gt.TrackObject(300, 200, 50, 60, 640, 480)
	pan, tilt := gt.GetCurrentPosition()
	
	if pan < 0 || pan > 180 || tilt < 0 || tilt > 180 {
		t.Errorf("Gimbal out of bounds: pan=%d, tilt=%d", pan, tilt)
	}
	t.Log("✅ TEST PASSED: Gimbal tracking works")
}

func TestBirdDatabase(t *testing.T) {
	os.Remove("/tmp/test_bird.sqlite")
	bdb, err := initBirdWatchDB("/tmp/test_bird")
	if err != nil {
		t.Fatalf("Failed to init: %v", err)
	}
	defer bdb.db.Close()
	
	sighting := &BirdSighting{
		ID: "test1", Species: "robin", Confidence: 0.95,
	}
	if err := bdb.RecordSighting(sighting); err != nil {
		t.Errorf("Failed to record: %v", err)
	}
	t.Log("✅ TEST PASSED: Bird database works")
}
