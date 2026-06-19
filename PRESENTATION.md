# THE-PATHFINDER-EYE — The Integrated AI Cortex

> **An autonomous AI agent with a physical body. Powered by a 70B parameter brain. Running entirely offline on a Raspberry Pi 5.**

---

## 🎙️ The One-Sentence Pitch

**THE-PATHFINDER-EYE is the first autonomous robotic scout to run a world-class 70-Billion parameter AI model natively on consumer-grade hardware, transforming raw sensory data into agentic, real-time physical action.**

---

## 🏔️ The Challenge: Intelligence in the Wilderness

Most modern robots are just "remote-controlled cars with a camera." If they have AI, it usually relies on the cloud. In the wilderness, the cloud doesn't exist.
- **Problem 1:** High-end AI models (Llama 70B) typically require $20,000 GPUs and massive power.
- **Problem 2:** Traditional "Multimodal" AI is too slow (10-20 seconds per thought) for a moving robot.
- **Problem 3:** Communication is clunky. You talk, you wait, the robot "thinks," then acts. It feels mechanical, not alive.

**We didn't just build a robot. We built a Cortex.**

---

## 🧠 The Breakthrough: The Integrated Cortex

Instead of duct-taping separate machine learning scripts together, we designed a **Biological-Inspired Architecture** composed of three optimized layers:

### 1. The 70B Parameter Barrier (Broken) 🏆
Using our custom **LeafcutterLLM** engine, we proved that intelligence is not reserved for the elite. 
- **The Feat:** We successfully ran the **Llama 3.1 70B model** (38GB) on a Raspberry Pi 5.
- **The Secret:** Layer-by-layer streaming. We only keep the active part of the "brain" in the 8GB RAM, achieving a peak footprint of only **1.2GB**.

### 2. Sensor-to-Text Fusion (Real-Time Awareness)
To eliminate AI latency, we developed **Textual Awareness**.
- **Rust Vision:** A high-speed Rust engine (YOLOv5) scans the world at 5-10 FPS.
- **Fusion:** It converts pixels and ultrasonic distances into a live text stream (e.g., *"I see a person 2m ahead. A bird is high in a tree to the left."*).
- **Result:** The AI already "knows" the world before you even ask it a question. No image-processing lag.

### 3. The Starling Auditory Loop (Interruptible Speech)
Inspired by human conversation, we implemented a **Multi-threaded Auditory Feedback Loop**.
- **Interrupts:** If the robot is talking or moving and hears your voice, it instantly pauses to listen. 
- **Snappiness:** Uses a context-biased Whisper model and neural TTS (Piper) for a professional, responsive female character that sounds like sci-fi cinema.

---

## 🛠️ Technical Stack & Specs

| Component | Technology | Role |
|-----------|------------|------|
| **Lizard Brain** | **Go (Golang)** | High-concurrency orchestrator, I2C, and API layer. |
| **Visual Cortex** | **Rust + OpenCV** | Zero-latency object detection and facial recognition. |
| **Logic Engine** | **LeafcutterLLM** | Native layer-streaming engine for massive GGUF models. |
| **Memory** | **Dendrite (SQLite)** | Knowledge graph that remembers users, ranks, and facts. |
| **Hardware** | **Raspberry Pi 5** | 8GB RAM, 4WD I2C Chassis, Pan/Tilt Gimbal, OLED. |

---

## 🌟 Presentation Demo Highlights

### 🎮 Tactical Maneuvers
Say *"About Turn"*, and the robot executes a precision 180-degree tactical spin.

### 🦅 Birdwatch Mode
Activate *"Bird"* mode, and the robot enters an autonomous search pattern, sweeping the gimbal vertically and horizontally across the trees. If a bird is spotted, it tracks the motion and announces the discovery.

### 👥 Face Enrollment
Walk up to the robot. It detects you are a stranger and asks: *"Please Identify Yourself. What Rank?"* Once you answer, you are permanently etched into its **Dendrite memory** as a recognized ally.

### 💬 Natural Conversation
Skip the commands. Just say *"Pathfinder, move forward and tell me what you see."* The AI understands the context, actuates the motors, and describes its environment simultaneously.

---

## 🚀 The Vision: Intelligence for Everyone

THE-PATHFINDER-EYE proves that the future of robotics isn't in a data center—it's in the palm of your hand. By optimizing software to respect limited hardware, we've created a machine that can think, see, and act in the most remote corners of the world.

**"The synapse is not the neuron. The synapse is the connection. We are the connection."**

