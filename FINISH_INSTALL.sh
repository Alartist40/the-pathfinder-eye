#!/bin/bash
set -e

echo "=================================="
echo "THE-PATHFINDER-EYE v7.3 FINAL SETUP"
echo "=================================="

PROJECT_DIR="/home/pi/the-pathfinder-eye_ai"
cd "$PROJECT_DIR"

echo ""
echo "Step 1: Setting up Python Virtual Environment..."
if [ ! -d "venv" ]; then
    python3 -m venv venv --system-site-packages
fi
source venv/bin/activate
pip install opencv-python-headless Pillow requests flask PyYAML python-dotenv smbus2 openai-whisper kokoro soundfile

echo ""
echo "Step 2: Installing Go dependencies..."
export PATH=$PATH:/usr/local/go/bin
cd go_brain
go mod tidy

echo ""
echo "Step 3: Building ARM64 optimized binary..."
go build -o pathfinder *.go

echo ""
echo "Step 4: Verifying build..."
if [ -f pathfinder ]; then
    echo "✅ Build successful"
    ls -lh pathfinder
else
    echo "❌ Build failed"
    exit 1
fi

echo ""
echo "Step 5: Running all 28 unit tests..."
go test -v ./...

echo ""
echo "=================================="
echo "✅ SETUP COMPLETE"
echo "=================================="
echo "Start robot: cd $PROJECT_DIR/go_brain && ./pathfinder"
echo ""

# Restart services to use new binary
sudo systemctl restart brain
sudo systemctl restart leafcutter

echo "Services restarted. Robot is LIVE."
