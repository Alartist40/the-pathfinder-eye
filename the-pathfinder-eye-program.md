# THE-PATHFINDER-EYE: Technical Program Specification (v9.0 PRO)

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

## 📁 System Architecture (v9.0: NEEDLE 2 UNIFIED ENGINE)

### File Structure
```text
/home/pi/the-pathfinder-eye_ai/
├── go_brain/                 # Central Intelligence (GO)
│   ├── main.go               # Master Orchestrator (v9.0)
│   ├── voice_commands.go     # Advanced Command Engine
│   ├── needle_intent.go      # Needle 2 Client (replaces trm_intent.go)
│   └── ...
├── resources/                # NEW: Audio Anthems & Text Pledges
│   ├── Pathfinder Song.mp3
│   ├── Adventurer Law.md
│   └── ...
├── rust_vision/              # Performance Vision Engine (RUST)
├── db/                       # Persistent Memory
├── models/                   # Local Intelligence Models
│   ├── needle2.cact          # Needle 2 (45M params, 14MB, 28MB RAM)
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

**Last Updated:** September 8, 2026
**Status:** ✅ v9.0 NEEDLE 2 INTEGRATION - PRODUCTION CERTIFIED 🟢


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

---

## v9.0 — Needle 2 Integration: Unified Tool & Intent Engine (2026-09-08)

### What changed
The voice command pipeline has been upgraded with **Needle 2**
— a 45M-parameter edge model that unifies hardware tool calling and intent classification
into a single self-contained binary running in ~28MB of RAM on port 8082.

**The model:** Needle 2 (`needle2.cact`, 14MB).
Hadamard MLP, GQA attention, engram key-value memory, and grammar-constrained decoding.
Decode speed: 500 tok/s on Raspberry Pi 5 CPU.

### Architecture (v9.0)

```
┌────────────────────────────────────────────────────────┐
│                    main.go                              │
│  ┌──────────┐  ┌──────────┐  ┌─────────────────────┐  │
│  │ cortex.go │  │ voice.go  │  │   vision.go         │  │
│  │(awareness)│  │(PocketTTS)│  │ (YOLO + Face DB)    │  │
│  └─────┬────┘  └────┬─────┘  └──────────┬──────────┘  │
│        │            │                   │               │
│  ┌─────┴────────────┴───────────────────┴──────────┐  │
│  │     needle_intent.go (Needle 2 Client)          │  │
│  │  processCommandNeedle() → POST localhost:8082   │  │
│  │  confidence < 0.5? → fallback Ministral 3B      │  │
│  └──────────────────┬───────────────────────────────┘  │
│                     │                                  │
│  ┌──────────────────┴───────────────────────┐  │
│  │  dispatchAction() & Hardware Tool Exec   │  │
│  └──────────────────────────────────────────┘  │
│                                                        │
│  Memory:  dendrite.go  birdwatch.go  vision.go         │
│  Safety:  redact.go  approval.go  http_auth.go         │
│  LLM:     leafcutter.service (Ministral 3B)            │
│  TTS:     pocket-tts.service (Port 8020)               │
└────────────────────────────────────────────────────────┘
         │                          │
         ▼                          ▼
┌──────────────────────┐  ┌──────────────────────┐
│  rust_vision          │  │  needle.service      │
│  detection.rs (YOLO)  │  │  (Needle 2 Engine)   │
│  face_recognition.rs  │  │  localhost:8082      │
│  main.rs (orchestr.)  │  │  /complete endpoint  │
└──────────────────────┘  └──────────────────────┘
```

### Components

| Component | Location | Role |
|-----------|----------|------|
| Needle Tools JSON | `config/needle_tools.json` | 7 tools: move, look, light, play_resource, read_document, activate, deactivate |
| Go integration | `go_brain/needle_intent.go` | `processCommandNeedle` + confidence fallback & health circuit breaker |
| Systemd daemon | `systemd/needle.service` | ARM64 Needle 2 daemon serving `/complete` on port 8082 |
| Pocket-TTS daemon | `systemd/pocket-tts.service` | RAM-bound Pocket-TTS daemon serving `/tts` on port 8020 |

### Fallback safety net


With Needle 2 handling intent classification and tool calling in a single
45M-param binary, both the old TRM and FunctionalGemma are removed — freeing
~4 GB of RAM. The "thinking mode" (triggered by saying "Attention") already
uses LeafcutterLLM/Ministral via `leafcutter.service` — that path is unchanged.

### Deployment

1. Copy `needle.service` and `pocket-tts.service` to `/etc/systemd/system/`
2. Enable and start: `sudo systemctl daemon-reload && sudo systemctl enable --now needle.service pocket-tts.service`
3. Test: say "Instruction ... move forward" — robot executes in sub-10ms via Needle 2

### AntiDoom companion deployment (conversation quality)

The Ministral-3B conversation model gets an AntiDoom FTPO LoRA adapter to eliminate repetition loops. This is a model-level upgrade — zero code changes in go_brain/.

1. Build prompts: `python antidoom/build_prompts.py --dendrite ... --output conversation_prompts.jsonl`
2. Train on Colab: open `antidoom/AntiDoom_Training_Colab.ipynb`, run all cells (~30 min on T4), download `ministral-3b-antidoom.gguf`
3. Copy to Pi: `scp ministral-3b-antidoom.gguf pi@robot:~/the-pathfinder-eye_ai/models/`
4. Deploy: `./antidoom/deploy_antidoom.sh ministral-3b-antidoom.gguf` (backs up old model, swaps in systemd unit, restarts leafcutter, verifies)
5. Test: say "Attention" and have a long conversation — model should not enter repetition loops
