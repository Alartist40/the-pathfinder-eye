/**
 * THE-PATHFINDER-EYE : Context & Memory System (Phase 3)
 * Version: 1.0 (SQLite Persistence)
 */

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type ContextDB struct {
	db *sql.DB
}

type MemoryEntry struct {
	Key        string      `json:"key"`
	Value      interface{} `json:"value"`
	Importance int         `json:"importance"`
}

func initContextDB() (*ContextDB, error) {
	path := "/home/pi/the-pathfinder-eye_ai/db/context.sqlite"
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY,
		type TEXT,
		description TEXT,
		data JSON,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS conversations (
		id INTEGER PRIMARY KEY,
		role TEXT,
		content TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS memory (
		key TEXT PRIMARY KEY,
		value JSON,
		importance INTEGER,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return &ContextDB{db: db}, nil
}

func (c *ContextDB) RecordEvent(typ, desc string, data interface{}) {
	d, _ := json.Marshal(data)
	_, _ = c.db.Exec("INSERT INTO events (type, description, data) VALUES (?, ?, ?)", typ, desc, string(d))
}

func (c *ContextDB) RecordTurn(role, content string) {
	_, _ = c.db.Exec("INSERT INTO conversations (role, content) VALUES (?, ?)", role, content)
}

func (c *ContextDB) StoreFact(key string, val interface{}, importance int) {
	d, _ := json.Marshal(val)
	_, _ = c.db.Exec("INSERT OR REPLACE INTO memory (key, value, importance) VALUES (?, ?, ?)", key, string(d), importance)
}

func (c *ContextDB) PruneOldData(hours int) error {
	infoLog.Printf("DB_OPTIMIZE: Pruning events older than %d hours...", hours)
	_, err := c.db.Exec("DELETE FROM events WHERE created_at < datetime('now', '-' || ? || ' hours')", hours)
	if err != nil {
		return err
	}

	// Reclaim disk space
	_, err = c.db.Exec("VACUUM")
	return err
}

func (c *ContextDB) BuildPromptContext() string {
	// Fetch top 5 important memories
	rows, _ := c.db.Query("SELECT key, value FROM memory ORDER BY importance DESC LIMIT 5")
	defer rows.Close()

	var context strings.Builder
	context.WriteString("RELEVANT MEMORIES:\n")
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		context.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
	}

	// Fetch last 3 conversation turns
	history, _ := c.db.Query("SELECT role, content FROM conversations ORDER BY created_at DESC LIMIT 3")
	defer history.Close()

	context.WriteString("\nRECENT HISTORY:\n")
	for history.Next() {
		var r, c string
		history.Scan(&r, &c)
		context.WriteString(fmt.Sprintf("%s: %s\n", r, c))
	}

	return context.String()
}
