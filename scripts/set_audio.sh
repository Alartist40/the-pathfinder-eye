#!/bin/bash
# THE-PATHFINDER-EYE Absolute Audio Calibration (Multi-Card RPi 5 Compatible)

# 1. Kill any runaway audio processes
sudo killall -9 arecord rec sox aplay piper 2>/dev/null || true

# 2. Iterate through all sound cards (0-4) to configure mixer levels and unmute
for card in 0 1 2 3 4; do
    if [ -d "/proc/asound/card$card" ]; then
        # Unmute & set volume on common controls
        amixer -c $card set Master 100% unmute 2>/dev/null || true
        amixer -c $card set Speaker 100% unmute 2>/dev/null || true
        amixer -c $card set PCM 100% unmute 2>/dev/null || true
        amixer -c $card set Headphone 100% unmute 2>/dev/null || true
        amixer -c $card set Mic 90% unmute 2>/dev/null || true
        amixer -c $card set Capture 90% unmute 2>/dev/null || true

        # Disable Loopback and Auto Gain where supported
        amixer -c $card cset numid=3 off 2>/dev/null || true
        amixer -c $card cset numid=9 off 2>/dev/null || true
        amixer -c $card cset numid=8 32 2>/dev/null || true
        amixer -c $card cset numid=6 33 2>/dev/null || true
        amixer -c $card cset numid=5 on 2>/dev/null || true
    fi
done
