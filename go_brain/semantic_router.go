/**
 * THE-PATHFINDER-EYE : Semantic Router v1.0
 * Pure Go implementation - no Python at runtime
 *
 * Uses pre-computed embeddings for ultra-fast route classification.
 * Embeddings are computed once at build time, stored as constants.
 *
 * Build-time computation: Run scripts/compute_embeddings.go once to regenerate
 * the centroid constants below.
 */

package main

import (
	"math"
	"strings"
)

// Route classification for robot commands
type Route int

const (
	RouteUnknown   Route = iota
	RouteBasicChat       // Greetings, simple facts, chitchat
	RouteThinking        // Complex reasoning, analysis, "why", "explain"
	RouteToolCall        // Motor control, sensors, hardware commands
)

func (r Route) String() string {
	switch r {
	case RouteBasicChat:
		return "basic_chat"
	case RouteThinking:
		return "thinking"
	case RouteToolCall:
		return "tool_call"
	default:
		return "unknown"
	}
}

// Embedding is a normalized dense vector (pre-computed at build time)
type Embedding []float32

// RouteConfig defines a routing category with pre-computed centroid
type RouteConfig struct {
	Route    Route
	Centroid Embedding // average embedding for this route
}

// cosineSimilarity computes similarity between two normalized vectors
func cosineSimilarity(a, b Embedding) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return dot
}

// robotWordBoost gives extra weight to known robot command words
// allowing the embedding to better distinguish tool_call from chitchat.
var robotWordBoost = map[string]float32{
	"birdwatch": 3.0, "third": 3.0, "bired": 3.0, "beard": 3.0, "word": 3.0,
	"pathfinder": 2.0, "adventurer": 2.0,
	"motor": 2.5, "servo": 2.5, "move": 2.0, "turn": 2.0, "left": 1.5, "right": 1.5,
	"forward": 2.0, "back": 2.0, "backward": 2.0,
	"look": 2.5, "scan": 2.0, "gaze": 2.0, "pan": 2.0, "tilt": 2.0,
	"activate": 2.0, "enable": 2.0, "start": 2.0, "mode": 1.5,
	"security": 2.5, "camera": 2.0, "pir": 2.0, "cameras": 2.0,
	"follow": 2.5, "track": 2.0, "tracking": 2.0,
	"song": 2.0, "music": 2.0, "play": 1.5,
	"law": 2.0, "pledge": 2.0, "aim": 2.0, "motto": 2.0, "read": 2.0,
	"translate": 2.5, "japanese": 2.5, "translation": 2.0,
	"test": 2.0, "diagnostic": 2.0, "check": 1.5,
	"remote": 2.5, "control": 1.5,
	"deep": 2.0, "thinking": 1.5, "thought": 1.5,
	"attention": 2.0, "awaken": 2.0, "wake": 2.0,
	"about": 1.5, "exit": 1.5, "stop": 1.5, "sleep": 1.5,
}

// sentenceEmbedding computes a fast embedding for a sentence using
// word frequency + position weighting + robot word boost.
func sentenceEmbedding(text string, vocabSize int) Embedding {
	words := strings.Fields(strings.ToLower(text))
	n := len(words)
	if n == 0 {
		return make(Embedding, vocabSize)
	}

	vec := make(Embedding, vocabSize)
	for i, word := range words {
		// Position weight: words earlier in sentence get higher weight
		posWeight := 1.0 - (float64(i) / float64(n))

		// Boost for known robot command words (default 1.0)
		boost := 1.0
		if b := robotWordBoost[word]; b > 0 {
			boost = float64(b)
		}

		// Hash word to two vector dimensions
		h := hashString(word)
		dim1 := h % vocabSize
		dim2 := (h >> 8) % vocabSize

		// Add word contribution with position weighting and boost
		vec[dim1] += float32(0.7 * posWeight * boost)
		vec[dim2] += float32(0.3 * posWeight * boost)
	}

	normalize(vec)
	return vec
}

