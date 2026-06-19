#!/bin/bash
# THE-PATHFINDER-EYE : Complete Raspberry Pi 5 Setup Script (v7.3-RECOVERY)
# Run this ON the Raspberry Pi 5 after flashing the SD card.
# This replaces REPAIR_PHASE_1.sh, REPAIR_PHASE_2.sh, REPAIR_PHASE_3.sh,
# and INTEGRATE_LEAFCUTTER.sh with a single unified flow.
#
# Usage:
#   chmod +x setup-pi.sh
#   ./setup-pi.sh

set -e

# =============================================================================
# CONFIGURATION
# =============================================================================
PROJECT_DIR="/home/pi/the-pathfinder-eye_ai"
LEAFCUTTER_DIR="/home/pi/LeafcutterLLM"
MODELS_DIR="$PROJECT_DIR/models"
DB_DIR="$PROJECT_DIR/db"
LOGS_DIR="$PROJECT_DIR/logs"
CONFIG_DIR="$PROJECT_DIR/config"

# Model files
WHISPER_MODEL="$MODELS_DIR/ggml-small.bin"
WHISPER_URL="https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin"

# Qwen3.5-9B-IQ4_NL — 5.3GB, confirmed working on 8GB Pi 5
LLM_MODEL="$MODELS_DIR/Qwen3.5-9B-IQ4_NL.gguf"
LLM_URL="https://huggingface.co/bartowski/Qwen_Qwen3.5-9B-GGUF/resolve/main/Qwen_Qwen3.5-9B-IQ4_NL.gguf?download=true"

LLAMA_CPP_DIR="/home/pi/llama.cpp"

# =============================================================================
# HELPERS
# =============================================================================
log_info()  { echo -e "\033[1;34m[INFO]\033[0m  $1"; }
log_warn()  { echo -e "\033[1;33m[WARN]\033[0m  $1"; }
log_error() { echo -e "\033[1;31m[ERROR]\033[0m $1"; }
log_success() { echo -e "\033[1;32m[OK]\033[0m    $1"; }

ensure_dir() {
    mkdir -p "$1"
}

check_cmd() {
    command -v "$1" >/dev/null 2>&1
}

# =============================================================================
# PHASE 0: SYSTEM PREP
# =============================================================================
log_info "THE-PATHFINDER-EYE v7.3 Recovery Setup"
log_info "=========================================="

if [[ "$EUID" -eq 0 ]]; then
    log_error "Do not run this script as root. Run as user 'pi' with passwordless sudo."
    exit 1
fi

if ! check_cmd sudo; then
    log_error "sudo is required."
    exit 1
fi

log_info "Updating package lists..."
sudo apt-get update

# =============================================================================
# PHASE 1: SYSTEM-LEVEL DEPENDENCIES
# =============================================================================
log_info "Installing system dependencies (this may take a while)..."

sudo apt-get install -y \
    i2c-tools libi2c-dev \
    espeak-ng alsa-utils libportaudio2 portaudio19-dev libasound2-dev \
    libopencv-dev opencv-data libopencv-contrib-dev \
    libsqlite3-dev sqlite3 \
    libopenblas-dev pkg-config \
    mpg123 \
    git curl wget htop vim \
    build-essential cmake make g++ gcc \
    python3 python3-pip python3-venv

# Enable I2C and Camera interfaces
log_info "Enabling I2C and Camera interfaces..."
sudo raspi-config nonint do_i2c 0 || log_warn "Could not auto-enable I2C. Run 'sudo raspi-config' manually."
sudo raspi-config nonint do_camera 0 || log_warn "Could not auto-enable Camera. Run 'sudo raspi-config' manually."

# Add user to required groups
sudo usermod -a -G i2c,spi,gpio,audio,video pi

# =============================================================================
# PHASE 2: GO RUNTIME
# =============================================================================
log_info "Checking Go installation..."
if ! check_cmd go; then
    log_info "Go not found. Installing Go 1.23..."
    GO_VER="1.23.4"
    GO_TAR="go${GO_VER}.linux-arm64.tar.gz"
    wget -q "https://go.dev/dl/${GO_TAR}" -O "/tmp/${GO_TAR}"
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "/tmp/${GO_TAR}"
    rm "/tmp/${GO_TAR}"
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    export PATH=$PATH:/usr/local/go/bin
    log_success "Go ${GO_VER} installed."
else
    log_success "Go already installed: $(go version)"
fi

# =============================================================================
# PHASE 3: RUST TOOLCHAIN
# =============================================================================
log_info "Checking Rust installation..."
if ! check_cmd cargo; then
    log_info "Rust not found. Installing via rustup..."
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain stable
    source "$HOME/.cargo/env"
    log_success "Rust installed."
