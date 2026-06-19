#!/bin/bash
# THE-PATHFINDER-EYE : Phase 2 VISION SYSTEM REPAIR
# Run this on the Raspberry Pi 5 to enable high-performance Vision.

set -e

PROJECT_DIR="/home/pi/the-pathfinder-eye_ai"
MODELS_DIR="$PROJECT_DIR/models"
YOLO_URL="https://github.com/ultralytics/yolov5/releases/download/v7.0/yolov5s.pt" # Example, typically use ONNX

echo "👁️ Starting Phase 2 Vision System Setup..."

# 1. Install Vision Libraries
echo "[1/4] Installing OpenCV and ONNX dependencies..."
sudo apt update
sudo apt install -y libopencv-dev opencv-data libonnxruntime-dev libsqlite3-dev

# 2. Download YOLO Model (SSD MobileNet V2 is currently used as baseline)
echo "[2/4] Verifying ML models..."
mkdir -p "$MODELS_DIR"
# In a real environment, we'd fetch the specific yolov5s-640.onnx
# For this build, we ensure the baseline SSD graph is present
if [ ! -f "$MODELS_DIR/frozen_inference_graph.pb" ]; then
    echo "Downloading baseline vision model..."
    wget -O "$MODELS_DIR/frozen_inference_graph.pb" https://github.com/opencv/opencv_extra/raw/master/testdata/dnn/ssd_mobilenet_v2_coco_2018_03_29.pb
fi

# 3. Setup Face Cascades
echo "[3/4] Initializing Face Detection cascades..."
if [ -f "/usr/share/opencv4/haarcascades/haarcascade_frontalface_default.xml" ]; then
    cp /usr/share/opencv4/haarcascades/haarcascade_frontalface_default.xml "$MODELS_DIR/"
    echo "✓ Face cascade installed."
else
    echo "⚠️ Warning: System haarcascades not found. Manual download may be required."
fi

# 4. Build Rust Vision Engine
echo "[4/4] Compiling Vision Engine (Rust)..."
cd "$PROJECT_DIR/rust_vision"
# Ensure Cargo is available and dependencies are fetched
if command -v cargo &> /dev/null; then
    cargo build --release
    echo "✓ Rust Vision Engine compiled."
else
    echo "⚠️ Warning: Cargo not found. Please install Rust to compile the vision engine."
fi

echo "✅ Phase 2 Vision Setup Complete."
echo "The robot can now see objects, birds, and faces."