---
*Built by Xander | 2026 Production Release*  
*GitHub: [https://github.com/Alartist40/the-pathfinder-eye](https://github.com/Alartist40/the-pathfinder-eye)*


---

## Project Update (2026-06-18): v8.0 Sophistication Pass

The robot's brain has been hardened against:

1. Silent boot crash (cortex nil-deref).
2. Empty /stream responses (camera-feed.service wired to write
   `/tmp/vision_feed.jpg` for the brain's `/stream` route).
3. Audio policy violations (capture windows capped to the agreed
   3s wake / 5s command constants).
4. Catastrophic 70B model loads on 8GB hardware (`MinFreeMBFor70B`
   guard).
5. Drift between the v6.4 "Zero-Python Architecture" promise and
   the actually present `camera_feed.py` in go_brain/ (both moved
   to scripts/, __pycache__ cleaned).
6. Unredacted credentials in logs (24-pattern scanner + safeLogf).
7. Unbounded conversation context (Cynapse-style compactor archiver
   into DENDRITE).
8. Racy `sed -i` systemd unit edits (drop-in unit swap helper).
9. Bare sudo invocations (destructive-shell classifier with rank
   table).
10. LAN-wide-open HTTP API (bearer-token / loopback-only auth).

Where the v8.0 source diagram now stands:

| Layer              | Component                            | Notes |
|--------------------|--------------------------------------|-------|
| Voice pipeline     | whisper.cpp + espeak-ng              | AUDIO POLICY bound. |
| Cortex loop        | cortex.StartUnifiedAwareness         | Stable after `newCortex()`. |
| Memory             | DENDRITE graph + ContextDB           | compressor.go archives here. |
| Authority          | AuthorityManager                     | Re-rank on every retry. |
| Vision             | pathfinder-vision.service + camera-feed.service | Last frame always available. |
| Reasoning          | leafcutter.service (8B → 70B swap)    | Drop-in, never sed. |
| Safety             | redact.go + approval.go + http_auth.go | Bearer-token gate. |

For full detail, read handoff-the-pathfinder-eye.md (v8.0 section).

## Project Update (2026-06-19): v8.1 Core Stability & Bug Fixes

This pass focused on fixing code-level bugs that made the brain silently non-functional:

1. **Live API key in source** — The `.env` contained a real NVIDIA API key (`nvapi-...`). Replaced with placeholder.
2. **YOLOv5 stub (Rust)** — `detection.rs` returned `Vec::new()` — zero objects ever detected. Implemented full ONNX inference with NMS via `nms_boxes_batched_def`.
3. **VisionDB stub (Go)** — `vision.go` stored faces in a map that reset on restart. Replaced with persistent SQLite-backed VisionDB.
4. **CPU usage inverted** — `status_aware.go` calculated `cpu = 1 - fraction` wrong, reporting 95% usage at 5% load. Fixed to `cpu >= 0.8`.
5. **Dead volume gate** — `cortex.go` had `if false { isQuiet(...) }` — audio gate was permanently bypassed. Reactivated.
6. **Thread-unsafe whisper** — `voice.go` called whisper.cpp from goroutines with no lock. Added `whisperMu sync.Mutex`.
7. **Wake word false positives** — `strings.Contains` matched "destruction" inside "introduction". Rewrote to whole-word boundary matching.
8. **Motor blocking** — Command loop blocked on each movement. Wrapped in goroutines via `go startAIConversationLoop`.
9. **`commandBusy` leak** — Errors in conversation loop left `commandBusy=1` permanently. Fixed with `defer atomic.StoreInt32(&commandBusy, 0)`.
10. **Missing graceful shutdown** — `main.go` had no signal handler. Added `context.Context` + OS signal cancel.
11. **Test isolation failure** — Unit tests wrote to production DB. Changed to temp databases.
12. **`seekGimbal` race** — Unprotected map access in seek gimbal. Added `seekMu sync.Mutex`.
13. **`Close()` methods** — Dendrite, BirdWatchDB, VisionDB had no cleanup. Added explicit close.
14. **Face recognizer wiring** — `rust_vision/src/main.rs` now inits FaceRecognizer with graceful fallback.
15. **Gofmt** — All Go files reformatted with `gofmt -w`.

| Fix Area            | Files Changed                                    | Impact |
|---------------------|--------------------------------------------------|--------|
| Secret exposure     | `.env`, `.env.example`                           | No more leaked API keys in repo |
| Object detection    | `rust_vision/src/detection.rs`, `Cargo.toml`     | Robot now actually sees things |
| Face DB             | `go_brain/vision.go`                             | Face authority actually works |
| CPU monitoring      | `go_brain/status_aware.go`                       | Health reports are accurate |
| Audio gate          | `go_brain/cortex.go`                             | Quiet environment respected |
| Thread safety       | `go_brain/voice.go`, `vision.go`                 | No more whisper/gimbal races |
| Wake word           | `go_brain/voice_commands.go`, `cortex.go`        | No more false positives |
| Concurrency         | `go_brain/voice_commands.go`, `main.go`          | Motor commands non-blocking |
| Cleanup             | `go_brain/dendrite.go`, `birdwatch.go`, `vision.go` | Resources released on shutdown |
| Tests               | `dendrite_test.go`, `authority_test.go`          | Isolated, repeatable tests |

Build verification needed on Pi: `cd go_brain && go build -o ../brain .` & `cd rust_vision && cargo build --release`
