#!/bin/bash
# THE-PATHFINDER-EYE Absolute Audio Calibration (V7.4 Verified High Sensitivity)

# 1. Kill any runaway audio processes
sudo killall -9 arecord rec sox aplay piper 2>/dev/null || true

# 2. Disable Hardware Loopback (The Squeal Fix)
amixer -D hw:0 cset numid=3 off

# 3. Disable Auto Gain (The Static Fix)
amixer -D hw:0 cset numid=9 off

# 4. Maximize Hearing Sensitivity (The "Amazing" level user loved)
# numid=8 32 is roughly 90% gain. High but below distortion clipping.
amixer -D hw:0 cset numid=8 32

# 5. High Volume Output
amixer -D hw:0 cset numid=6 33
amixer -D hw:0 cset numid=5 on
