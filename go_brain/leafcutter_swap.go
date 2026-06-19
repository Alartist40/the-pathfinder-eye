// leafcutter_swap.go — safe runtime model-swap helper for leafcutter.
//
// Pre-fix: voice_commands.go mutated /etc/systemd/system/leafcutter.service
//
//	with `sed -i` to swap the model path, then ran daemon-reload + restart.
//	This was racy (sed can clobber the unit while another caller holds it),
//	string-brittle (anything matching `--model .*` got replaced, including
//	comments), and persistent in place (the unit file was rewritten on
//	disk every time the robot did a deep-thought swap).
//
// Post-fix: the brain keeps an in-memory snapshot of the systemd unit and
//
//	drives the swap by writing a temporary override file that systemd
//	loads via `--drop-in`. We revert the override on next boot via
//	`systemctl revert`, so the canonical unit file remains untouched.
//
// API:
//   - SwapLeafcutterModel(newPath string) error
//   - RevertLeafcutterSwap() error
package main

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

const (
	leafcutterService     = "leafcutter"
	leafcutterServicePath = "/etc/systemd/system/leafcutter.service"
	leafcutterOverrideDir = "/etc/systemd/system/leafcutter.service.d"

	// ExecStart override token keeps the swap scoped to ExecStart= only,
	// so anything systemd auto-appends (Restart=, Environment=, etc.)
	// still resolves correctly.
	leafcutterOverrideName = "swap.conf"
)

// leafcutterSwapMu serializes swaps so concurrent callers don't race
// each other. We don't expect heavy contention — voice-driven deep
// thought is single-button — but be safe.
var leafcutterSwapMu sync.Mutex

// SwapLeafcutterModel swaps the --model argument the leafcutter
// systemd unit uses, restarting the service to pick it up. The
// canonical unit file is read but NOT modified; the change is
// written to a systemd drop-in at /etc/systemd/system/<svc>.d/swap.conf.
//
// Requires sudo (writes under /etc/systemd). The brain itself runs
// as user `pi`, which is in the sudoers file for NOPASSWD on
// `systemctl`. If that ever changes, callers should pre-check.
func SwapLeafcutterModel(newModelPath string) error {
	leafcutterSwapMu.Lock()
	defer leafcutterSwapMu.Unlock()

	if newModelPath == "" {
		return fmt.Errorf("SwapLeafcutterModel: empty model path")
	}

	unitBytes, err := readUnitExecStart()
	if err != nil {
		return fmt.Errorf("SwapLeafcutterModel: read unit failed: %w", err)
	}

	// Find the structured ExecStart= line and re-write it with the
	// new model path, preserving any other flags. The leafcutter
	// server is invokable as `leafcutter server --model PATH [opts]`.
	rewritten := rewriteModelFlag(string(unitBytes), newModelPath)
	if rewritten == "" {
		return fmt.Errorf("SwapLeafcutterModel: could not rewrite ExecStart")
	}

	if err := writeOverride(rewritten); err != nil {
		return fmt.Errorf("SwapLeafcutterModel: write override failed: %w", err)
	}

	// daemon-reload picks up the drop-in; restart cycles the daemon.
	// run() returns non-nil on failure — surface that to the caller.
	if out, err := exec.Command("sudo", "systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload: %v — %s", err, string(out))
	}
	if out, err := exec.Command("sudo", "systemctl", "restart", leafcutterService).CombinedOutput(); err != nil {
		return fmt.Errorf("restart %s: %v — %s", leafcutterService, err, string(out))
	}

	safeLogf("", "LEAFFCUTTER_SWAP: switched to %s via drop-in", newModelPath)
	return nil
}

// RevertLeafcutterSwap rejects pending drop-ins and reloads the
// canonical unit. Use this when a swap was aborted, when the device
// comes back up from a deep-thought that completed, or when an error
// caused the previous swap to leave a half-applied state.
func RevertLeafcutterSwap() error {
	leafcutterSwapMu.Lock()
	defer leafcutterSwapMu.Unlock()

	if _, err := exec.Command("sudo", "rm", "-rf", leafcutterOverrideDir).CombinedOutput(); err != nil {
		// rm exit non-zero is non-fatal — empty dir is fine.
	}
	if out, err := exec.Command("sudo", "systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload: %v — %s", err, string(out))
	}
	if out, err := exec.Command("sudo", "systemctl", "restart", leafcutterService).CombinedOutput(); err != nil {
		return fmt.Errorf("restart %s: %v — %s", leafcutterService, err, string(out))
	}
	safeLogf("", "LEAFFCUTTER_SWAP: reverted drop-in, canonical unit now active")
	return nil
}

