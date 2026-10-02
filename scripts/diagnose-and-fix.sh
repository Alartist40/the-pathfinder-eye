#!/bin/bash
# THE-PATHFINDER-EYE Boot-Time Auto-Build & Diagnostic Script
set -e

PROJECT_DIR="/home/pi/the-pathfinder-eye_ai"
BRAIN_BIN="$PROJECT_DIR/brain"
GO_DIR="$PROJECT_DIR/go_brain"

export PATH=$PATH:/usr/local/go/bin:/usr/bin:/bin
export CGO_CFLAGS="-I/usr/local/lib/whisper"
export CGO_LDFLAGS="-L/usr/local/lib/whisper -lwhisper -lm -lopenblas"

# 1. Check if brain binary is missing or if any Go source file is newer than binary
NEED_BUILD=0
if [ ! -f "$BRAIN_BIN" ]; then
    NEED_BUILD=1
else
    # Compare timestamps of Go sources against the binary
    for src in "$GO_DIR"/*.go; do
        if [ "$src" -nt "$BRAIN_BIN" ]; then
            NEED_BUILD=1
            break
        fi
    done
fi

# 2. Recompile on-device if necessary
if [ "$NEED_BUILD" -eq 1 ]; then
    echo "[$(date)] Auto-building Go Brain binary..."
    if which go > /dev/null 2>&1 || [ -x /usr/local/go/bin/go ]; then
        cd "$GO_DIR"
        go build -o "$BRAIN_BIN" .
        cp -f "$BRAIN_BIN" "$GO_DIR/brain"
        chmod +x "$BRAIN_BIN" "$GO_DIR/brain"
        echo "[$(date)] Auto-build completed successfully."
    else
        echo "[$(date)] Go compiler not found, using existing binary."
    fi
fi

# 3. Ensure binary is executable
if [ -f "$BRAIN_BIN" ]; then
    chmod +x "$BRAIN_BIN"
fi

# 4. Run audio calibration and unmuting
if [ -x "$PROJECT_DIR/scripts/set_audio.sh" ]; then
    bash "$PROJECT_DIR/scripts/set_audio.sh" 2>/dev/null || true
fi
