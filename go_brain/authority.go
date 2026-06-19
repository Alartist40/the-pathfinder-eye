package main

import (
	"strings"
)

type AuthorityLevel int

const (
	LevelGuest       AuthorityLevel = 0
	LevelScout       AuthorityLevel = 1
	LevelMasterguide AuthorityLevel = 2
	LevelLeader      AuthorityLevel = 3
	LevelProgrammer  AuthorityLevel = 4 // HIGHEST - Administrator
)

type AuthorizedFigure struct {
	Name   string
	Title  string
	Level  AuthorityLevel
	FaceID string
}

type AuthorityRank struct {
	Name    string
	Level   AuthorityLevel
	Color   string
	Program string
	Age     string
}

var allRanks = map[string]AuthorityRank{
	"programmer":          {Name: "Programmer", Level: LevelProgrammer, Color: "#FF0000", Program: "System", Age: "N/A"},
	"master_guide_senior": {Name: "Master Guide Senior", Level: LevelLeader, Color: "#FFD700", Program: "Pathfinder", Age: "22+"},
	"master_guide":        {Name: "Master Guide", Level: LevelLeader, Color: "#FFD700", Program: "Pathfinder", Age: "15+"},
	"guide":               {Name: "Guide", Level: LevelMasterguide, Color: "#FFD700", Program: "Pathfinder", Age: "15-16"},
	"voyager":             {Name: "Voyager", Level: LevelScout, Color: "#800020", Program: "Pathfinder", Age: "14-15"},
	"ranger":              {Name: "Ranger", Level: LevelScout, Color: "#C0C0C0", Program: "Pathfinder", Age: "13-14"},
	"explorer":            {Name: "Explorer", Level: LevelScout, Color: "#00AA00", Program: "Pathfinder", Age: "12-13"},
	"companion":           {Name: "Companion", Level: LevelScout, Color: "#FF0000", Program: "Pathfinder", Age: "11-12"},
	"friend":              {Name: "Friend", Level: LevelScout, Color: "#0000FF", Program: "Pathfinder", Age: "10-11"},
	"helping_hand":        {Name: "Helping Hand", Level: LevelGuest, Color: "#FFA500", Program: "Adventurer", Age: "6-9"},
	"builder":             {Name: "Builder", Level: LevelGuest, Color: "#FFA500", Program: "Adventurer", Age: "6-9"},
	"sunbeam":             {Name: "Sunbeam", Level: LevelGuest, Color: "#FFD700", Program: "Adventurer", Age: "6-9"},
	"busy_bee":            {Name: "Busy Bee", Level: LevelGuest, Color: "#FFD700", Program: "Adventurer", Age: "6-9"},
	"early_birds":         {Name: "Early Birds", Level: LevelGuest, Color: "#87CEEB", Program: "Adventurer", Age: "6-9"},
	"little_lamb":         {Name: "Little Lamb", Level: LevelGuest, Color: "#FFFFFF", Program: "Adventurer", Age: "6-9"},
}

type AuthorityManager struct {
	dendrite *Dendrite
}

func initAuthority(d *Dendrite) (*AuthorityManager, error) {
	am := &AuthorityManager{dendrite: d}
	if _, ok := d.Get("leader_list"); !ok {
		d.Upsert("leader_list", "Registered Leaders", "System bootstrap initiated. [[admin_01]] is the first [[Programmer]].", NodeTypeIdentity, []string{"security"})
	}
	return am, nil
}

func (am *AuthorityManager) VerifyFigure(faceID string) (AuthorizedFigure, bool) {
	node, ok := am.dendrite.Get(faceID)
	if !ok {
		return AuthorizedFigure{}, false
	}

	content := strings.ToLower(node.Content)
	if strings.Contains(content, "[[programmer]]") {
		return AuthorizedFigure{Name: node.Title, Title: "Programmer", Level: LevelProgrammer, FaceID: faceID}, true
	}
	if strings.Contains(content, "[[leader]]") {
		return AuthorizedFigure{Name: node.Title, Title: "Leader", Level: LevelLeader, FaceID: faceID}, true
	}
	if strings.Contains(content, "[[masterguide]]") {
		return AuthorizedFigure{Name: node.Title, Title: "Masterguide", Level: LevelMasterguide, FaceID: faceID}, true
	}
	if strings.Contains(content, "[[scout]]") {
		return AuthorizedFigure{Name: node.Title, Title: "Scout", Level: LevelScout, FaceID: faceID}, true
	}
	return AuthorizedFigure{Name: node.Title, Title: "Guest", Level: LevelGuest, FaceID: faceID}, true
}

func (am *AuthorityManager) CanExecuteCommand(userLevel AuthorityLevel, cmd string) bool {
	restrictedPrefixes := []string{"sleep", "calibrate", "delete", "shutdown", "reboot"}
	for _, prefix := range restrictedPrefixes {
		if strings.HasPrefix(strings.ToLower(cmd), prefix) {
			if userLevel < LevelMasterguide {
				_ = speak("Access denied. Insufficient authority.")
				return false
			}
		}
	}
	return true
}

func (am *AuthorityManager) GetRankByID(rankID string) (AuthorityRank, bool) {
	rank, ok := allRanks[rankID]
	return rank, ok
}
