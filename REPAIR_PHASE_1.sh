#!/bin/bash
# THE-PATHFINDER-EYE : Phase 1 REPAIR SCRIPT (v1.2 Optimized)

set -e

PROJECT_DIR="/home/pi/the-pathfinder-eye_ai"
MODEL_DIR="$PROJECT_DIR/models"
# OPTIMIZATION: Switched to ggml-small.bin (5x faster, 100MB)
MODEL_URL="https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin"

echo "🛠️ Starting Phase 1 Voice System Repair & Optimization..."

# 1. Install System Dependencies
echo "[1/4] Installing PortAudio, espeak-ng, and ALSA tools..."
sudo apt update
sudo apt install -y espeak-ng alsa-utils libportaudio2 portaudio19-dev libasound2-dev make g++

# 2. Download Optimized Whisper Model
echo "[2/4] Downloading optimized Whisper ML model (small)..."
mkdir -p "$MODEL_DIR"
if [ ! -f "$MODEL_DIR/ggml-base.bin" ]; then
    wget -O "$MODEL_DIR/ggml-base.bin" "$MODEL_URL"
else
    echo "✓ Model already exists."
fi

# 3. Build Whisper.cpp
echo "[3/4] Compiling Whisper.cpp core..."
TEMP_WHISPER="/tmp/whisper_build"
rm -rf "$TEMP_WHISPER"
git clone https://github.com/ggerganov/whisper.cpp.git "$TEMP_WHISPER"
cd "$TEMP_WHISPER"
make libwhisper.a

# 4. Finalize Go Bindings
echo "[4/4] Finalizing Go workspace..."
cd "$PROJECT_DIR/go_brain"
go get github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper

echo "✅ Phase 1 Repair & Optimization Complete."
