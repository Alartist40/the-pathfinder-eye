package main

import (
	"os"
	"testing"
	"time"
)

// Setup and teardown
func setupTestDendrite(t *testing.T) *Dendrite {
	os.Remove("/tmp/test_dendrite.sqlite")
	
	// Mock database for testing
	d, err := initDendrite()
	if err != nil {
		t.Fatalf("Failed to initialize Dendrite: %v", err)
	}
	return d
}

func teardownTestDendrite(d *Dendrite) {
	d.db.Close()
	os.Remove("/tmp/test_dendrite.sqlite")
}

// TEST 1: Node Creation
func TestDendriteNodeCreation(t *testing.T) {
	d := setupTestDendrite(t)
	defer teardownTestDendrite(d)
	
	// Create a node
	node := d.Upsert("alex", "Alex", "Scout named Alex", NodeTypePerson, []string{"scout"})
	
	if node == nil {
		t.Fatal("Node creation failed")
	}
	if node.ID != "alex" {
		t.Errorf("Node ID mismatch: expected 'alex', got '%s'", node.ID)
	}
	if node.Title != "Alex" {
		t.Errorf("Node title mismatch: expected 'Alex', got '%s'", node.Title)
	}
	if node.Type != NodeTypePerson {
		t.Errorf("Node type mismatch: expected NodeTypePerson, got %s", node.Type)
	}
	
	t.Log("✅ TEST PASSED: Node creation works correctly")
}

// TEST 2: Node Retrieval
func TestDendriteNodeRetrieval(t *testing.T) {
	d := setupTestDendrite(t)
	defer teardownTestDendrite(d)
	
	// Create node
	d.Upsert("john", "John", "Leader named John", NodeTypeIdentity, []string{"leader"})
	
	// Retrieve node
	retrieved, ok := d.Get("john")
	if !ok {
		t.Fatal("Node retrieval failed")
	}
	if retrieved.Title != "John" {
		t.Errorf("Retrieved node title mismatch: expected 'John', got '%s'", retrieved.Title)
	}
	
	t.Log("✅ TEST PASSED: Node retrieval works correctly")
}

// TEST 3: Link Parsing
func TestDendriteLinkParsing(t *testing.T) {
	d := setupTestDendrite(t)
	defer teardownTestDendrite(d)
	
	// Create node with wiki-style links
	content := "This is [[John]] who is a [[Leader]] at [[Pine Trail]]"
	d.Upsert("test", "Test", content, NodeTypeEvent, []string{})
	
	// Retrieve and check links
	node, _ := d.Get("test")
	if len(node.Links) != 3 {
		t.Errorf("Expected 3 links, got %d", len(node.Links))
	}
	
	expectedLinks := []string{"john", "leader", "pine_trail"}
	for i, link := range node.Links {
		if i < len(expectedLinks) && link != expectedLinks[i] {
			t.Errorf("Link mismatch at index %d: expected '%s', got '%s'", i, expectedLinks[i], link)
		}
	}
	
	t.Log("✅ TEST PASSED: Link parsing works correctly")
}

// TEST 4: Tag Parsing
func TestDendriteTagParsing(t *testing.T) {
	d := setupTestDendrite(t)
	defer teardownTestDendrite(d)
	
	// Create node with tags
	content := "This is a #safety incident at #pine_trail involving #lost_scout"
	d.Upsert("incident1", "Lost Scout", content, NodeTypeEvent, []string{})
	
	// Retrieve and check tags
	node, _ := d.Get("incident1")
	if len(node.Tags) < 1 {
		t.Errorf("Expected tags, got %d", len(node.Tags))
	}
	
	t.Log("✅ TEST PASSED: Tag parsing works correctly")
}

// TEST 5: Bidirectional Links (Backlinks)
func TestDendriteBidirectionalLinks(t *testing.T) {
	d := setupTestDendrite(t)
	defer teardownTestDendrite(d)
	
	// Create first node
	d.Upsert("node1", "Node One", "Contains [[node2]]", NodeTypeConcept, []string{})
	
	// Create second node
	d.Upsert("node2", "Node Two", "Referenced by node1", NodeTypeConcept, []string{})
	
	// Check that node2 has backlink to node1
	node2, _ := d.Get("node2")
	if len(node2.Backlinks) != 1 {
		t.Errorf("Expected 1 backlink, got %d", len(node2.Backlinks))
	}
	
	t.Log("✅ TEST PASSED: Bidirectional links work correctly")
}

// TEST 6: Update Timestamp
func TestDendriteUpdateTimestamp(t *testing.T) {
	d := setupTestDendrite(t)
	defer teardownTestDendrite(d)
	
	// Create node
	node1 := d.Upsert("test", "Test", "Original content", NodeTypeConcept, []string{})
	time1 := node1.UpdatedAt
	
	// Wait and update
	time.Sleep(1100 * time.Millisecond)
	node2 := d.Upsert("test", "Test", "Updated content", NodeTypeConcept, []string{})
	time2 := node2.UpdatedAt
	
	if time2 <= time1 {
		t.Errorf("Update timestamp not changed: %d vs %d", time1, time2)
	}
	
	t.Log("✅ TEST PASSED: Update timestamp works correctly")
}

// TEST 7: Persistence (SQLite)
func TestDendritePersistence(t *testing.T) {
	// Create first instance
	d1 := setupTestDendrite(t)
	d1.Upsert("persistent", "Persistent Node", "Should survive reboot", NodeTypePerson, []string{"test"})
	d1.db.Close()
	
	// Create second instance (simulates reboot)
	d2, _ := initDendrite()
	defer teardownTestDendrite(d2)
	
	// Check if node still exists
	retrieved, ok := d2.Get("persistent")
	if !ok {
		t.Fatal("Node not persisted to database")
	}
	if retrieved.Title != "Persistent Node" {
		t.Errorf("Persisted node data mismatch")
	}
	
	t.Log("✅ TEST PASSED: Persistence works correctly")
}

// TEST 8: Context Building
func TestDendriteContextBuilding(t *testing.T) {
	d := setupTestDendrite(t)
	defer teardownTestDendrite(d)
	
	// Create some nodes
	d.Upsert("alex", "Alex", "A scout with #hiking_experience", NodeTypePerson, []string{"scout"})
	d.Upsert("hiking", "Hiking", "[[Alex]] loves [[hiking]]", NodeTypeConcept, []string{"activity"})
	
	// Build context
	context := d.BuildPromptContext("hiking")
	
	if context == "" {
		t.Fatal("Context building returned empty string")
	}
	if len(context) < 10 {
		t.Errorf("Context too short: %s", context)
	}
	
	t.Log("✅ TEST PASSED: Context building works correctly")
}
