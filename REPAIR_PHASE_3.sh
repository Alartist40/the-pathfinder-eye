#!/bin/bash
# THE-PATHFINDER-EYE : Phase 3 AI BRAIN REPAIR
# Run this on the Raspberry Pi 5 to enable local LLM intelligence.

set -e

PROJECT_DIR="/home/pi/the-pathfinder-eye_ai"
MODELS_DIR="$PROJECT_DIR/models"
# We recommend Mistral-7B-Instruct-v0.2-GGUF for RPi 5
MODEL_URL="https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.2-GGUF/resolve/main/mistral-7b-instruct-v0.2.Q4_K_M.gguf"

echo "🧠 Starting Phase 3 AI Brain Setup..."

# 1. Install AI Runtimes (llama.cpp)
echo "[1/4] Installing llama.cpp for local inference..."
TEMP_BUILD="/tmp/llama_build"
rm -rf "$TEMP_BUILD"
git clone https://github.com/ggerganov/llama.cpp.git "$TEMP_BUILD"
cd "$TEMP_BUILD"
make -j$(nproc)
mkdir -p "$PROJECT_DIR/bin"
cp ./llama-cli "$PROJECT_DIR/bin/"

# 2. Download Leafcutter (Mistral) Model
echo "[2/4] Downloading local LLM model (~4.5GB)..."
mkdir -p "$MODELS_DIR"
if [ ! -f "$MODELS_DIR/mistral-7b-q4.gguf" ]; then
    wget -O "$MODELS_DIR/mistral-7b-q4.gguf" "$MODEL_URL"
else
    echo "✓ Model already exists."
fi

# 3. Setup Music Library
echo "[3/4] Initializing music library..."
mkdir -p "$PROJECT_DIR/music"
sudo apt install -y mpg123

# 4. Finalizing Go Brain
echo "[4/4] Rebuilding Master Brain v6.0..."
cd "$PROJECT_DIR/go_brain"
export PATH=$PATH:/usr/local/go/bin
go get github.com/mattn/go-sqlite3
go build -o brain main.go

echo "✅ Phase 3 Intelligence Setup Complete."
echo "The robot is now an autonomous wilderness agent."
