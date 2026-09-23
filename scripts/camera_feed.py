#!/usr/bin/env python3
"""
THE-PATHFINDER-EYE Camera Feed Service
Uses rpicam-still (libcamera) for CSI camera on Pi 5.
Writes frames to /tmp/vision_feed.jpg and detection stub to /tmp/detections.json
"""
import json
import time
import os
import sys
import subprocess

WIDTH = int(os.environ.get("CAMERA_WIDTH", 640))
HEIGHT = int(os.environ.get("CAMERA_HEIGHT", 480))
OUT_JPG = "/tmp/vision_feed.jpg"
OUT_JSON = "/tmp/detections.json"

def write_detections_stub():
    stub = {
        "timestamp": time.time(),
        "detections": []
    }
    try:
        with open(OUT_JSON, "w") as f:
            json.dump(stub, f)
    except Exception:
        pass

def capture_frame():
    tmp = OUT_JPG + ".tmp"
    cmd = [
        "rpicam-still",
        "--nopreview",
        "--timeout", "1",
        "--width", str(WIDTH),
        "--height", str(HEIGHT),
        "--output", tmp,
    ]
    try:
        subprocess.run(cmd, capture_output=True, timeout=5)
        if os.path.exists(tmp) and os.path.getsize(tmp) > 1000:
            os.replace(tmp, OUT_JPG)
            return True
    except Exception:
        pass
    return False

def main():
    print(f"Camera feed started: {WIDTH}x{HEIGHT} -> {OUT_JPG}")
    write_detections_stub()
    frame_interval = 0.5  # 2 FPS to keep CPU low
    last_time = time.time()
    fail_count = 0

    while True:
        now = time.time()
        if now - last_time < frame_interval:
            time.sleep(0.1)
            continue
        last_time = now

        ok = capture_frame()
        if ok:
            fail_count = 0
        else:
            fail_count += 1
            if fail_count > 5:
                print("Camera capture failing repeatedly", file=sys.stderr)
                time.sleep(2)

        if int(now) % 5 == 0:
            write_detections_stub()


try:
    main()
except KeyboardInterrupt:
    print("Camera feed stopped.")
    sys.exit(0)
