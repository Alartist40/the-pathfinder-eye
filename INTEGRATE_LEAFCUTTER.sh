#!/bin/bash
# THE-PATHFINDER-EYE : LeafcutterLLM Rust FFI Integration Script (v8.0)
# Builds the Rust leafcutter server (NOT the broken Go code).
# Uses llama.cpp shared libraries — LD_LIBRARY_PATH must be set.
#
# Usage:
#   chmod +x INTEGRATE_LEAFCUTTER.sh
#   ./INTEGRATE_LEAFCUTTER.sh

set -e

PROJECT_DIR="/home/pi/the-pathfinder-eye_ai"
LEAFCUTTER_DIR="/home/pi/LeafcutterLLM"
LLAMA_CPP_DIR="/home/pi/llama.cpp"
MODELS_DIR="$PROJECT_DIR/models"

# Qwen3.5-9B-IQ4_NL.gguf — 5.3GB, confirmed working on 8GB Pi 5
LLM_MODEL="$MODELS_DIR/Qwen3.5-9B-IQ4_NL.gguf"
LLM_URL="https://huggingface.co/bartowski/Qwen_Qwen3.5-9B-GGUF/resolve/main/Qwen_Qwen3.5-9B-IQ4_NL.gguf?download=true"

echo "🌿 Integrating LeafcutterLLM v0.9.0 (Rust FFI + llama.cpp)"

# ---------------------------------------------------------------------------
# 1. Build llama.cpp (if not already built)
# ---------------------------------------------------------------------------
if [[ ! -d "$LLAMA_CPP_DIR/build/bin" ]]; then
    echo "[1/5] Building llama.cpp..."
    if [[ ! -d "$LLAMA_CPP_DIR" ]]; then
        git clone https://github.com/ggerganov/llama.cpp.git "$LLAMA_CPP_DIR"
    fi
    cd "$LLAMA_CPP_DIR"
    cmake -B build -DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF
    cmake --build build -j$(nproc)
else
    echo "[1/5] llama.cpp already built."
fi

# ---------------------------------------------------------------------------
# 2. Build Leafcutter Rust Server
# ---------------------------------------------------------------------------
echo "[2/5] Building Leafcutter Rust server..."
cd "$LEAFCUTTER_DIR/rust"

# Point to the Pi-local llama.cpp build
export LLAMA_CPP_BUILD="$LLAMA_CPP_DIR/build"
export LD_LIBRARY_PATH="$LLAMA_CPP_DIR/build/bin:$LD_LIBRARY_PATH"

cargo build --release --bin leafcutter

echo "✅ Leafcutter binary: $LEAFCUTTER_DIR/rust/target/release/leafcutter"

# ---------------------------------------------------------------------------
# 3. Deploy Model
# ---------------------------------------------------------------------------
echo "[3/5] Deploying LLM model..."
mkdir -p "$MODELS_DIR"
if [[ ! -f "$LLM_MODEL" ]]; then
    echo "Downloading Qwen3.5-9B-IQ4_NL (~5.3GB)..."
    wget --progress=bar:force -O "$LLM_MODEL" "$LLM_URL"
else
    echo "Model already exists."
fi

# ---------------------------------------------------------------------------
# 4. Launch Leafcutter Server
# ---------------------------------------------------------------------------
echo "[4/5] Starting Leafcutter server on port 8081..."
sudo killall leafcutter 2>/dev/null || true

nohup "$LEAFCUTTER_DIR/rust/target/release/leafcutter" server \
    --model "$LLM_MODEL" \
    --port 8081 \
    > "$PROJECT_DIR/logs/leafcutter.log" 2>&1 &

echo "   PID: $!"
sleep 2

# Quick health check
curl -s http://localhost:8081/health || {
    echo "⚠️  Server may not be ready yet. Check logs: $PROJECT_DIR/logs/leafcutter.log"
}

# ---------------------------------------------------------------------------
# 5. Rebuild Go Brain
# ---------------------------------------------------------------------------
echo "[5/5] Building Go Brain..."
cd "$PROJECT_DIR/go_brain"
export PATH=$PATH:/usr/local/go/bin
export CGO_CFLAGS="-I/usr/local/lib/whisper"
export CGO_LDFLAGS="-L/usr/local/lib/whisper -lwhisper -lm -lopenblas"

go mod tidy
go build -o "$PROJECT_DIR/brain" main.go

echo ""
echo "✅ [SUCCESS] LeafcutterLLM integrated."
echo "   Model:   $LLM_MODEL"
echo "   Server:  http://localhost:8081"
echo "   Brain:   $PROJECT_DIR/brain"
echo ""
echo "To start everything:"
echo "   sudo systemctl start leafcutter"
echo "   sudo systemctl start pathfinder-eye"