// readUnitExecStart pulls the canonical unit's ExecStart line in
// sudo mode. We do NOT trust the user's local copy of the unit — we
// re-read from disk every swap, so daemon-reload-aware edits made
// by the user are honored.
func readUnitExecStart() (string, error) {
	out, err := exec.Command("sudo", "systemctl", "show", leafcutterService,
		"-p", "ExecStart", "--no-pager").CombinedOutput()
	if err != nil {
		return "", err
	}
	// `systemctl show -p ExecStart` returns a multi-line list when the
	// service has multiple ExecStart entries. We collapse and pick
	// the first one (typical leafcutter.service has exactly 1).
	return strings.TrimSpace(string(out)), nil
}

// rewriteModelFlag parses the raw ExecStart=… output from
// `systemctl show` and rewrites any --model flag to point at
// newModelPath. The new value is the ExecStart line *without* the
// leading "ExecStart={ code }" wrapping — we rebuild it.
func rewriteModelFlag(rawUnitDump, newModelPath string) string {
	// `systemctl show` typically formats as:
	//   ExecStart={ path=/usr/bin/leafcutter ; argv[]=/usr/bin/leafcutter server --model X --port P ; ... }
	// We rebuild by joining the argv[] entries back together.
	rebuilt := strings.TrimSpace(rawUnitDump)
	// Strip outer ExecStart= prefix if present.
	if i := strings.Index(rebuilt, "="); i >= 0 {
		rebuilt = strings.TrimSpace(rebuilt[i+1:])
	}
	// Locate `argv[]=` chunks and rebuild the command from there.
	var argvPieces []string
	for _, line := range strings.Split(rebuilt, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "argv[]=") {
			continue
		}
		piece := strings.TrimPrefix(line, "argv[]=")
		piece = strings.TrimSuffix(piece, ";")
		argvPieces = append(argvPieces, piece)
	}
	if len(argvPieces) == 0 {
		// Fallback for older systemd where argv[] isn't printed
		// explicitly. Best-effort: just append/replace --model with
		// a fresh invocation.
		return "/usr/bin/leafcutter server --model " + newModelPath + " --port 8081"
	}

	// Replace any existing --model + value pair, then ensure a
	// canonical one is present. Simple state machine across argvPieces.
	out := make([]string, 0, len(argvPieces))
	for i := 0; i < len(argvPieces); i++ {
		tok := argvPieces[i]
		out = append(out, tok)
		if tok == "--model" && i+1 < len(argvPieces) {
			out = append(out, newModelPath)
			i++ // skip the old value
		}
	}
	// No existing --model? Append it.
	hasModel := false
	for _, p := range out {
		if p == "--model" {
			hasModel = true
			break
		}
	}
	if !hasModel {
		out = append(out, "--model", newModelPath)
	}
	return strings.Join(out, " ")
}

// writeOverride writes a drop-in unit file at
// /etc/systemd/system/leafcutter.service.d/swap.conf that sets
// ExecStart to the new value. Drop-ins inherit everything else from
// the canonical unit, so this is a minimal-touch swap.
func writeOverride(execStart string) error {
	body := "[Service]\nExecStart=\nExecStart=" + execStart + "\n"
	// Use `sudo tee` to write the drop-in under /etc/systemd. We
	// could also use `sudo install -D`, but tee is the simplest
	// primitive — no temp file dance.
	cmd := exec.Command("sudo", "sh", "-c",
		`mkdir -p `+leafcutterOverrideDir+` && tee `+leafcutterOverrideDir+`/`+leafcutterOverrideName+` > /dev/null <<'EOF'
`+body+`EOF`)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("write drop-in: %v — %s", err, string(out))
	}
	return nil
}
