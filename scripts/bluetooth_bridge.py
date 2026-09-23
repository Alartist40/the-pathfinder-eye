#!/usr/bin/env python3
"""
THE-PATHFINDER-EYE : Bluetooth SPP Transport Bridge
Listens on /dev/rfcomm0 (or specified serial port) and relays JSON commands
to the local Go Brain HTTP API (127.0.0.1:8080). Relays telemetry back over serial.

Protocol: Line-delimited JSON
Inbound:
  {"action": "move", "direction": "forward", "speed": 150}
  {"action": "camera", "axis": "pan", "val": 10}
  {"action": "think", "q": "what do you see?"}
  {"action": "light", "color": "green"}
  {"action": "status"}

Outbound:
  {"type": "telemetry", "status": "online", "uptime": "...", "timestamp": 123456}
  {"type": "response", "action": "think", "speech": "I see a path ahead"}
"""

import sys
import os
import time
import json
import threading
import urllib.request
import urllib.parse
import urllib.error

SERIAL_PORT = os.environ.get("BLUETOOTH_PORT", "/dev/rfcomm0")
BAUD_RATE = int(os.environ.get("BLUETOOTH_BAUD", "115200"))
GO_BRAIN_URL = os.environ.get("GO_BRAIN_URL", "http://127.0.0.1:8080")
AUTH_TOKEN = os.environ.get("PATHFINDER_EYE_HTTP_TOKEN", "pathfinder_secret_token")

def send_http(path, params=None):
    url = f"{GO_BRAIN_URL}{path}"
    if params:
        url += "?" + urllib.parse.urlencode(params)
    req = urllib.request.Request(url)
    if AUTH_TOKEN:
        req.add_header("Authorization", f"Bearer {AUTH_TOKEN}")
    try:
        with urllib.request.urlopen(req, timeout=4) as resp:
            data = resp.read()
            return json.loads(data.decode("utf-8")) if data else {"status": "ok"}
    except Exception as e:
        return {"error": str(e)}

def handle_command(cmd_obj):
    action = cmd_obj.get("action", "")
    if action == "move":
        direction = cmd_obj.get("direction", "stop")
        return send_http("/move", {"dir": direction})
    elif action == "camera":
        axis = cmd_obj.get("axis", "pan")
        val = str(cmd_obj.get("val", 0))
        return send_http("/camera", {"axis": axis, "val": val})
    elif action == "think":
        query = cmd_obj.get("q", "")
        return send_http("/ai/think", {"q": query})
    elif action == "status":
        return send_http("/health")
    else:
        return {"error": f"Unknown action: {action}"}

def telemetry_loop(ser):
    while getattr(ser, "is_open", True):
        try:
            stats = send_http("/health")
            if "error" not in stats:
                msg = json.dumps({"type": "telemetry", "data": stats}) + "\n"
                ser.write(msg.encode("utf-8"))
                ser.flush()
        except Exception:
            pass
        time.sleep(2.0)

def main():
    try:
        import serial
    except ImportError:
        print("[!] pyserial not installed. Run: pip install pyserial", file=sys.stderr)
        sys.exit(1)

    print(f"[*] Starting Bluetooth SPP Bridge on {SERIAL_PORT} @ {BAUD_RATE} baud...")
    while True:
        try:
            if not os.path.exists(SERIAL_PORT):
                print(f"[-] Waiting for {SERIAL_PORT} device node...", flush=True)
                time.sleep(3)
                continue

            with serial.Serial(SERIAL_PORT, BAUD_RATE, timeout=1) as ser:
                print(f"[+] Connected to {SERIAL_PORT}", flush=True)
                t = threading.Thread(target=telemetry_loop, args=(ser,), daemon=True)
                t.start()

                while ser.is_open:
                    line = ser.readline()
                    if not line:
                        continue
                    try:
                        cmd_text = line.decode("utf-8", errors="replace").strip()
                        if not cmd_text:
                            continue
                        cmd_obj = json.loads(cmd_text)
                        res = handle_command(cmd_obj)
                        out_msg = json.dumps({"type": "response", "action": cmd_obj.get("action"), "result": res}) + "\n"
                        ser.write(out_msg.encode("utf-8"))
                        ser.flush()
                    except json.JSONDecodeError:
                        err_msg = json.dumps({"type": "error", "message": "invalid_json"}) + "\n"
                        ser.write(err_msg.encode("utf-8"))
                        ser.flush()
        except Exception as e:
            print(f"[!] Serial connection error: {e}. Retrying in 2s...", flush=True)
            time.sleep(2)

if __name__ == "__main__":
    main()
