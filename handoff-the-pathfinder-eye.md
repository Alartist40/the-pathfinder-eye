# THE-PATHFINDER-EYE — Handoff Document

## Overview
Autonomous wilderness robot scout with I2C motor control, offline multimodal reasoning (LeafcutterLLM + Ministral), local vision/voice processing, advanced memory/context systems, and Adventurer/Pathfinder rank tracking.

**Date:** 2026-06-02
**Author:** Gemini CLI

## Current Status: OPERATIONAL ✅

### Hardware (Pi 5, 8GB RAM)
| Component | Status | Details |
|---|---|---|
| **Motors/I2C** | ✅ Online | 4WD via I2C at `0x2B`; Gimbal pan/tilt active |
| **Audio (Mic)** | ✅ Online | USB Audio Device, 44.1kHz stereo capture |
| **Audio (Speaker)** | ✅ Online | Lightweight `espeak-ng` TTS working via async queue |
| **Whisper Voice** | ✅ Loaded | `ggml-tiny.en.bin` on CPU for fast wake-word |
| **Wake Word** | ✅ Listening | Keyword: `"Instruction"` (5-second command window) |

### Software Stack (Split Architecture)
| Component | Status | Details |
|---|---|---|
| **Go Brain** | ✅ Running | Main orchestrator; handles I2C, STT/TTS, DBs |
| **Rust Vision** | ✅ Running | Standalone `pathfinder-vision` service (YOLOv5 + Haar) |
| **LeafcutterLLM** | ✅ Built | Uses `llama-ffi` backend for native high-performance execution |
| **AI Model** | ✅ Loaded | `Ministral-3-3B-Reasoning-2512-Q4_K_M.gguf` |
| **Memory** | ✅ Online | `Dendrite` SQLite graph for identity, ranks, and context |

---

## Recent Architectural Upgrades

### 1. High-Performance LLM Reasoning (`llama-ffi`)
- **Problem:** The previous 'Native Engine' scalar loops were too slow on the Pi 5 CPU, causing freezes.
- **Solution:** Rebuilt `leafcutter` in Rust using the `llama-ffi` feature flag, linking directly to the highly optimized `llama.cpp` backend.
- **Result:** Real-time generation speeds restored. Verified using the `Ministral-3B` model (peaks around ~500MB RSS).

### 2. Lightweight & Non-Blocking TTS
- **Problem:** Piper TTS was heavy and voice processing blocked the main brain loop.
- **Solution:** Switched exclusively to `espeak-ng` for minimum RAM footprint. Implemented a channel-based queue (`ttsWorker`) so speech requests are handled asynchronously.
- **Result:** Instant status messages and dialogue without overlapping or stalling the orchestrator.

### 3. Standalone Vision Pipeline
- **Problem:** Running vision inside the main loop risked latency and OOM issues.
- **Solution:** Moved vision to a dedicated Rust service (`pathfinder-vision.service`). It polls the camera at 30 FPS and outputs structured events (JSON) to shared memory.
- **Result:** Deterministic perception. The Go brain simply reads the lightweight JSON output.

### 4. Socially Aware Command Security
- **Logic:** Face ID -> Rank Lookup -> Permission Check -> Execution.
- **Flow:** When the wake word is triggered, the robot identifies the speaker via the vision database. It assigns a rank (Guest, Scout, Leader, etc.). Critical commands (like activating the AI via "Attention") are blocked for Guests.
- **Context:** The speaker's identity is prepended to the prompt context, giving the AI model social awareness during conversation.

---

## Architecture Flow

### The "Attention" Lifecycle (RAM Saver)
To preserve the 8GB RAM limit, the heavy LLM is kept offline until explicitly requested.
1.  **Idle State:** Go Brain + Rust Vision + Whisper (listening for "Instruction"). LLM is OFF.
2.  **Wake Word:** User says "Instruction". Robot flashes green, says "Yes".
3.  **Command Window:** Robot listens for 5 seconds.
4.  **Activate AI:** User says "Attention".
5.  **Authority Check:** Robot verifies the speaker is at least a Scout. If yes, it starts `leafcutter.service`.
6.  **Reasoning Loop:** Robot listens for 5s intervals, passes transcripts to the LLM (along with Dendrite context and speaker ID), and speaks the response.
7.  **Deactivate AI:** User says "Instruction Sleep". The LLM service is killed, freeing RAM immediately.

