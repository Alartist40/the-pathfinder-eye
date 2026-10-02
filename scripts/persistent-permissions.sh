#!/bin/bash
# THE-PATHFINDER-EYE Persistent Permissions Setup
set -e

# 1. Ensure I2C device permissions for non-root pi user
if [ -e /dev/i2c-1 ]; then
    chmod 666 /dev/i2c-1 2>/dev/null || true
fi

# 2. Ensure camera video devices are accessible
for dev in /dev/video*; do
    if [ -e "$dev" ]; then
        chmod 666 "$dev" 2>/dev/null || true
    fi
done

# 3. Ensure log and DB directories exist with proper ownership
mkdir -p /home/pi/the-pathfinder-eye_ai/logs
mkdir -p /home/pi/the-pathfinder-eye_ai/db
chown -R pi:pi /home/pi/the-pathfinder-eye_ai/logs /home/pi/the-pathfinder-eye_ai/db 2>/dev/null || true

# 4. Clean stale temporary audio files
rm -f /tmp/speech.wav /tmp/capture.wav /tmp/vision_feed.jpg
