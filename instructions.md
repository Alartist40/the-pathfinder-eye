# THE-PATHFINDER-EYE : User Instruction Manual

## 🎙️ Voice Interaction
To command the robot, you must first use the **Wake Word**.

1.  Say: **"Instruction"**
2.  The robot will reply: **"Standing by."**
3.  You then have **7 seconds** to say one of the following commands:

### 🎮 Locomotion Commands
*   **"Move forward"** - Moves the robot ahead.
*   **"Move back"** - Reverses the robot.
*   **"Move left"** - Sidesteps (crab-walk) to the left.
*   **"Move right"** - Sidesteps (crab-walk) to the right.
*   **"Rotate left"** - Spins the robot to the left.
*   **"Rotate right"** - Spins the robot to the right.
*   **"About turn"** - Performs a tactical 180-degree turn to the right.

### 📐 Eye Calibration Commands
*   **"Eye tilt [angle]"** - Set vertical angle (e.g., "Eye tilt 110"). 
    *   *Lower (30) = Floor | Higher (180) = Ceiling*
*   **"Eye rotate [angle]"** - Set horizontal angle (e.g., "Eye rotate 90").
    *   *90 = Center | 0 = Left | 180 = Right*

### 🎵 Resource & Organization Commands
*   **"Pathfinder Song"** - Plays the Pathfinder anthem MP3.
*   **"Pathfinder Pledge"** - Recites the Pathfinder pledge via TTS.
*   **"Pathfinder Law"** - Recites the Pathfinder law.
*   **"Pathfinder Aim"** - Recites the Pathfinder aim.
*   **"Pathfinder Motto"** - Recites the Pathfinder motto.
*   **"Adventurer Song"** - Plays the Adventurer anthem.
*   **"Adventurer Law"** - Recites the Adventurer law.
*   **"Adventurer Pledge"** - Recites the Adventurer pledge.

### 🧠 Advanced AI, Vision & Memory
*   **"Attention"** - Activates the **LeafcutterLLM** (Local AI). 
    *   In this mode, you can talk naturally to the robot.
    *   Say **"Instruction sleep"** to turn off the AI and save RAM.
*   **"Follow"** - Engages the Vision Tracking engine.
*   **"Register new leader [name]"** - Tactically register a new leader into the Neural-Link memory.
*   **"Register new masterguide [name]"** - Register a masterguide to grant them system authority.

---

## 💻 Web Dashboard
Access the visual feed and status at:
👉 **`http://<robot-ip>:8080`**

---

## v9.0 Update: Needle 2 Tool & Intent Engine + AntiDoom Conversation Quality

The robot now uses two complementary ML upgrades:

1. **Needle 2 Unified Tool & Intent Engine** (45M params, 14 MB binary, ~28 MB RAM session) to understand voice commands and execute hardware tools. Runs locally on port 8082 with sub-10ms response time.

2. **AntiDoom** FTPO pipeline for the Ministral-3B conversation model. Eliminates repetition loops in the "Attention" AI conversation mode by training a LoRA adapter on Final Token Preference Optimization pairs. Installed in the `antidoom/` directory. See `antidoom/README.md`.

**Status:** v9.0 NEEDLE 2 + ANTIDOOM 🟢

---

## Needle 2 Voice Recognition & Tool Calling

The robot now uses **Needle 2** (45M params, 14 MB) to understand
voice commands and direct hardware tool calls. Instead of requiring exact keywords, Needle 2 interprets
natural speech, paraphrases, and parameters directly into structured tool executions.

**What this means for users:** You can say commands naturally.
"Drive ahead fast" works for moving forward. "Recite the pledge" works
for reciting the pledge. "Look up" adjusts camera gimbal.

**Thinking mode:** Say "Attention" to activate the AI conversation
loop (LeafcutterLLM/Ministral), just like before. Say "Instruction sleep" to
deactivate.
