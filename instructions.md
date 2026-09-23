# THE-PATHFINDER-EYE : User Instruction Manual
**Version:** v9.2-PRODUCTION  
**Hardware Platform:** Raspberry Pi 5 (8GB) + Yahboom Mecanum Chassis

---

## 🎙️ Voice Interaction & Audio Timing

Voice interactions strictly follow the **[AUDIO_POLICY.md](AUDIO_POLICY.md)** timing rules:

1. **Wake Word:** Say **`"Instruction"`** (or *"instruct"*, *"restruction"*).
   - *Wake Listening Window:* **3 seconds** of continuous ambient monitoring.
2. **Acknowledgment:** The robot flashes green LEDs and responds: **`"Standing by"`** (or *"Ready"*).
3. **Command Window:** You have a dedicated **5-second window** to speak your command naturally.

---

## 🧠 3-Tier Intelligent Voice Routing

Incoming speech is processed through a fast, deterministic 3-tier cascade:

```
Voice Input (Whisper Small STT)
  ├── 1. Needle 2 Intent Classifier (<100ms, confidence ≥ 0.5) ──> Instant Tool Execution
  ├── 2. Rule-Based Parser (<1ms, pattern matching) ─────────────> Hardware Action
  └── 3. LeafcutterLLM (Qwen3.5-2B, 3-8s local inference) ───────> Open Reasoning & Multi-turn Chat
```

---

## 🗣️ Voice Command Reference

### 🎮 Locomotion & Drive
* **"Move forward" / "Drive ahead"** — Moves forward (default speed: 150).
* **"Move backward" / "Reverse"** — Moves backward.
* **"Turn left" / "Step left"** — Turns/sidesteps left.
* **"Turn right" / "Step right"** — Turns/sidesteps right.
* **"About turn" / "Turn around"** — Executes a tactical 180° rotation.
* **"Stop" / "Halt" / "Stop now"** — Immediately stops all motors.

### 📷 Camera Gimbal Controls
* **"Look up"** — Tilts camera upward (tilt servo $\approx 170^\circ$).
* **"Look down"** — Tilts camera downward (tilt servo $\approx 30^\circ$).
* **"Look left"** — Pans camera left (pan servo $\approx 150^\circ$).
* **"Look right"** — Pans camera right (pan servo $\approx 30^\circ$).
* **"Look center"** — Centers gimbal (pan $90^\circ$, tilt $75^\circ$).

### 💡 LED Lighting
* **"Light red"** / **"Light green"** / **"Light blue"** / **"Light yellow"** — Sets all WS2812 chassis LEDs.
* **"Light off"** — Turns off all LEDs.

### 📜 Organizational Readings & Anthems
* **"Pathfinder Pledge"** — Recites the Pathfinder Pledge aloud via TTS.
* **"Pathfinder Law"** — Recites the 8 points of the Pathfinder Law.
* **"Pathfinder Aim"** — Recites the Pathfinder Aim.
* **"Pathfinder Motto"** — Recites the Pathfinder Motto.
* **"Pathfinder Song"** — Plays the Pathfinder Song audio soundtrack.
* **"Adventurer Pledge"** — Recites the Adventurer Pledge.
* **"Adventurer Law"** — Recites the Adventurer Law.
* **"Adventurer Song"** — Plays the Adventurer Song audio soundtrack.

### 🤖 Modes & System Authority
* **"Attention"** — Enters conversational AI loop (LeafcutterLLM / Qwen3.5-2B).
* **"Deep thought"** — Swaps to high-reasoning Qwen3.5-4B model.
* **"Follow" / "Track mode"** — Activates autonomous vision face/object tracking.
* **"Bird watch"** — Activates wildlife/bird observation logging into SQLite.
* **"Security mode"** — Activates perimeter PIR and vision motion tripwire.
* **"Japanese mode"** — Activates real-time Japanese $\leftrightarrow$ English voice translation loop.
* **"Instruction sleep" / "Exit"** — Deactivates active modes, shuts down motors, and returns to low-power idle.

---

## 🖥️ Desktop GUI Controller (Offline Capable)

A standalone Tkinter desktop interface is available in `desktop_gui/pathfinder_gui.py`.

### Features
* **Live Video Canvas:** Real-time MJPEG camera view from `/tmp/vision_feed.jpg`.
* **D-Pad Controls:** On-screen drive buttons with `WASD` / Arrow key bindings.
* **Gimbal Sliders:** Real-time Pan ($0^\circ-180^\circ$) and Tilt ($0^\circ-100^\circ$) controls with Centering.
* **Text Command Terminal:** Direct fallback input for noisy environments where voice commands fail.
* **Telemetry & Activity Logs:** Battery status, CPU temperature, mode indicators, and voice logs.

### Launching the GUI
```bash
# On your PC / laptop:
python3 desktop_gui/pathfinder_gui.py
```
* **WiFi Mode:** Enter `http://<robot-ip>:8080` (or `http://192.168.4.1:8080` on Pi WiFi Hotspot).
* **Bluetooth SPP Mode:** Select `/dev/rfcomm0` (Linux) or `COMx` (Windows) at 115200 baud.

---

## 📶 Bluetooth SPP Bridge (True Offline Serial Control)

The Pi includes an automatic serial bridge (`scripts/bluetooth_bridge.py`):
```bash
# Run the Bluetooth SPP listener:
python3 scripts/bluetooth_bridge.py
```
* **Protocol:** Line-delimited JSON over serial (`/dev/rfcomm0` @ 115200 baud).
* **Commands:**
  ```json
  {"action": "move", "direction": "forward", "speed": 150}
  {"action": "camera", "axis": "pan", "val": 10}
  {"action": "think", "q": "what do you see?"}
  ```

---

## 🌐 REST API Endpoints (Local & Remote)

The Master Brain serves HTTP REST endpoints on port `8080`:

| Endpoint | Method | Params | Description |
|---|---|---|---|
| `/health` | `GET` | — | System status, uptime, version |
| `/move` | `GET/POST` | `dir=forward\|backward\|left\|right\|stop` | Drive motors |
| `/camera` | `GET/POST` | `axis=pan\|tilt&val=<int>` | Gimbal pan/tilt with limit checks |
| `/stream` | `GET` | — | Real-time JPEG vision feed |
| `/ai/think` | `GET/POST` | `q=<text>` | Query the 3-tier intelligence stack |
