/**
 * THE-PATHFINDER-EYE : Neural-Link Memory Engine (Phase 7)
 * Based on Cynapse Dendrite Architecture
 */

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type NodeType string

const (
	NodeTypeIdentity NodeType = "identity"
	NodeTypePerson   NodeType = "person"
	NodeTypeConcept  NodeType = "concept"
	NodeTypeEvent    NodeType = "event"
	NodeTypeCustom   NodeType = "custom"
)

type Node struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Type      NodeType `json:"type"`
	Tags      []string `json:"tags"`
	Links     []string `json:"links"`
	Backlinks []string `json:"backlinks"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

type Dendrite struct {
	db          *sql.DB
	nodes       map[string]*Node
	mu          sync.RWMutex
	linkPattern *regexp.Regexp
	tagPattern  *regexp.Regexp
}

func initDendrite() (*Dendrite, error) {
	return initDendritePath("../db/dendrite.sqlite")
}

func initDendritePath(dbPath string) (*Dendrite, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal=WAL")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS dendrite_nodes (
        id         TEXT PRIMARY KEY,
        title      TEXT NOT NULL,
        content    TEXT NOT NULL,
        type       TEXT NOT NULL,
        tags       TEXT NOT NULL,
        links      TEXT NOT NULL,
        backlinks  TEXT NOT NULL,
        created_at INTEGER NOT NULL,
        updated_at INTEGER NOT NULL
    );
    `)
	if err != nil {
		return nil, err
	}

	d := &Dendrite{
		db:          db,
		nodes:       make(map[string]*Node),
		linkPattern: regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`),
		tagPattern:  regexp.MustCompile(`#([A-Za-z0-9_-]+)`),
	}

	if err := d.loadAll(); err != nil {
		return nil, err
	}

	if _, ok := d.Get("identity"); !ok {
		d.Upsert("identity", "The Pathfinder Eye", "I am an autonomous wilderness robot designed to assist and guide.", NodeTypeIdentity, []string{"core"})
	}

	return d, nil
}

func (d *Dendrite) loadAll() error {
	rows, err := d.db.Query(`SELECT id, title, content, type, tags, links, backlinks, created_at, updated_at FROM dendrite_nodes`)
	if err != nil {
		return err
	}
	defer rows.Close()

	d.mu.Lock()
	defer d.mu.Unlock()

	for rows.Next() {
		n := &Node{}
		var t, tg, l, bl string
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &t, &tg, &l, &bl, &n.CreatedAt, &n.UpdatedAt); err == nil {
			n.Type = NodeType(t)
			json.Unmarshal([]byte(tg), &n.Tags)
			json.Unmarshal([]byte(l), &n.Links)
			json.Unmarshal([]byte(bl), &n.Backlinks)
			d.nodes[n.ID] = n
		}
	}
	return nil
}

func (d *Dendrite) Upsert(id, title, content string, nodeType NodeType, tags []string) *Node {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().Unix()
	links := d.parseLinks(content)
	if len(tags) == 0 {
		tags = d.parseTags(content)
	}

	if old, ok := d.nodes[id]; ok {
		for _, oldLink := range old.Links {
			if target, ok := d.nodes[oldLink]; ok {
				target.Backlinks = removeStr(target.Backlinks, id)
			}
		}
	}

	node, exists := d.nodes[id]
	if !exists {
		node = &Node{ID: id, CreatedAt: now}
		d.nodes[id] = node
	}

	node.Title = title
	node.Content = content
	node.Type = nodeType
	node.Tags = tags
	node.Links = links
	node.UpdatedAt = now

	for _, link := range links {
		target, exists := d.nodes[link]
		if !exists {
			target = &Node{ID: link, Title: link, Type: NodeTypeCustom, CreatedAt: now, UpdatedAt: now}
			d.nodes[link] = target
		}
		if !containsStr(target.Backlinks, id) {
			target.Backlinks = append(target.Backlinks, id)
		}
	}

	tg, _ := json.Marshal(node.Tags)
	l, _ := json.Marshal(node.Links)
	bl, _ := json.Marshal(node.Backlinks)

	d.db.Exec(`INSERT OR REPLACE INTO dendrite_nodes (id, title, content, type, tags, links, backlinks, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		node.ID, node.Title, node.Content, string(node.Type), string(tg), string(l), string(bl), node.CreatedAt, node.UpdatedAt)

	return node
}

func (d *Dendrite) Get(id string) (*Node, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	n, ok := d.nodes[id]
	return n, ok
}

func (d *Dendrite) BuildPromptContext(query string) string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var parts []string
	if core, ok := d.nodes["identity"]; ok {
		parts = append(parts, fmt.Sprintf("## %s\n%s", core.Title, core.Content))
	}

	q := strings.ToLower(query)
	var relevant []*Node

	for _, n := range d.nodes {
		if n.ID == "identity" {
			continue
		}
		score := 0
		if strings.Contains(strings.ToLower(n.Title), q) {
			score += 10
		}
		if strings.Contains(strings.ToLower(n.Content), q) {
			score += 5
		}
		if score > 0 {
			relevant = append(relevant, n)
		}
	}

	sort.Slice(relevant, func(i, j int) bool {
		return relevant[i].UpdatedAt > relevant[j].UpdatedAt
	})

	for i, n := range relevant {
		if i >= 3 {
			break
		}
		parts = append(parts, fmt.Sprintf("## %s\n%s", n.Title, n.Content))
	}

	return strings.Join(parts, "\n\n")
}

func (d *Dendrite) parseLinks(content string) []string {
	matches := d.linkPattern.FindAllStringSubmatch(content, -1)
	var links []string
	for _, m := range matches {
		if len(m) > 1 {
			links = append(links, toNodeID(m[1]))
		}
	}
	return links
}

func (d *Dendrite) parseTags(content string) []string {
	matches := d.tagPattern.FindAllStringSubmatch(content, -1)
	var tags []string
	for _, m := range matches {
		if len(m) > 1 {
			tags = append(tags, m[1])
		}
	}
	return tags
}

func toNodeID(s string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), " ", "_")
}

func (d *Dendrite) Close() {
	if d != nil && d.db != nil {
		d.db.Close()
	}
}

func containsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeStr(slice []string, item string) []string {
	out := slice[:0]
	for _, s := range slice {
		if s != item {
			out = append(out, s)
		}
	}
	return out
}
