package main

import (
	"os"
	"testing"
)

func setupTestAuthority(t *testing.T) (*Dendrite, *AuthorityManager) {
	os.Remove("/tmp/test_authority.sqlite")
	d, _ := initDendrite()
	am, err := initAuthority(d)
	if err != nil {
		t.Fatalf("Failed to initialize Authority: %v", err)
	}
	return d, am
}

func teardownTestAuthority(d *Dendrite) {
	d.db.Close()
	os.Remove("/tmp/test_authority.sqlite")
}

// TEST 1: Authority Levels Are Distinct
func TestAuthorityLevels(t *testing.T) {
	levels := []AuthorityLevel{LevelGuest, LevelScout, LevelMasterguide, LevelLeader}
	for i := 0; i < len(levels)-1; i++ {
		if levels[i] >= levels[i+1] {
			t.Errorf("Authority levels not properly ordered: %d >= %d", levels[i], levels[i+1])
		}
	}
	t.Log("✅ TEST PASSED: Authority levels are properly ordered")
}

// TEST 2: Restricted Commands Are Blocked for Low Authority
func TestAuthorityCommandBlocking(t *testing.T) {
	d, am := setupTestAuthority(t)
	defer teardownTestAuthority(d)
	
	restrictedCommands := []string{"sleep", "calibrate", "delete", "shutdown"}
	
	for _, cmd := range restrictedCommands {
		// Scout (Level 1) should NOT be able to execute
		if am.CanExecuteCommand(LevelScout, cmd) {
			t.Errorf("Scout should not be able to execute: %s", cmd)
		}
		
		// Leader (Level 3) SHOULD be able to execute
		if !am.CanExecuteCommand(LevelLeader, cmd) {
			t.Errorf("Leader should be able to execute: %s", cmd)
		}
	}
	
	t.Log("✅ TEST PASSED: Command blocking works correctly")
}

// TEST 3: Face Verification
func TestAuthorityFaceVerification(t *testing.T) {
	d, am := setupTestAuthority(t)
	defer teardownTestAuthority(d)
	
	// Register a leader in Dendrite
	d.Upsert("john_smith", "John Smith", "[[Leader]] at camp", NodeTypePerson, []string{"authority", "leader"})
	
	// Verify face (should be recognized as leader)
	figure, ok := am.VerifyFigure("john_smith")
	if !ok {
		t.Fatal("Face verification failed for registered leader")
	}
	if figure.Level != LevelLeader {
		t.Errorf("Expected LevelLeader, got %d", figure.Level)
	}
	
	// Verify unknown face (should fail)
	_, ok = am.VerifyFigure("unknown_person")
	if ok {
		t.Fatal("Should not verify unknown face")
	}
	
	t.Log("✅ TEST PASSED: Face verification works correctly")
}

// TEST 4: Regular Commands Are Always Allowed
func TestAuthorityRegularCommands(t *testing.T) {
	d, am := setupTestAuthority(t)
	defer teardownTestAuthority(d)
	
	regularCommands := []string{"move forward", "tell me a joke", "what time is it"}
	
	for _, cmd := range regularCommands {
		// Even guests should be able to execute regular commands
		if !am.CanExecuteCommand(LevelGuest, cmd) {
			t.Errorf("Guest should be able to execute: %s", cmd)
		}
	}
	
	t.Log("✅ TEST PASSED: Regular commands work for all levels")
}
