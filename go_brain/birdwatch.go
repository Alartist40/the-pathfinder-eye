package main

import (
	"database/sql"
	"math"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// BirdSighting represents a single bird observation
type BirdSighting struct {
	ID         string    `json:"id"`
	Species    string    `json:"species"`
	Confidence float32   `json:"confidence"`
	Timestamp  time.Time `json:"timestamp"`
	Location   string    `json:"location"`
	BBoxX      int       `json:"bbox_x"`
	BBoxY      int       `json:"bbox_y"`
	BBoxW      int       `json:"bbox_w"`
	BBoxH      int       `json:"bbox_h"`
	Notes      string    `json:"notes"`
	UserRank   string    `json:"user_rank"`
	SessionID  string    `json:"session_id"`
}

// BirdTrack represents a tracked bird across multiple frames
type BirdTrack struct {
	TrackID       int64     `json:"track_id"`
	LastDetected  time.Time `json:"last_detected"`
	Species       string    `json:"species"`
	Confidence    float32   `json:"confidence"`
	CentroidX     int       `json:"centroid_x"`
	CentroidY     int       `json:"centroid_y"`
	X             int       `json:"x"`
	Y             int       `json:"y"`
	W             int       `json:"w"`
	H             int       `json:"h"`
	FrameCount    int       `json:"frame_count"`
	TotalDistance int       `json:"total_distance"`
	Status        string    `json:"status"` // "active" or "lost"
}

type BirdWatchDB struct {
	db           *sql.DB
	activeTracks map[int64]*BirdTrack
	mu           sync.Mutex
}

func initBirdWatchDB(dbPath string) (*BirdWatchDB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal=WAL")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS bird_sightings (
        id          TEXT PRIMARY KEY,
        species     TEXT NOT NULL,
        confidence  REAL NOT NULL,
        timestamp   INTEGER NOT NULL,
        location    TEXT,
        bbox_x      INTEGER,
        bbox_y      INTEGER,
        bbox_w      INTEGER,
        bbox_h      INTEGER,
        notes       TEXT,
        user_rank   TEXT,
        session_id  TEXT
    );
    
    CREATE TABLE IF NOT EXISTS bird_tracks (
        track_id    INTEGER PRIMARY KEY,
        species     TEXT,
        confidence  REAL,
        centroid_x  INTEGER,
        centroid_y  INTEGER,
        frame_count INTEGER,
        total_dist  INTEGER,
        last_seen   INTEGER,
        status      TEXT
    );
    
    CREATE INDEX IF NOT EXISTS idx_sightings_timestamp ON bird_sightings(timestamp);
    CREATE INDEX IF NOT EXISTS idx_sightings_species ON bird_sightings(species);
    CREATE INDEX IF NOT EXISTS idx_tracks_status ON bird_tracks(status);
    `)
	if err != nil {
		return nil, err
	}

	return &BirdWatchDB{
		db:           db,
		activeTracks: make(map[int64]*BirdTrack),
	}, nil
}

func (bdb *BirdWatchDB) RecordSighting(sighting *BirdSighting) error {
	_, err := bdb.db.Exec(`
    INSERT INTO bird_sightings 
    (id, species, confidence, timestamp, location, bbox_x, bbox_y, bbox_w, bbox_h, notes, user_rank, session_id)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, sighting.ID, sighting.Species, sighting.Confidence, sighting.Timestamp.Unix(),
		sighting.Location, sighting.BBoxX, sighting.BBoxY, sighting.BBoxW, sighting.BBoxH,
		sighting.Notes, sighting.UserRank, sighting.SessionID)
	return err
}

func (bdb *BirdWatchDB) GetSpeciesList() ([]string, error) {
	rows, err := bdb.db.Query("SELECT DISTINCT species FROM bird_sightings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var species []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			continue
		}
		species = append(species, s)
	}
	return species, nil
}

// Detection struct removed, now in vision.go

type BirdTracker struct {
	nextTrackID   int64
	distThreshold int
	maxMissedAge  int
}

func NewBirdTracker() *BirdTracker {
	return &BirdTracker{
		nextTrackID:   1,
		distThreshold: 100,
		maxMissedAge:  30,
	}
}

func (bt *BirdTracker) UpdateTracks(detections []Detection, bdb *BirdWatchDB) []BirdTrack {
	bdb.mu.Lock()
	defer bdb.mu.Unlock()

	matchedDetections := make(map[int]bool)
	var activeTracks []BirdTrack

	for _, track := range bdb.activeTracks {
		bestMatch := -1
		bestDistance := bt.distThreshold

		for i, det := range detections {
			if matchedDetections[i] {
				continue
			}
			centroidX := det.X + det.W/2
			centroidY := det.Y + det.H/2

			dx := float32(centroidX - track.CentroidX)
			dy := float32(centroidY - track.CentroidY)
			dist := int(math.Sqrt(float64(dx*dx + dy*dy)))

			if dist < bestDistance {
				bestDistance = dist
				bestMatch = i
			}
		}

		if bestMatch >= 0 {
			det := detections[bestMatch]
			track.X = det.X
			track.Y = det.Y
			track.W = det.W
			track.H = det.H
			track.CentroidX = det.X + det.W/2
			track.CentroidY = det.Y + det.H/2
			track.Species = det.Species
			track.Confidence = det.Confidence
			track.LastDetected = time.Now()
			track.FrameCount++
			track.Status = "active"
			track.TotalDistance += bestDistance
			matchedDetections[bestMatch] = true
		} else {
			track.Status = "lost"
		}
		activeTracks = append(activeTracks, *track)
	}

	for i, det := range detections {
		if !matchedDetections[i] {
			trackID := bt.nextTrackID
			bt.nextTrackID++
			newTrack := &BirdTrack{
				TrackID:      trackID,
				Species:      det.Species,
				Confidence:   det.Confidence,
				X:            det.X,
				Y:            det.Y,
				W:            det.W,
				H:            det.H,
				CentroidX:    det.X + det.W/2,
				CentroidY:    det.Y + det.H/2,
				LastDetected: time.Now(),
				FrameCount:   1,
				Status:       "active",
			}
			bdb.activeTracks[trackID] = newTrack
			activeTracks = append(activeTracks, *newTrack)
		}
	}

	for _, track := range bdb.activeTracks {
		if track.Status == "lost" && time.Since(track.LastDetected) > time.Duration(bt.maxMissedAge)*33*time.Millisecond {
			delete(bdb.activeTracks, track.TrackID)
		}
	}
	return activeTracks
}