else
    log_success "Rust already installed: $(rustc --version)"
fi

# =============================================================================
# PHASE 4: PROJECT DIRECTORIES
# =============================================================================
log_info "Creating project directories..."
ensure_dir "$PROJECT_DIR"
ensure_dir "$MODELS_DIR"
ensure_dir "$DB_DIR"
ensure_dir "$LOGS_DIR"
ensure_dir "$CONFIG_DIR"
ensure_dir "$PROJECT_DIR/resources"
ensure_dir "$PROJECT_DIR/music"
log_success "Directories ready."

# =============================================================================
# PHASE 5: WHISPER.CPP + GO BINDINGS (Voice)
# =============================================================================
log_info "Setting up Whisper.cpp (Speech-to-Text)..."

TEMP_WHISPER="/tmp/whisper_build_$$"
rm -rf "$TEMP_WHISPER"
git clone --depth 1 https://github.com/ggerganov/whisper.cpp.git "$TEMP_WHISPER"
cd "$TEMP_WHISPER"

# Build whisper static library for ARM64 Neon
make clean 2>/dev/null || true
make libwhisper.a -j$(nproc)

# Install libwhisper.a to a system path the Go bindings can find
sudo mkdir -p /usr/local/lib/whisper
sudo cp libwhisper.a /usr/local/lib/whisper/
sudo cp whisper.h /usr/local/lib/whisper/

# Download Whisper model
if [[ ! -f "$WHISPER_MODEL" ]]; then
    log_info "Downloading Whisper 'small' model (~500MB)..."
    wget --progress=bar:force -O "$WHISPER_MODEL" "$WHISPER_URL"
else
    log_success "Whisper model already exists."
fi

log_success "Whisper.cpp ready."

# =============================================================================
# PHASE 6: LEAFCUTTERLLM (Rust FFI + llama.cpp)
# =============================================================================
log_info "Setting up LeafcutterLLM (Rust FFI)..."

# Clone llama.cpp
if [[ ! -d "$LLAMA_CPP_DIR" ]]; then
    log_info "Cloning llama.cpp..."
    git clone https://github.com/ggerganov/llama.cpp.git "$LLAMA_CPP_DIR"
else
    log_success "llama.cpp already cloned."
fi

# Build llama.cpp shared libraries
if [[ ! -d "$LLAMA_CPP_DIR/build/bin" ]]; then
    log_info "Building llama.cpp (CPU backend)..."
    cd "$LLAMA_CPP_DIR"
    cmake -B build -DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF
    cmake --build build -j$(nproc)
    log_success "llama.cpp built."
else
    log_success "llama.cpp already built."
fi

# Clone LeafcutterLLM
if [[ ! -d "$LEAFCUTTER_DIR" ]]; then
    log_info "Cloning LeafcutterLLM to $LEAFCUTTER_DIR..."
    git clone https://github.com/Alartist40/LeafcutterLLM.git "$LEAFCUTTER_DIR" || {
        log_warn "Could not clone LeafcutterLLM. You may need to clone it manually."
    }
else
    log_success "LeafcutterLLM already cloned."
fi

# Build Rust leafcutter server
if [[ -d "$LEAFCUTTER_DIR/rust" ]]; then
    log_info "Building Leafcutter Rust server..."
    cd "$LEAFCUTTER_DIR/rust"
    export LLAMA_CPP_BUILD="$LLAMA_CPP_DIR/build"
    export LD_LIBRARY_PATH="$LLAMA_CPP_DIR/build/bin:$LD_LIBRARY_PATH"
    cargo build --release --bin leafcutter 2>/dev/null || {
        log_warn "Leafcutter Rust build failed. You may need to build it manually."
        log_warn "  cd $LEAFCUTTER_DIR/rust"
        log_warn "  export LLAMA_CPP_BUILD=$LLAMA_CPP_DIR/build"
        log_warn "  cargo build --release --bin leafcutter"
    }
else
    log_warn "LeafcutterLLM rust/ directory not found."
fi

# Download LLM model
if [[ ! -f "$LLM_MODEL" ]]; then
    log_warn "=========================================="
    log_warn "LLM model NOT found at:"
    log_warn "  $LLM_MODEL"
    log_warn ""
    log_warn "This is a ~5.3GB download. Options:"
    log_warn "  1. Let this script download it now"
    log_warn "  2. Copy it from another machine via USB/SCP"
    log_warn "=========================================="
    read -p "Download Qwen3.5-9B-IQ4_NL now? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        wget --progress=bar:force -O "$LLM_MODEL" "$LLM_URL"
    else
        log_warn "Skipping LLM download. You MUST place a .gguf model at:"
        log_warn "  $LLM_MODEL"
    fi
else
    log_success "LLM model already exists."
fi

log_success "LeafcutterLLM setup complete."