// hashString is a fast string hash
func hashString(s string) int {
	h := 5381
	for _, c := range s {
		h = ((h << 5) + h) + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// normalize scales vector to unit length (L2)
func normalize(v Embedding) {
	var mag float64
	for _, x := range v {
		mag += float64(x) * float64(x)
	}
	if mag == 0 {
		return
	}
	mag = math.Sqrt(mag)
	for i := range v {
		v[i] = float32(float64(v[i]) / mag)
	}
}

// RouteClassifier performs semantic routing using pre-computed centroids
type RouteClassifier struct {
	routes   []RouteConfig
	fallback Route
}

// Classify determines the route for a given text prompt
// Returns the matched route and confidence score (0-1)
func (c *RouteClassifier) Classify(prompt string) (Route, float64) {
	if len(strings.TrimSpace(prompt)) == 0 {
		return c.fallback, 0
	}

	emb := sentenceEmbedding(prompt, 256)

	bestRoute := c.fallback
	bestScore := 0.0

	for _, r := range c.routes {
		score := cosineSimilarity(emb, r.Centroid)
		if score > bestScore {
			bestScore = score
			bestRoute = r.Route
		}
	}

	return bestRoute, bestScore
}

// Pre-computed route centroids (dimensionality 256)
// Generated from sample utterances for each route category
//
// BASIC_CHAT_UTTERANCES: "hi hello hey good morning good night thanks thank
// you bye ok sure cool nice to meet you what's up how are you"
//
// THINKING_UTTERANCES: "why explain how would you compare analyze what are
// the pros and cons walk me through the logic what causes summarize"
//
// TOOL_CALL_UTTERANCES: "move forward turn left look up activate security
// scan network search web weather stock camera led motor"
var basicChatCentroid = Embedding{
	0.000, 0.000, 0.000, 0.000, 0.099, 0.000, 0.000, 0.002,
	0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.124, 0.000,
	0.000, 0.000, 0.000, 0.078, 0.000, 0.000, 0.191, 0.036,
	0.000, 0.061, 0.000, 0.000, 0.000, 0.000, 0.000, 0.045,
	0.000, 0.064, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.000, 0.042, 0.000, 0.000,
	0.137, 0.042, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.120, 0.000, 0.000, 0.000, 0.044,
	0.011, 0.000, 0.000, 0.046, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.000, 0.110, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.000, 0.026, 0.032, 0.127,
	0.000, 0.000, 0.276, 0.048, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.038, 0.000, 0.000, 0.000, 0.000, 0.615, 0.000,
	0.000, 0.264, 0.000, 0.000, 0.000, 0.053, 0.000, 0.000,
	0.129, 0.076, 0.138, 0.000, 0.033, 0.000, 0.000, 0.007,
	0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.085, 0.000, 0.053, 0.039,
	0.000, 0.141, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.163, 0.000, 0.000, 0.000, 0.145, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.095, 0.000, 0.000, 0.024, 0.000, 0.000, 0.000, 0.000,
	0.004, 0.134, 0.000, 0.000, 0.000, 0.023, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.000, 0.064, 0.000, 0.148,
	0.000, 0.000, 0.000, 0.000, 0.118, 0.113, 0.000, 0.000,
	0.000, 0.000, 0.081, 0.212, 0.000, 0.112, 0.003, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.029, 0.000, 0.000, 0.000, 0.085, 0.055,
	0.000, 0.000, 0.000, 0.000, 0.106, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.011, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.092, 0.000, 0.103, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.110,
}

var thinkingCentroid = Embedding{
	0.000, 0.000, 0.058, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.000, 0.079, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.165, 0.000, 0.000, 0.000, 0.172,
	0.151, 0.031, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.046, 0.000, 0.016, 0.000, 0.495, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.162, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.043, 0.000, 0.066, 0.056,
	0.000, 0.045, 0.069, 0.086, 0.000, 0.000, 0.000, 0.001,
	0.000, 0.000, 0.082, 0.000, 0.000, 0.000, 0.003, 0.027,
	0.027, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.006, 0.151, 0.000, 0.000, 0.000, 0.065, 0.163, 0.000,
	0.000, 0.000, 0.247, 0.000, 0.000, 0.000, 0.000, 0.010,
	0.000, 0.000, 0.000, 0.000, 0.055, 0.143, 0.000, 0.034,
	0.069, 0.000, 0.000, 0.000, 0.000, 0.000, 0.046, 0.040,
	0.088, 0.025, 0.000, 0.000, 0.094, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.103, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.052, 0.000, 0.000, 0.007, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.127, 0.000, 0.000, 0.000,
	0.093, 0.000, 0.000, 0.000, 0.000, 0.144, 0.000, 0.029,
	0.000, 0.072, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.137, 0.000, 0.000, 0.065, 0.000, 0.128, 0.000,
	0.000, 0.000, 0.000, 0.113, 0.000, 0.000, 0.000, 0.155,
	0.000, 0.000, 0.072, 0.000, 0.007, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.106, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.000, 0.000, 0.028, 0.000, 0.000,
	0.014, 0.000, 0.000, 0.000, 0.062, 0.000, 0.000, 0.000,
	0.142, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.082, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.107, 0.000, 0.000, 0.155, 0.035, 0.000, 0.037,
	0.034, 0.000, 0.000, 0.000, 0.000, 0.000, 0.100, 0.000,
	0.049, 0.402, 0.000, 0.054, 0.000, 0.183, 0.000, 0.000,
}

var toolCallCentroid = Embedding{
	0.000, 0.000, 0.073, 0.073, 0.015, 0.031, 0.000, 0.000,
	0.000, 0.039, 0.190, 0.002, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.012, 0.000, 0.000, 0.000, 0.000, 0.038,
	0.000, 0.015, 0.000, 0.044, 0.006, 0.053, 0.038, 0.000,
	0.019, 0.039, 0.000, 0.000, 0.018, 0.000, 0.082, 0.000,
	0.027, 0.000, 0.102, 0.009, 0.049, 0.000, 0.000, 0.153,
	0.000, 0.059, 0.019, 0.000, 0.000, 0.000, 0.062, 0.106,
	0.000, 0.011, 0.024, 0.000, 0.017, 0.000, 0.000, 0.067,
	0.009, 0.134, 0.000, 0.029, 0.110, 0.000, 0.030, 0.048,
	0.052, 0.119, 0.226, 0.104, 0.154, 0.027, 0.201, 0.000,
	0.008, 0.035, 0.100, 0.000, 0.043, 0.069, 0.000, 0.000,
	0.013, 0.000, 0.378, 0.101, 0.017, 0.124, 0.041, 0.000,
	0.014, 0.012, 0.000, 0.165, 0.000, 0.266, 0.036, 0.000,
	0.081, 0.009, 0.000, 0.000, 0.031, 0.014, 0.000, 0.007,
	0.000, 0.000, 0.000, 0.121, 0.040, 0.000, 0.231, 0.061,
	0.029, 0.062, 0.004, 0.000, 0.075, 0.123, 0.029, 0.056,
	0.016, 0.002, 0.068, 0.113, 0.000, 0.068, 0.089, 0.010,
	0.045, 0.039, 0.054, 0.056, 0.014, 0.000, 0.039, 0.050,
	0.000, 0.000, 0.021, 0.000, 0.000, 0.000, 0.000, 0.000,
	0.012, 0.133, 0.037, 0.000, 0.170, 0.003, 0.013, 0.004,
	0.068, 0.035, 0.011, 0.047, 0.000, 0.000, 0.000, 0.022,
	0.008, 0.000, 0.000, 0.030, 0.146, 0.000, 0.040, 0.102,
	0.171, 0.000, 0.031, 0.033, 0.000, 0.033, 0.000, 0.028,
	0.000, 0.094, 0.159, 0.043, 0.052, 0.114, 0.000, 0.000,
	0.034, 0.017, 0.015, 0.072, 0.000, 0.000, 0.000, 0.000,
	0.045, 0.000, 0.097, 0.016, 0.000, 0.048, 0.000, 0.000,
	0.029, 0.048, 0.032, 0.078, 0.000, 0.000, 0.000, 0.000,
	0.000, 0.000, 0.000, 0.028, 0.072, 0.020, 0.008, 0.000,
	0.063, 0.009, 0.000, 0.000, 0.045, 0.000, 0.054, 0.058,
	0.012, 0.006, 0.000, 0.000, 0.026, 0.027, 0.003, 0.015,
	0.000, 0.000, 0.012, 0.000, 0.049, 0.000, 0.071, 0.006,
	0.110, 0.000, 0.009, 0.062, 0.000, 0.013, 0.000, 0.000,
}

// Default classifier instance
var defaultClassifier = &RouteClassifier{
	routes: []RouteConfig{
		{Route: RouteBasicChat, Centroid: basicChatCentroid},
		{Route: RouteThinking, Centroid: thinkingCentroid},
		{Route: RouteToolCall, Centroid: toolCallCentroid},
	},
	fallback: RouteBasicChat,
}

// ClassifyRoute is the main entry point for routing a voice/text prompt
// Returns the appropriate route and confidence score
func ClassifyRoute(prompt string) (Route, float64) {
	return defaultClassifier.Classify(prompt)
}

// ClassifyRouteSimple returns just the route (no confidence)
func ClassifyRouteSimple(prompt string) Route {
	route, _ := defaultClassifier.Classify(prompt)
	return route
}
