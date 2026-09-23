// compressor.go — slim port of cynapse/internal/compressor for the
// robot's brain.
//
// Why we need it: the robot's AIConversationLoop accumulates user
// speech + assistant responses indefinitely. Hermes-3-8B has an
// 8K-token context; once a long conversation overflows that, the
// model starts producing truncated, hallucinated, or refused
// responses — and worst, the prompt grows unbounded until the brain
// process gets killed by OOM.
//
// The Cynapse compressor solves this by moving the middle of the
// transcript into DENDRITE (the brain's graph memory) as a memory
// node and dropping it from the active context. We do the same
// here, scoped to the robot's local Dendrite. We don't have
// multi-session persistence so we only need one method:
// MaybeCompress returns []message with the middle archived.
//
// All token accounting is intentionally cheap — runes / 4, plus a
// fixed per-message envelope overhead. No external tokenizer.
package main

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Defaults follow cynapse/internal/compressor but tuned for the
// Hermes-3-8B 8K context window. Bump ContextLength if the model
// gets swapped.
const (
	compactorDefaultCtxLen  = 8192 // Hermes-3-8B nominal context
	compactorThresholdPct   = 50   // compress at 50% of context
	compactorProtectHead    = 3    // first N messages kept verbatim
	compactorProtectTail    = 4    // last N messages kept verbatim
	compactorMinThreshold   = 1024 // floor in tokens
	compactorCharsPerToken  = 4    // ceiling-divided approximation
	compactorPerMsgOverhead = 10   // role + metadata overhead
	compactorTagSession     = "compaction"
)

// transcriptMsg is the smallest possible message that survives a
// compress pass. role/body is enough to rebuild the assistant prompt.
type transcriptMsg struct {
	Role string
	Body string
}

// transcriptCompactor is a stateful compressor for a single
// AIConversationLoop session. Liveness: the caller mutates the
// transcript (slice appends) under its own logic; MaybeCompress is
// called between user turns and returns a NEW slice.
type transcriptCompactor struct {
	ContextLength int
	Threshold     int
	ProtectHead   int
	ProtectTail   int
	// Archive is the persistence sink. The robot already drives
	// dendrite through initDendrite(); we thread the same instance
	// into the compactor.
	Archive *Dendrite
}

// newCompactor wires the compactor to the bot's actual Dendrite.
// When dendrite is nil (boot race, corruption, etc.) the compactor
// still works but silently drops archival, returning the original
// transcript unaltered.
func newCompactor(d *Dendrite) *transcriptCompactor {
	threshold := compactorDefaultCtxLen * compactorThresholdPct / 100
	if threshold < compactorMinThreshold {
		threshold = compactorMinThreshold
	}
	return &transcriptCompactor{
		ContextLength: compactorDefaultCtxLen,
		Threshold:     threshold,
		ProtectHead:   compactorProtectHead,
		ProtectTail:   compactorProtectTail,
		Archive:       d,
	}
}

// estimateTokens is the cheap token estimator. Same formula as
// cynapse/internal/compressor: chars/4 + per-msg overhead.
func estimateTokens(m transcriptMsg) int {
	runeCount := utf8.RuneCountInString(m.Body)
	toks := runeCount / compactorCharsPerToken
	if toks < 1 {
		toks = 1
	}
	return toks + compactorPerMsgOverhead
}

// totalTokens sums estimates over the transcript.
func (c *transcriptCompactor) totalTokens(msgs []transcriptMsg) int {
	total := 0
	for _, m := range msgs {
		total += estimateTokens(m)
	}
	return total
}

// MaybeCompress returns the transcript as-is if under the
// threshold. If over, the middle (everything between the protected
// head and the protected tail) is archived into DENDRITE as a
// memory node tagged "compaction", then dropped. The returned slice
// is the trimmed transcript and is safe to keep mutating.
func (c *transcriptCompactor) MaybeCompress(msgs []transcriptMsg) []transcriptMsg {
	if len(msgs) <= c.ProtectHead+c.ProtectTail {
		return msgs
	}
	if c.totalTokens(msgs) < c.Threshold {
		return msgs
	}

	head := msgs[:c.ProtectHead]
	tail := msgs[len(msgs)-c.ProtectTail:]
	middle := msgs[c.ProtectHead : len(msgs)-c.ProtectTail]
	if len(middle) == 0 {
		return msgs
	}

	// Archive the middle into DENDRITE. Single node per compress
	// pass keeps the graph tidy and avoids spamming persona nodes.
	if c.Archive != nil {
		id := "compaction-" + time.Now().UTC().Format("20060102T150405")
		body := renderArchive(middle)
		// Compression is best modelled as an "event" node — the robot's
		// NodeType enum lacks a Memory variant; using Event keeps it
		// timestamp-friendly and lets the relevance engine surface it.
		c.Archive.Upsert(
			id,
			"Compacted conversation",
			body,
			NodeTypeEvent,
			[]string{compactorTagSession},
		)
	}

	out := make([]transcriptMsg, 0, len(head)+1+len(tail))
	out = append(out, head...)
	out = append(out, transcriptMsg{
		Role: "system",
		Body: "[… transcript was compacted into DENDRITE memory. Prior turns are available via context retrieval …]",
	})
	out = append(out, tail...)
	return out
}

// renderArchive flattens a transcript slice into a single DENDRITE
// node body. Format is human-readable so the relevance engine can
// surface it later.
func renderArchive(middle []transcriptMsg) string {
	var b strings.Builder
	b.WriteString("Compacted AI conversation segment:\n\n")
	for i, m := range middle {
		b.WriteString("[")
		b.WriteString(m.Role)
		b.WriteString("] ")
		b.WriteString(m.Body)
		if i < len(middle)-1 {
			b.WriteString("\n\n")
		}
	}
	return b.String()
}
