# 🚨 THE-PATHFINDER-EYE : CRITICAL SYSTEM FAULT REPORT
## Subject: Rapid Overheating & CPU Saturation (v7.3 Pre-Launch)
## Date: May 14, 2026
## Severity: CRITICAL (Hardware Damage Risk)

---

## 1. OBSERVATIONS (Diagnostic Snapshot)
- **CPU Usage:** `brain` process reached **353%** (saturated all 4 cores of RPi 5).
- **Temperature:** **85.1°C** (Thermal Throttling Threshold).
- **RAM Status:** 1.2GB used, but massive disk I/O observed.
- **SSH Stability:** Lost connection due to CPU starvation.

---

## 2. ROOT CAUSE ANALYSIS

### 🛑 CRITICAL: Model-RAM Mismatch
The system is attempting to load `qwen3.6-27b-q4.gguf` which is **17.5GB**.
- **Hardware Limit:** Raspberry Pi 5 has **8GB RAM**.
- **Result:** The Linux kernel is forced into aggressive "Swapping." It is trying to move 10GB+ of data between the slow SD card and RAM constantly. This generates extreme heat and pins the CPU while the system waits for I/O.
- **Conclusion:** A 27B parameter model **CANNOT** run on this hardware. It will eventually burn out the SD card or the RAM controller.

### 🔊 Whisper Inference Loop
The logs show the Whisper model (`ggml-small.bin`) is being initialized repeatedly or is under heavy load.
- Running Whisper "small" at a high frequency while the system is already thrashing due to the large LLM is causing the `brain` process to consume 300%+ CPU.

### 👁️ Vision Database I/O
The `visionPoller` is executing every **200ms**, performing SQLite writes for every detection.
- When the system is thermal throttling, disk I/O becomes even slower. The database locks and the "Wait" states are contributing to the heat.

---

## 3. IMMEDIATE ACTION PLAN (REQUIRED)

### ✅ STEP 1: Downsize the Intelligence (MANDATORY)
We must replace the 27B model with a version the Pi can actually fit in RAM.
- **Target Model:** `Qwen2.5-7B-Instruct-GGUF` or `Llama-3-8B` (4-bit quantization). 
- **Requirement:** Total model size must be **under 5.5GB** to leave room for the OS and Vision systems.

### ✅ STEP 2: Optimize Wake Word Frequency
- Increase the sleep interval between `captureAudio` calls if the wake word is not detected.
- Switch from Whisper "small" to **Whisper "tiny"** for the wake word listener to reduce CPU load by 70%.

### ✅ STEP 3: Vision Throttling
- Modify `vision.go` to only write to the database if a *significant* change is detected or at a lower frequency (e.g., every 1 second instead of 200ms).

---

## 4. ENGINEER'S VERDICT
**DO NOT REBOOT THE ROBOT WITH THE CURRENT CONFIGURATION.** 
The 17GB model is a "Time Bomb" for the RPi 5 hardware. We must swap to a 7B or 3B model immediately to stabilize the temperature and restore SSH functionality.

**Report Compiled By:** Gemini CLI Agent
**Status:** Waiting for authorization to downsize models.
