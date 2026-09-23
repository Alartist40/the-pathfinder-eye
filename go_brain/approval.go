// approval.go — destructive-shell pattern detector for the robot.
//
// Slim port of cynapse/internal/approval: when a shell command or
// sudo invocation is being considered, classify its severity and
// decide whether the current speaker is allowed to run it. This is
// a HEURISTIC — a determined attacker can defeat it via variable
// expansion, eval, base64 — but the robot's threat model is
// accidental misuse, not adversarial prompt escape.
//
// Callsite contract:
//   - `CheckShell(cmd, level)` is called before any exec.Command
//     that runs a shell or a privileged binary.
//   - The robot's voice layer treats authority at three levels
//     (Guest, Scout, Programmer). Programmer = admin.
package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// approvalSeverity — order matters, see Compare below.
type approvalSeverity int

const (
	approvalNone approvalSeverity = iota
	approvalInfo
	approvalWarn
	approvalDanger
	approvalCritical
)

func (s approvalSeverity) String() string {
	switch s {
	case approvalCritical:
		return "critical"
	case approvalDanger:
		return "danger"
	case approvalWarn:
		return "warn"
	case approvalInfo:
		return "info"
	default:
		return "none"
	}
}

// approvalRule is a single pattern matched against the cleaned
// command. We retain Cynapse's pattern set with simplified names;
// rules are matched longest-first, first-hit-wins.
type approvalRule struct {
	name     string
	re       *regexp.Regexp
	severity approvalSeverity
	reason   string
}

var approvalRules = []approvalRule{
	// Critical — filesystem / device destruction
	{name: "mkfs", re: regexp.MustCompile(`\bmkfs(\.[a-z0-9]+)?\b`), severity: approvalCritical, reason: "formatting a filesystem"},
	{name: "dd-of-dev", re: regexp.MustCompile(`\bdd\b[^\n]*\bof=/dev/`), severity: approvalCritical, reason: "writing directly to a device node"},
	{name: "chmod-recursive-root", re: regexp.MustCompile(`\bchmod\s+(-R\s+)?777\s+/\b`), severity: approvalCritical, reason: "world-writable on /"},
	{name: "wipefs", re: regexp.MustCompile(`\bwipefs\b`), severity: approvalCritical, reason: "wiping filesystem signatures"},

	// Danger — local destructive
	{name: "rm-rf-root", re: regexp.MustCompile(`\brm\s+(-\w+|\S+\s)*\s*/(\s|$)`), severity: approvalDanger, reason: "rm targeting /"},
	{name: "rm-rf-glob", re: regexp.MustCompile(`\brm\b[^\n|&;]*\s-\w*r\w*f\w*[^\n|&;]*\*`), severity: approvalDanger, reason: "rm -rf with glob"},
	{name: "rm-rf", re: regexp.MustCompile(`\brm\b[^\n|&;]*\s-\w*r\w*f\w*\b`), severity: approvalDanger, reason: "rm -rf (review target)"},
	{name: "find-delete", re: regexp.MustCompile(`\bfind\b[^\n]*-delete\b`), severity: approvalDanger, reason: "find -delete"},
	{name: "truncate-zero", re: regexp.MustCompile(`\btruncate\b[^\n]*-s\s*0\b`), severity: approvalDanger, reason: "truncate to zero bytes"},

	// Danger — fork bombs / resource exhaustion
	{name: "forkbomb", re: regexp.MustCompile(`:\(\)\s*\{`), severity: approvalDanger, reason: "fork bomb pattern"},
	{name: "while-true", re: regexp.MustCompile(`\bwhile\s+true\s*;?\s*do\b`), severity: approvalDanger, reason: "infinite loop"},

	// Warn — outbound network / risky installs
	{name: "curl-pipe-shell", re: regexp.MustCompile(`\b(curl|wget|fetch)\b[^\n]*\|\s*(bash|sh|zsh)\b`), severity: approvalWarn, reason: "curl|pipe-to-shell pattern"},
	{name: "nc-reverse", re: regexp.MustCompile(`\bnc\b[^\n]*-[a-zA-Z]*e\b`), severity: approvalWarn, reason: "netcat with -e (reverse shell)"},
	{name: "bash-dev-tcp", re: regexp.MustCompile(`/dev/tcp/`), severity: approvalWarn, reason: "bash /dev/tcp reverse-shell pattern"},
	{name: "git-push-force", re: regexp.MustCompile(`\bgit\s+push\b[^\n]*--force`), severity: approvalWarn, reason: "force-push to git remote"},

	// Info — outbound reads
	{name: "curl", re: regexp.MustCompile(`\bcurl\b`), severity: approvalInfo, reason: "HTTP request"},
	{name: "wget", re: regexp.MustCompile(`\bwget\b`), severity: approvalInfo, reason: "HTTP request"},
	{name: "ssh", re: regexp.MustCompile(`\bssh\b`), severity: approvalInfo, reason: "ssh command"},
}

