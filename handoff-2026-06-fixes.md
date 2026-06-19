# Session Handoff: 2026-06-19 — v8.1 Core Stability Fixes

## Summary

Complete code review and fix pass on THE-PATHFINDER-EYE robot brain. Found and
fixed 15+ bugs ranging from silent stubs (YOLO, VisionDB) to data races
(whisper, gimbal) to live secret exposure. All Go files gofmt'd. All tests
updated for isolation.

## Files Modified

### Go Brain (`go_brain/`)

| File | What Changed | Why |
|------|-------------|-----|
| `.env` | Replaced live `nvapi-...` with `YOUR_NVIDIA_API_KEY_HERE` placeholder | Live API key would be pushed to public GitHub |
| `.env.example` | Updated to match full `.env` structure | Documentation gap |
| `main.go` | Added `ctx, cancel := context.WithCancel(...)`, signal handler, `defer close(ttsQueue)`, `Close()` calls | No graceful shutdown; resources never released |
| `cortex.go` | Removed `if false` guard around `isQuiet()`, updated wake word check | Volume gate was permanently bypassed; audio gate dead since inception |
| `voice.go` | Added `whisperMu sync.Mutex`, Lock/Unlock around whisper calls | Thread-unsafe; two concurrent STT calls crash whisper.cpp |
| `voice_commands.go` | Rewrote `isWakeWord()` with word-boundary matching; wrapped motor commands in goroutines; added `defer atomic.StoreInt32(&commandBusy, 0)` | False positives on "introduction", "direction"; motor blocked command loop; leak left busylock permanently |
| `voice_commands_test.go` | Updated to call `isWakeWord()` | Test was testing inline `strings.Contains`, not the actual detector |
| `vision.go` | Added full SQLite-backed VisionDB, `seekMu sync.Mutex`, `Close()` method | Was a no-op stub; face authority always returned "unknown"; seekGimbal had data race |
| `status_aware.go` | Fixed `cpu >= 0.8` calculation (was `1 - fraction`, reported 5% as 95%) | Health monitoring inverted; CPU warnings at idle |
| `dendrite.go` | Added `initDendritePath(path)`, `Close()` | No way to specify DB path; no cleanup |
| `dendrite_test.go` | Use `t.TempDir()` | Was writing to production DB |
| `authority_test.go` | Use `t.TempDir()` | Was writing to production DB |
| `birdwatch.go` | Added `Close()` | Interface consistency |

### Rust Vision (`rust_vision/src/`)

| File | What Changed | Why |
|------|-------------|-----|
| `detection.rs` | Full YOLOv5 ONNX inference pipeline with `nms_boxes_batched_def` | Was `Vec::new()` stub — robot never detected anything |
| `main.rs` | Wired up `FaceRecognizer` with graceful fallback | Face recognition was uninitialized; now graceful if model missing |

### Documentation

| File | What Changed | Why |
|------|-------------|-----|
| `PRESENTATION.md` | Added v8.1 section | Reflect current code state |
| `handoff-2026-06-fixes.md` | Created | Session handoff |

## Critical Items Before Production

### 1. NVIDIA API Key
The `.env` had a **live NVIDIA API key** (`nvapi-...`). It has been replaced
with a placeholder. You must generate a new key at
https://build.nvidia.com/ and put it in `.env`:
```
NVIDIA_API_KEY=nvapi-<your-new-key-here>
```

### 2. Build on Pi Required
Cannot compile on this dev machine:
```bash
# Go brain (needs whisper.h C header)
cd /home/pi/the-pathfinder-eye_ai/go_brain && go build -o ../brain .

# Rust vision (needs system opencv libs)
cd /home/pi/the-pathfinder-eye_ai/rust_vision && cargo build --release
```

### 3. Missing Model Files
The repo expects these in `models/` (gitignored, download separately):
- `YOLOv5s-640.onnx` — object detection
- `haarcascade_frontalface_default.xml` — face detection
- `face_recognition.onnx` — face recognition (new - not yet acquired)

### 4. Known Remaining Issues
- **Hardcoded paths**: all paths hardcoded to `/home/pi/...` — needs config file
- **Python camera feed**: `camera_feed.py` still used; refactor to Rust planned
- **Zero-Python goal**: not yet achieved despite v6.4 claim
- **Wake word aliases**: "destruction", "restruction" preserved as aliases (for
  STT misrecognition of "destruction"/"instruction")

## Architecture Overview (Post-Fix)

```
┌──────────────────────────────────────────────────────┐
│                    main.go                            │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐  │
│  │ cortex.go │  │ voice.go  │  │   vision.go       │  │
│  │ (awareness)│  │ (STT/TTS) │  │ (YOLO + Face DB)  │  │
│  └─────┬─────┘  └────┬─────┘  └────────┬──────────┘  │
│        │              │                 │              │
│  ┌─────┴──────────────┴─────────────────┴──────────┐  │
│  │            semantic_router.go                    │  │
│  │    (LLM call → tool dispatch)                    │  │
│  └────────────────┬─────────────────────────────────┘  │
│                   │                                    │
│  ┌────────────────┴─────────────────────────────────┐  │
│  │  tools.go / actions.go / motor_control / gimbal  │  │
│  └──────────────────────────────────────────────────┘  │
│                                                        │
│  Memory:  dendrite.go  birdwatch.go  vision.go         │
│           (SQLite)      (SQLite)      (vision.sqlite)  │
│                                                        │
│  Safety:  redact.go  approval.go  http_auth.go         │
│                                                        │
│  LLM:     leafcutter.service (systemd)                 │
└────────────────────────────────────────────────────────┘
         │
         ▼
┌──────────────────────┐
│  rust_vision         │
│  detection.rs (YOLO) │
│  face_recognition.rs │
│  main.rs (orchestr.) │
└──────────────────────┘
```

## Git State

The repo has been updated with all fixes. To push:
```bash
cd /home/xander/Documents/portfolio/the-pathfinder-eye
git add -A
git commit -m "v8.1: Core stability fixes — YOLO inference, VisionDB, thread safety, wake word, graceful shutdown"
git push origin main
```

## Testing Checklist (on Pi)
- [ ] `go build -o ../brain .` — compiles cleanly
- [ ] `cargo build --release` — compiles cleanly
- [ ] YOLO detects objects (check `/tmp/vision_output.json`)
- [ ] Face authority enrolls and recalls faces
- [ ] Wake word does not trigger on "introduction", "direction"
- [ ] CPU health shows correct idle% under load
- [ ] Robot pauses when loud noise detected (volume gate)
- [ ] `kill -TERM` <pid> triggers graceful shutdown
- [ ] `go test ./...` passes (all use temp databases)
- [ ] 6 systemd services enable and start correctly