### Subsystems Audit
1.  **Dendrite (Memory):** ✅ FUNCTIONAL (SQLite WAL mode. Handles relationships via `[[links]]` and `#tags`).
2.  **Authority (Ranks):** ✅ FUNCTIONAL (5 levels. Integrated into voice command loop).
3.  **Vision Engine:** ✅ FUNCTIONAL (Rust service writes to `/tmp/detections.json`).
4.  **Voice (STT/TTS):** ✅ FUNCTIONAL (Whisper STT + Async `espeak-ng`).

---

## Code Locations

```
/home/xander/Documents/portfolio/THE-PATHFINDER-EYE copy/
├── go_brain/
│   ├── main.go              # Orchestrator & API
│   ├── voice.go             # Async TTS queue + Whisper STT
│   ├── voice_commands.go    # Wake word + Rank verification loop
│   ├── dendrite.go          # Memory Engine
│   └── vision.go            # Reads JSON events + Speaker ID
├── rust_vision/
│   ├── src/main.rs          # 30FPS Rust vision engine
│   └── Cargo.toml
├── LeafcutterLLM/
│   └── rust/target/release/leafcutter  # FFI-enabled LLM Server
├── models/
│   └── Ministral-3-3B-Reasoning-2512-Q4_K_M.gguf
├── db/                      # SQLite databases (dendrite, vision)
└── systemd/                 # Service configs (brain, vision, leafcutter)
```

## Voice Commands Summary

| Command | Rank Required | Action |
|---|---|---|
| `"Instruction"` | Any | Wake word — activates 5s command mode |
| `"Move forward/back"` | Any | I2C deterministic movement |
| `"Look left/right"` | Any | Gimbal servo movement |
| `"Attention"` | Scout+ | Starts LLM server and enters AI conversation loop |
| `"Sleep"` | Masterguide+ | Kills LLM server to save power/RAM |
| `"Test"` | Any | Runs full hardware diagnostic sequence |


---

## v8.0 (2026-06-18) — Cynapse-Inspired Sophistication Pass

The robot was reviewed against the v2.3.0 architecture of Cynapse —
the CLI agent that originally inspired DENDRITE and the Cynapse
Neural-Link. The robot's brain had ~13 known issues at the point
of review. Of those, the following CRITICAL bugs were fixed:

### Critical fixes shipped
1. **main.go nil-cortex panic.** `cortex` was declared as
   `*AICortex` but never instantiated; the boot goroutine called
   `cortex.StartUnifiedAwareness()` and crashed silently before
   wake-word listening and authority could fire. `newCortex()`
   constructor added; safe nil-guard in StartUnifiedAwareness.
2. **/stream endpoint returned empty payload forever.** camera_feed.py
   writes `/tmp/vision_feed.jpg`, but no systemd unit installed
   the script — so the JPEG never existed. New
   `camera-feed.service` (in `systemd/`). `/stream` now also
   returns 503 + JSON error when the file is absent, instead of
   200 with an empty body.
3. **AUDIO_POLICY 5-second window violated in code.** voice.go and
   `handleCommandSequence` had capture loops expanding the listen
   window out to ~13 seconds. Constants `PerWakeWordListenSec`,
   `PerCommandListenSec`, `PostSpeechCooldownSec` defined in
   voice.go; both code paths use them. The robot now honors the
   policy and `time.Since(lastSpokeTime)` is bound to the
   constants too.
4. **enterDeepThought would freeze the Pi 5.** The router swapped
   the systemd unit to a 70B Q4 model; that needs ~22 GB RAM, the
   Pi has 8 GB. Added `MinFreeMBFor70B=20480` guard plus
   post-swap re-check that reverts if memory drops under the
   floor. Voice announces "Deep thought swap failed" on refusal.
5. **Zero-Python Architecture promise broken.** camera_feed.py
   still lived in `go_brain/` with a `__pycache__/` next to it
   despite the v6.4 changelog claim. Both moved to `scripts/` and
   cleaned up.
6. **No log redaction** — cloud API keys, transcripts, and tool
   output went straight into the journal. New `redact.go` with
   `safeLogf` helper, 24 credential-shaped patterns, plus agent
   wrapper around the cloud-error path.
7. **Long AI conversations were unbounded.** Hermes-3-8B has an
   8K context window; ask the robot 30 things and it'll start
   hallucinating. New `compressor.go` mirrors Cynapse's
   context-compression: archive middle turns into DENDRITE
   (typed as Event nodes), keep head+tail in the active
   transcript. Same `chars/4` token accounting, no external
   tokenizer dependency.