// approvalDecision is the result of an inspection.
type approvalDecision struct {
	Allow    bool
	Severity approvalSeverity
	Reason   string
	RuleName string
}

// CheckShell inspects a shell command and decides whether the
// supplied AuthorityLevel may run it. The matrix is roughly:
//
//	Severity     | Guest/Scout             | Programmer
//	------------ | ----------------------- | --------------
//	None/Info    | Allow (with log line)   | Allow
//	Warn         | Refuse                 | Allow
//	Danger       | Refuse                 | Refuse (must use voice confirmation)
//	Critical     | Refuse                 | Refuse (must use voice confirmation)
//
// "Refuse" returns Allow=false so callers can veto the action.
// Programmer voice-confirmation is enforced one layer up.
func CheckShell(cmd string, level AuthorityLevel) approvalDecision {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return approvalDecision{Allow: true}
	}

	highest := approvalNone
	hit := approvalRule{}
	for _, r := range approvalRules {
		if r.re.MatchString(cmd) && r.severity > highest {
			highest = r.severity
			hit = r
		}
	}
	if highest == approvalNone {
		return approvalDecision{Allow: true, Severity: approvalNone}
	}

	d := approvalDecision{
		Severity: highest,
		Reason:   hit.reason,
		RuleName: hit.name,
	}

	switch highest {
	case approvalCritical, approvalDanger:
		// Both ranks default-deny on danger; needs explicit Programmer
		// confirmation through voice layer (which has its own per-rank
		// table).
		d.Allow = false
	case approvalWarn:
		// Programmer may run, others may not.
		d.Allow = level >= LevelProgrammer
	case approvalInfo:
		// Informational only — log but allow.
		d.Allow = true
		safeLogf("", "APPROVAL_INFO: rule=%s cmd=%s", hit.name, redactOnce(cmd))
	}
	return d
}

// RunSudoCommand is the safe wrapper around `exec.Command("sudo", ...)`.
// It classifies the requested shell line (the args joined with spaces
// after the leading program name), and only when the classifier allows
// does it execute. The caller passes level to indicate the current
// speaker's authority; TouchedAuthority=LevelGuest is appropriate from
// wake-word callers that haven't authenticated yet.
//
// Returned string is a one-line summary for the safety log; nil error
// means the command actually ran. If approval refuses, error explains
// why and exec was *not* invoked.
//
// This replaces the dozens of bare `exec.Command("sudo", ...)` in
// voice_commands.go / main.go / leafcutter_swap.go etc. over time;
// roll out incrementally as each caller is reviewed.
func RunSudoCommand(level AuthorityLevel, args ...string) (string, error) {
	flat := strings.Join(args, " ")
	d := CheckShell(flat, level)
	if !d.Allow {
		safeLogf("", "APPROVAL_DENY: rule=%s sev=%s level=%d cmd=%s",
			d.RuleName, d.Severity, level, redactOnce(flat))
		return "", fmt.Errorf("approval refused (%s/%s): %s",
			d.RuleName, d.Severity, d.Reason)
	}
	if d.Severity != approvalNone {
		safeLogf("", "APPROVAL_PASS: rule=%s sev=%s level=%d",
			d.RuleName, d.Severity, level)
	}
	out, err := exec.Command("sudo", args...).CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil {
		return s, fmt.Errorf("sudo %s: %v", flat, err)
	}
	return s, nil
}
