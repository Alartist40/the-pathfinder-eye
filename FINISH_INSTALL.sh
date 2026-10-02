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
export CGO_CFLAGS="-I/usr/local/lib/whisper"
export CGO_LDFLAGS="-L/usr/local/lib/whisper -lwhisper -lm -lopenblas"
cd go_brain
go mod tidy

echo ""
echo "Step 3: Building ARM64 optimized binary..."
go build -o "$PROJECT_DIR/brain" .
cp "$PROJECT_DIR/brain" "$PROJECT_DIR/go_brain/pathfinder"

echo ""
echo "Step 4: Verifying build..."
if [ -f "$PROJECT_DIR/brain" ]; then
    echo "✅ Build successful"
    ls -lh "$PROJECT_DIR/brain"
else
    echo "❌ Build failed"
    exit 1
fi

echo ""
echo "Step 5: Running unit tests..."
go test -v ./...

echo ""
echo "=================================="
echo "✅ SETUP COMPLETE"
echo "=================================="
echo "Start robot: cd $PROJECT_DIR/go_brain && ./pathfinder"
echo ""

# Restart services to use new binary
sudo systemctl restart pathfinder-eye || true
sudo systemctl restart leafcutter || true

echo "Services restarted. Robot is LIVE."