8. **Leafcutter model swap used `sed -i` on the systemd unit.**
   Racy (clobbers concurrent edits), string-brittle (replaces any
   `--model anything`). Replaced with `leafcutter_swap.go`:
   `SwapLeafcutterModel()` and `RevertLeafcutterSwap()` write a
   drop-in unit at `/etc/systemd/system/leafcutter.service.d/`
   and call `daemon-reload + restart`. Canonical unit file
   remains untouched across swaps.
9. **No sudo-grade gating.** Original `exec.Command("sudo", ...)`
   calls scattered across voice_commands.go, main.go,
   leafcutter_swap.go. Added `approval.go` with destructive-shell
   pattern detector (`mkfs / dd-of-dev / rm-rf / fork-bomb /
   curl|bash / chmod-777-/-R`). `RunSudoCommand(level, args...)`
   replaces bare sudo — dangerous patterns default-deny, info-sev
   log-only.
10. **Loopback/LAN API had no auth.** Anyone on the LAN could
    drive motors via `/move`, run any LLM prompt via `/ai/think`,
    and tail camera frames via `/stream`. New `http_auth.go`
    enforces loopback-only as the default; with
    `PATHFINDER_EYE_HTTP_TOKEN` set, requires `Authorization:
    Bearer <token>`. Defense in depth: cloud-shaped tokens are
    rejected for the auth header itself so they don't land in
    logs the redact pass might miss.
11. **iterateAgent lost finalSpeech on tool-call loops.** The old
    code did `finalSpeech += cleaned + " "` across iterations
    and `break`'d after a tool-call turn, so the spoken reply
    could be concatenated noise. Replaced with "track latest
    cleaned content", preserved across iterations via
    `prevSpoken`.
12. **Authority snapshot went stale mid-sequence.** At the top of
    `handleCommandSequence`, `speaker` was snapshotted once and
    reused across the retry loop. Re-snapshot on every attempt
    so a face re-detection during the wake window doesn't run a
    command under a stale identity.

### New files
- `go_brain/redact.go` — secret-pattern scanner + safeLogf.
- `go_brain/compressor.go` — context-window compactor (DENDRITE
  archive).
- `go_brain/leafcutter_swap.go` — structured systemd drop-in
  swap helper.
- `go_brain/approval.go` — destructive-shell classifier +
  RunSudoCommand wrapper.
- `go_brain/http_auth.go` — bearer-token / loopback-only HTTP
  gate.
- `systemd/camera-feed.service` — runs camera_feed.py so
  /stream has data to serve.

### Files moved
- `go_brain/camera_feed.py` → `scripts/camera_feed.py`
  (matches the v6.4 zero-python architecture claim).

### Cynapse-equivalence table (after v8.0)
| Cynapse                  | Robot port                              |
|--------------------------|-----------------------------------------|
| internal/redact          | redact.go                               |
| internal/compressor      | compressor.go (DENDRITE-typed archive)  |
| internal/approval        | approval.go (no TUI confirm; uses rank) |
| internal/confirm         | rank-based gates in authority.go        |
| internal/netguard        | not ported (no network egress yet)      |
| internal/api             | http_auth.go + main.go routing          |

### Known gaps still open
- `internal/netguard` — the robot never makes outbound HTTP
  except via the leafcutter/ChatGPT fallback, so a netguard
  port isn't load-bearing yet. Revisit when more plugins
  appear.
- Branch coverage in `voice_commands_test.go` / `dendrite_test.go`
  needs new tests for compressor and redact.
- Cortex → Cynapse Cortex integration (`leafcutter.service`
  fanned out via Cynapse MCP) is on hold until Cynapse v3.x.

### Deployment notes
- Build on the Pi, not the dev workstation. The cgo whisper
  binding needs the `whisper.h` header from the robot's installed
  whisper.cpp. Dev machines can run `gofmt -l .` and LSP checks
  but not full `go build`.
- New systemd unit `camera-feed.service` must be enabled ONCE on
  the Pi: `sudo cp systemd/camera-feed.service /etc/systemd/system/
  && sudo systemctl daemon-reload && sudo systemctl enable --now
  camera-feed.service`.
- To lock down the LAN API, set
  `PATHFINDER_EYE_HTTP_TOKEN` in /etc/default/pathfinder-eye or
  the systemd unit's Environment=.
