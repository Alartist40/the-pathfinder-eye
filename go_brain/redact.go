// Package-internal redact helper. Slim port of Cynapse v2.3.0's
// internal/redact/redact.go — kept minimal because the robot's log
// surface is much smaller than the CLI agent's.
// CLOUD_API_KEY is set, and dumps brain transcripts to ./logs/*.log.
// If anyone speaks an API key near the mic, it would otherwise land
// in the logs unredacted.
//
// This file is a pure-function redact: zero dependencies on the rest
// of the brain, so tests can drop it in anywhere.
package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Patterns ported from cynapse/internal/redact — only the high-yield
// ones for the robot. New patterns can be added here without touching
// the rest of the brain.
var redactPatterns = []*regexp.Regexp{
	// OpenAI
	mustCompile(`sk-(proj-|svca-)?[A-Za-z0-9]{20,}`),
	mustCompile(`sk_live_[A-Za-z0-9]{10,}`),
	mustCompile(`sk_test_[A-Za-z0-9]{10,}`),
	// Anthropic
	mustCompile(`sk-ant-[A-Za-z0-9-]{20,}`),
	mustCompile(`sk_ant_[A-Za-z0-9]{10,}`),
	// Google AI Studio
	mustCompile(`AIza[A-Za-z0-9_-]{35}`),
	// AWS access-key IDs
	mustCompile(`AKIA[0-9A-Z]{16}`),
	// GitHub PATs
	mustCompile(`ghp_[A-Za-z0-9]{36}`),
	mustCompile(`github_pat_[A-Za-z0-9_]{82}`),
	mustCompile(`gho_[A-Za-z0-9]{36}`),
	mustCompile(`ghu_[A-Za-z0-9]{36}`),
	mustCompile(`ghs_[A-Za-z0-9]{36}`),
	mustCompile(`ghr_[A-Za-z0-9]{36}`),
	// HuggingFace
	mustCompile(`hf_[A-Za-z0-9]{20,}`),
	// Slack
	mustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`),
	// Stripe
	mustCompile(`sk_live_[A-Za-z0-9]{24,}`),
	mustCompile(`sk_test_[A-Za-z0-9]{24,}`),
	// PEM markers
	mustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),
	mustCompile(`-----BEGIN [A-Z ]*SECRET-----`),
	mustCompile(`-----BEGIN OPENSSH PRIVATE KEY-----`),
	// Bearer / JWT
	mustCompile(`(?i)Bearer\s+[A-Za-z0-9._\-+/=]{20,}`),
	mustCompile(`eyJ[A-Za-z0-9_\-+/=]{10,}\.[A-Za-z0-9_\-+/=]{10,}\.[A-Za-z0-9_\-+/=]{10,}`),
}

// envAssignmentPatterns: KEY=value where KEY is credential-like.
var envAssignmentPatterns = []*regexp.Regexp{
	mustCompile(`(?i)\b(aws_secret_access_key|aws_session_token|api_?key|access_?token|refresh_?token|auth_?token|bearer_?token|client_?secret|client_?id|private_?key|secret_?key|token|password|passwd|pwd)\b\s*=\s*['"]?([^'"\s]+)`),
	mustCompile(`(?i)"(api_?key|token|secret|password|access_?token|refresh_?token|auth_?token|bearer|client_?secret|client_?id|private_?key|secret_?value|key_?material)"\s*:\s*"([^"]+)"`),
}

const (
	redactHead  = 6
	redactTail  = 4
	redactFloor = 18
)

// mustCompile is a small wrapper around regexp.MustCompile that keeps
// the pattern declarations one-liners. Panics at init — same as the
// upstream cynapse/internal/redact package.
func mustCompile(p string) *regexp.Regexp {
	return regexp.MustCompile(p)
}

// redactOnce scans and replaces all detected secrets in `text` with a
// display-safe mask (head + *** + tail). Returns the original string
// when no secrets are detected. Safe to call from any goroutine.
func redactOnce(text string) string {
	if text == "" {
		return text
	}
	spans := scanRedact(text)
	if len(spans) == 0 {
		return text
	}
	var out strings.Builder
	cursor := 0
	for _, s := range spans {
		if s.Start > cursor {
			out.WriteString(text[cursor:s.Start])
		}
		out.WriteString(redactMask(text[s.Start:s.End]))
		cursor = s.End
	}
	if cursor < len(text) {
		out.WriteString(text[cursor:])
	}
	return out.String()
}

type redactSpan struct {
	Start int
	End   int
}

func scanRedact(text string) []redactSpan {
	var out []redactSpan
	for _, re := range redactPatterns {
		for _, loc := range re.FindAllStringIndex(text, -1) {
			out = appendUniqueSpan(out, loc[0], loc[1])
		}
	}
	for _, re := range envAssignmentPatterns {
		locs := re.FindAllStringSubmatchIndex(text, -1)
		for _, loc := range locs {
			// Loc shape: [m0_start, m0_end, g1_start, g1_end, g2_start, g2_end]
			if len(loc) >= 6 && loc[2] >= 0 && loc[3] >= 0 {
				out = appendUniqueSpan(out, loc[2], loc[3])
			}
		}
	}
	return out
}

func appendUniqueSpan(out []redactSpan, start, end int) []redactSpan {
	for _, s := range out {
		if s.Start <= start && end <= s.End {
			return out
		}
	}
	return append(out, redactSpan{Start: start, End: end})
}

func redactMask(value string) string {
	if value == "" {
		return ""
	}
	if len(value) < redactFloor {
		return strings.Repeat("*", len(value))
	}
	head := value[:redactHead]
	tail := value[len(value)-redactTail:]
	mid := strings.Repeat("*", len(value)-redactHead-redactTail)
	return head + mid + tail
}

// safeLogf wraps infoLog/errorLog so anything reaching the brain's
// log file passes through redactOnce first. Use this for log lines
// that may carry user speech transcripts or LLM responses.
//
// kind: "" for info-level, anything else for error-level.
func safeLogf(kind, format string, args ...interface{}) {
	msg := redactOnce(fmt.Sprintf(format, args...))
	switch kind {
	case "ERROR":
		if errorLog != nil {
			errorLog.Println(msg)
			return
		}
	default:
		if infoLog != nil {
			infoLog.Println(msg)
		}
	}
}