# =============================================================================
# PHASE 7: GO BRAIN BUILD
# =============================================================================
log_info "Building Go Brain (Master Orchestrator)..."

cd "$PROJECT_DIR/go_brain"

# Set CGO flags so the whisper bindings find libwhisper.a
export CGO_CFLAGS="-I/usr/local/lib/whisper"
export CGO_LDFLAGS="-L/usr/local/lib/whisper -lwhisper -lm -lopenblas"

# Set PATH for go
export PATH=$PATH:/usr/local/go/bin

# Tidy and build
go mod tidy
go build -o "$PROJECT_DIR/brain" main.go

log_success "Go Brain compiled: $PROJECT_DIR/brain"

# =============================================================================
# PHASE 8: RUST VISION ENGINE
# =============================================================================
log_info "Building Rust Vision Engine..."

cd "$PROJECT_DIR/rust_vision"

# OpenCV Rust bindings often need help finding the system OpenCV
export PKG_CONFIG_PATH=/usr/lib/aarch64-linux-gnu/pkgconfig:$PKG_CONFIG_PATH

# Build release binary
cargo build --release 2>/dev/null || {
    log_warn "Rust Vision build failed. This is common on first attempt."
    log_warn "Common fixes:"
    log_warn "  1. sudo apt-get install -y libclang-dev clang"
    log_warn "  2. export OPENCV_LINK_LIBS=opencv_core,opencv_imgproc,..."
    log_warn "  3. The ONNX Runtime crate is tricky on ARM64 — you may need to disable it."
}

if [[ -f "$PROJECT_DIR/rust_vision/target/release/rust_vision" ]]; then
    log_success "Rust Vision compiled."
else
    log_warn "Rust Vision binary not found. Vision features will be disabled."
fi

# =============================================================================
# PHASE 9: SYSTEMD SERVICES
# =============================================================================
log_info "Installing systemd services..."

# LeafcutterLLM service
sudo tee /etc/systemd/system/leafcutter.service > /dev/null <<EOF
[Unit]
Description=LeafcutterLLM Local AI Server
After=network.target

[Service]
Type=simple
User=pi
WorkingDirectory=$LEAFCUTTER_DIR
ExecStart=$LEAFCUTTER_DIR/rust/target/release/leafcutter server --model $LLM_MODEL --port 8081
Restart=on-failure
RestartSec=5
Environment="PATH=/usr/local/go/bin:/usr/bin:/bin"
Environment="LD_LIBRARY_PATH=$LLAMA_CPP_DIR/build/bin"

[Install]
WantedBy=multi-user.target
EOF

# Pathfinder Eye main service
sudo tee /etc/systemd/system/pathfinder-eye.service > /dev/null <<EOF
[Unit]
Description=THE-PATHFINDER-EYE Master Brain
After=network.target leafcutter.service
Wants=leafcutter.service

[Service]
Type=simple
User=pi
WorkingDirectory=$PROJECT_DIR/go_brain
ExecStart=$PROJECT_DIR/brain
Restart=on-failure
RestartSec=3
StandardOutput=append:$LOGS_DIR/go_brain.log
StandardError=append:$LOGS_DIR/go_brain.log
Environment="PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
log_success "Systemd services installed."

# =============================================================================
# PHASE 10: DEFAULT CONFIG
# =============================================================================
log_info "Creating default servo calibration..."

if [[ ! -f "$CONFIG_DIR/servo_calibration.json" ]]; then
cat > "$CONFIG_DIR/servo_calibration.json" <<'EOF'
{
  "version": "1.0",
  "calibrated_at": "2026-05-14T00:00:00Z",
  "pan": {
    "current": 90,
    "min": 0,
    "max": 180,
    "center": 90
  },
  "tilt": {
    "current": 110,
    "min": 0,
    "max": 180,
    "center": 110
  }
}
EOF
fi

# =============================================================================
# DONE
# =============================================================================
log_success "=========================================="
log_success "SETUP COMPLETE"
log_success "=========================================="
echo ""
echo "Next steps:"
echo "  1. Reboot:                    sudo reboot"
echo "  2. After reboot, start services:"
echo "     sudo systemctl start leafcutter"
echo "     sudo systemctl start pathfinder-eye"
echo "  3. Enable auto-start:"
echo "     sudo systemctl enable leafcutter"
echo "     sudo systemctl enable pathfinder-eye"
echo "  4. Check health:"
echo "     curl http://localhost:8080/health"
echo "     curl http://localhost:8081/health"
echo "  5. Open dashboard:"
echo "     http://$(hostname -I | awk '{print $1}'):8080"
echo ""
echo "Hardware test (run manually):"
echo "  cd $PROJECT_DIR && bash quick_test_hardware.sh"
echo ""
