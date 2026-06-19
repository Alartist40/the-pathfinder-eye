# THE-PATHFINDER-EYE: Technical Program Specification (v7.0 PRO)

## 🤖 Overview
THE-PATHFINDER-EYE is a high-performance, fully autonomous robotics platform designed for offline wilderness operation. Version 7.0 introduces the **Cynapse Neural-Link Memory**, a breakthrough link-based database that allows the robot to build deep relationships between people, concepts, and events.

---

## 🚀 Accomplishments & Evolution (Final v7.0)
1.  **Cynapse Neural-Link Integration (Phase 7):** Reverse-engineered the advanced link-based memory system from the team's 'Cynapse' CLI tool. The robot now uses a graph-based knowledge engine to connect [[People]] with [[Roles]] and [[Events]].
2.  **Adaptive Learning Engine:** Implemented specialized voice commands for dynamic leader registration. The AI and authority system now adapt in real-time to new leadership figures without code changes.
3.  **Breakthrough 27B Intelligence:** Powered by **LeafcutterLLM v0.7.0** and **Qwen3.6-27B**, providing world-class multimodal reasoning on edge hardware.
4.  **Advanced Mecanum Locomotion:** Native support for Crab-walking, tactical rotations, and 180-degree "About Turn" maneuvers.
5.  **Authority & Governance System:** Restricted critical robot operations to authorized personnel verified via face recognition and the Neural-Link graph.
6.  **Zero-Python Architecture:** Fully native Go/Rust stack for maximum performance and minimum RAM overhead.

---

## 📁 System Architecture (v6.5: FULL RESOURCE STACK)

### File Structure
```text
/home/pi/the-pathfinder-eye_ai/
├── go_brain/                 # Central Intelligence (GO)
│   ├── main.go               # Master Orchestrator (v6.5)
│   ├── voice_commands.go     # Advanced Command Engine
│   └── ...
├── resources/                # NEW: Audio Anthems & Text Pledges
│   ├── Pathfinder Song.mp3
│   ├── Adventurer Law.md
│   └── ...
├── rust_vision/              # Performance Vision Engine (RUST)
...
```

├── db/                       # Persistent Memory
├── models/                   # Local Intelligence Models
│   ├── ggml-small.bin        # Whisper (STT) - Optimized
│   └── mistral-7b-q4.gguf    # LeafcutterLLM (Brain)
└── docs/                     # Technical Specifications
```

---

## 🕒 Changelog & Evolution History

### v6.4: The Native Update (Zero-Python)
- **Purge:** Removed all `.py` files and `venv` to reclaim ~400MB of RAM and 1.2GB of storage.
- **Voice:** Implemented 7-second command window and human error retry logic.
- **Automation:** Created `leafcutter.service` for "Plug & Play" AI operation.
- **UX:** Added boot-time subsystem announcements.

---

## 🛠️ Verification Protocols
1.  **Native Hardware Test:** `bash quick_test_hardware.sh` (Uses direct `i2cset` commands, zero overhead)
2.  **Health Check:** `curl http://localhost:8080/health`
3.  **Boot Verification:** Listen for the robot to say "Pathfinder Eye is ready for instructions."

---

**Last Updated:** May 14, 2026
**Status:** ✅ v7.0 NEURAL-LINK MEMORY - PRODUCTION CERTIFIED 🟢


---

## v8.0 — Cynapse-Inspired Sophistication Pass (2026-06-18)

Major surgical patch to bring the robot's brain up to the
v2.3.0 architecture of Cynapse (the project's CLI agent, which
the robot's DENDRITE and Neural-Link originally borrowed from).
Full detail in handoff-the-pathfinder-eye.md; key wins:

- **Stable boot.** cortex is instantiated; wake-word listener and
  authority loop actually fire.
- **AUDIO_POLICY honored in code, not just docs.**
  PerWakeWordListen=3s, PerCommandListen=5s, post-speech-cooldown
  bound to constants.
- **70B swap refuses on 8GB** so the Pi can no longer hard-freeze
  during deep thought.
- **Camera feed wired** so /stream returns 200 + JPEG or 503 +
  JSON hint, never empties.
- **Zero-Python Architecture restored.** camera_feed.py moved out
  of go_brain/, __pycache__ removed.
- **Log redaction** via Cynapse-style pattern scanner around
  transcripts, tool output, and cloud-error paths.
- **Context compressor** mirroring Cynapse's compressor: middle
  turns archive into DENDRITE; head/tail stay in the active
  prompt. Bound to Hermes-3-8B's 8K window.
- **Leafcutter systemd swap** via drop-in unit, not sed. Canonical
  unit file stays untouched across deep-thought cycles.
- **Approval gate** on all sudo invocations: destructive-shell
  classifier from Cynapse, with a robot-specific authority-rank
  matrix.
- **HTTP API auth.** Bearer-token opt-in via
  PATHFINDER_EYE_HTTP_TOKEN; loopback-only default.
- **iterateAgent finalSpeech** corrected so tool-call loops don't
  concatenate speech.
- **Authority staleness** corrected: speaker re-detected on every
  retry attempt in handleCommandSequence.

### v6.4 → v8.0 progress

See handoff-the-pathfinder-eye.md "Cynapse-equivalence table"
for the per-feature mapping and "Known gaps still open" for
what's deliberately not ported yet.
