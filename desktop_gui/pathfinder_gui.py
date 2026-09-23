#!/usr/bin/env python3
"""
THE-PATHFINDER-EYE : Desktop Controller GUI
Lightweight, offline-capable Python/Tkinter GUI for Pathfinder Eye.
Supports:
  - Live Camera Stream (MJPEG via HTTP / WiFi-Direct or frame polling)
  - D-Pad Robot Drive Controls (Keyboard WASD + on-screen buttons)
  - Gimbal Pan/Tilt Sliders & Presets
  - Text Command Input (Fallback for voice commands)
  - Live Voice & Action Logs
  - Telemetry & Status Bar (Battery, CPU Temp, Connection State)
  - Dual Transport: Bluetooth SPP (Serial /dev/rfcomm0, COMx) or WiFi/HTTP
"""

import sys
import os
import json
import time
import threading
import urllib.request
import urllib.parse
import urllib.error
import tkinter as tk
from tkinter import ttk, scrolledtext, messagebox

try:
    from PIL import Image, ImageTk
    HAS_PIL = True
except ImportError:
    HAS_PIL = False

try:
    import serial
    HAS_SERIAL = True
except ImportError:
    HAS_SERIAL = False


class PathfinderApp:
    def __init__(self, root):
        self.root = root
        self.root.title("Pathfinder Eye — Robot Control Center")
        self.root.geometry("860x720")
        self.root.minsize(780, 600)

        self.ser = None
        self.transport_mode = tk.StringVar(value="HTTP")
        self.robot_host = tk.StringVar(value="http://192.168.4.1:8080")
        self.serial_port = tk.StringVar(value="/dev/rfcomm0")
        self.auth_token = tk.StringVar(value="pathfinder_secret_token")
        self.pan_val = tk.IntVar(value=90)
        self.tilt_val = tk.IntVar(value=75)
        self.is_connected = False
        self.running = True

        self._build_ui()
        self._bind_keys()
        self._start_background_threads()

    def _build_ui(self):
        # 1. Connection Header Bar
        conn_frame = ttk.LabelFrame(self.root, text=" Connection Settings ")
        conn_frame.pack(fill="x", padx=10, pady=5)

        ttk.Label(conn_frame, text="Transport:").pack(side="left", padx=5)
        ttk.Radiobutton(conn_frame, text="WiFi / HTTP", variable=self.transport_mode, value="HTTP", command=self._on_transport_change).pack(side="left", padx=5)
        ttk.Radiobutton(conn_frame, text="Bluetooth SPP", variable=self.transport_mode, value="SERIAL", command=self._on_transport_change).pack(side="left", padx=5)

        self.lbl_target = ttk.Label(conn_frame, text="Host URL:")
        self.lbl_target.pack(side="left", padx=5)
        self.ent_target = ttk.Entry(conn_frame, textvariable=self.robot_host, width=24)
        self.ent_target.pack(side="left", padx=5)

        self.btn_connect = ttk.Button(conn_frame, text="Connect", command=self.toggle_connection)
        self.btn_connect.pack(side="left", padx=10)

        # 2. Main Workspace (Paned Left/Right)
        main_paned = ttk.PanedWindow(self.root, orient="horizontal")
        main_paned.pack(fill="both", expand=True, padx=10, pady=5)

        # Left Column: Camera Feed
        left_frame = ttk.Frame(main_paned)
        main_paned.add(left_frame, weight=3)

        cam_box = ttk.LabelFrame(left_frame, text=" Live Camera Feed ")
        cam_box.pack(fill="both", expand=True, padx=5, pady=5)

        self.cam_canvas = tk.Canvas(cam_box, bg="#1a1a1a", width=480, height=360)
        self.cam_canvas.pack(fill="both", expand=True, padx=5, pady=5)
        self.cam_image_tk = None

        # Right Column: Controls & Commands
        right_frame = ttk.Frame(main_paned)
        main_paned.add(right_frame, weight=2)

        # D-Pad Control
        dpad_box = ttk.LabelFrame(right_frame, text=" Drive Controls (WASD / Arrows) ")
        dpad_box.pack(fill="x", padx=5, pady=5)

        dp_grid = ttk.Frame(dpad_box)
        dp_grid.pack(pady=5)

        btn_fwd = ttk.Button(dp_grid, text="▲ Forward (W)", command=lambda: self.send_move("forward"))
        btn_fwd.grid(row=0, column=1, padx=4, pady=4)
        btn_left = ttk.Button(dp_grid, text="◄ Left (A)", command=lambda: self.send_move("left"))
        btn_left.grid(row=1, column=0, padx=4, pady=4)
        btn_stop = ttk.Button(dp_grid, text="■ STOP (Space)", command=lambda: self.send_move("stop"))
        btn_stop.grid(row=1, column=1, padx=4, pady=4)
        btn_right = ttk.Button(dp_grid, text="► Right (D)", command=lambda: self.send_move("right"))
        btn_right.grid(row=1, column=2, padx=4, pady=4)
        btn_back = ttk.Button(dp_grid, text="▼ Backward (S)", command=lambda: self.send_move("backward"))
        btn_back.grid(row=2, column=1, padx=4, pady=4)

        # Gimbal Pan/Tilt
        gimbal_box = ttk.LabelFrame(right_frame, text=" Gimbal Pan / Tilt ")
        gimbal_box.pack(fill="x", padx=5, pady=5)

        g_grid = ttk.Frame(gimbal_box)
        g_grid.pack(fill="x", padx=5, pady=5)

        ttk.Label(g_grid, text="Pan:").grid(row=0, column=0, sticky="w")
        self.slider_pan = ttk.Scale(g_grid, from_=0, to=180, variable=self.pan_val, command=self._on_pan_change)
        self.slider_pan.grid(row=0, column=1, sticky="ew", padx=5)
        self.lbl_pan_val = ttk.Label(g_grid, text="90°")
        self.lbl_pan_val.grid(row=0, column=2)

        ttk.Label(g_grid, text="Tilt:").grid(row=1, column=0, sticky="w")
        self.slider_tilt = ttk.Scale(g_grid, from_=0, to=100, variable=self.tilt_val, command=self._on_tilt_change)
        self.slider_tilt.grid(row=1, column=1, sticky="ew", padx=5)
        self.lbl_tilt_val = ttk.Label(g_grid, text="75°")
        self.lbl_tilt_val.grid(row=1, column=2)

        btn_center = ttk.Button(g_grid, text="Center Gimbal", command=self.center_gimbal)
        btn_center.grid(row=2, column=0, columnspan=3, pady=4)
        g_grid.columnconfigure(1, weight=1)

        # 3. Bottom Area: Command Console & Voice Log
        bottom_frame = ttk.Frame(self.root)
        bottom_frame.pack(fill="both", expand=True, padx=10, pady=5)

        cmd_box = ttk.LabelFrame(bottom_frame, text=" Text Command Input (Fallback for Voice) ")
        cmd_box.pack(fill="x", pady=2)

        self.ent_cmd = ttk.Entry(cmd_box)
        self.ent_cmd.pack(side="left", fill="x", expand=True, padx=5, pady=5)
        self.ent_cmd.bind("<Return>", lambda e: self.send_text_command())

        btn_send = ttk.Button(cmd_box, text="Send Command", command=self.send_text_command)
        btn_send.pack(side="right", padx=5, pady=5)

        log_box = ttk.LabelFrame(bottom_frame, text=" Activity & Voice Logs ")
        log_box.pack(fill="both", expand=True, pady=2)

        self.txt_log = scrolledtext.ScrolledText(log_box, height=6, bg="#111827", fg="#10b981", insertbackground="white", font=("Monospace", 9))
        self.txt_log.pack(fill="both", expand=True, padx=5, pady=5)

        # 4. Status Bar
        self.status_bar = ttk.Frame(self.root, relief="sunken")
        self.status_bar.pack(fill="x", side="bottom")

        self.lbl_conn_status = ttk.Label(self.status_bar, text="Status: Disconnected", foreground="red")
        self.lbl_conn_status.pack(side="left", padx=8)

        self.lbl_telemetry = ttk.Label(self.status_bar, text="Battery: -- | Temp: -- | AI: Ready")
        self.lbl_telemetry.pack(side="right", padx=8)

    def _bind_keys(self):
        self.root.bind("<w>", lambda e: self.send_move("forward"))
        self.root.bind("<s>", lambda e: self.send_move("backward"))
        self.root.bind("<a>", lambda e: self.send_move("left"))
        self.root.bind("<d>", lambda e: self.send_move("right"))
        self.root.bind("<space>", lambda e: self.send_move("stop"))
        self.root.bind("<Up>", lambda e: self.send_move("forward"))
        self.root.bind("<Down>", lambda e: self.send_move("backward"))
        self.root.bind("<Left>", lambda e: self.send_move("left"))
        self.root.bind("<Right>", lambda e: self.send_move("right"))

    def _on_transport_change(self):
        mode = self.transport_mode.get()
        if mode == "HTTP":
            self.lbl_target.config(text="Host URL:")
            self.ent_target.config(textvariable=self.robot_host)
        else:
            self.lbl_target.config(text="Serial Port:")
            self.ent_target.config(textvariable=self.serial_port)

    def log(self, text):
        ts = time.strftime("[%H:%M:%S] ")
        self.txt_log.insert("end", ts + text + "\n")
        self.txt_log.see("end")

    def toggle_connection(self):
        if not self.is_connected:
            mode = self.transport_mode.get()
            if mode == "SERIAL":
                if not HAS_SERIAL:
                    messagebox.showerror("Error", "pyserial is not installed. Please run: pip install pyserial")
                    return
                port = self.serial_port.get()
                try:
                    self.ser = serial.Serial(port, 115200, timeout=1)
                    self.is_connected = True
                    self.btn_connect.config(text="Disconnect")
                    self.lbl_conn_status.config(text=f"Connected: {port}", foreground="green")
                    self.log(f"Connected to robot via Bluetooth Serial ({port})")
                except Exception as e:
                    messagebox.showerror("Connection Error", f"Failed to open {port}: {e}")
                    return
            else:
                self.is_connected = True
                self.btn_connect.config(text="Disconnect")
                self.lbl_conn_status.config(text=f"Connected: {self.robot_host.get()}", foreground="green")
                self.log(f"Connected to robot via HTTP ({self.robot_host.get()})")
        else:
            self.is_connected = False
            if self.ser:
                try:
                    self.ser.close()
                except Exception:
                    pass
                self.ser = None
            self.btn_connect.config(text="Connect")
            self.lbl_conn_status.config(text="Status: Disconnected", foreground="red")
            self.log("Disconnected from robot.")

    def _send_payload(self, cmd_dict, http_path, http_params):
        if not self.is_connected:
            self.log("Cannot send command: not connected.")
            return

        mode = self.transport_mode.get()
        if mode == "SERIAL" and self.ser:
            try:
                line = json.dumps(cmd_dict) + "\n"
                self.ser.write(line.encode("utf-8"))
                self.ser.flush()
            except Exception as e:
                self.log(f"Serial write error: {e}")
        else:
            def req_worker():
                base = self.robot_host.get().rstrip("/")
                url = f"{base}{http_path}"
                if http_params:
                    url += "?" + urllib.parse.urlencode(http_params)
                req = urllib.request.Request(url)
                token = self.auth_token.get()
                if token:
                    req.add_header("Authorization", f"Bearer {token}")
                try:
                    with urllib.request.urlopen(req, timeout=5) as resp:
                        res = resp.read().decode("utf-8")
                        if res:
                            data = json.loads(res)
                            if "speech" in data and data["speech"]:
                                self.root.after(0, lambda: self.log(f"Robot: {data['speech']}"))
                except Exception as e:
                    self.root.after(0, lambda: self.log(f"HTTP Error: {e}"))

            threading.Thread(target=req_worker, daemon=True).start()

    def send_move(self, direction):
        self.log(f"Drive: {direction}")
        self._send_payload(
            {"action": "move", "direction": direction, "speed": 150},
            "/move",
            {"dir": direction}
        )

    def _on_pan_change(self, val):
        v = int(float(val))
        self.lbl_pan_val.config(text=f"{v}°")
        self._send_payload(
            {"action": "camera", "axis": "pan", "val": v},
            "/camera",
            {"axis": "pan", "val": v}
        )

    def _on_tilt_change(self, val):
        v = int(float(val))
        self.lbl_tilt_val.config(text=f"{v}°")
        self._send_payload(
            {"action": "camera", "axis": "tilt", "val": v},
            "/camera",
            {"axis": "tilt", "val": v}
        )

    def center_gimbal(self):
        self.pan_val.set(90)
        self.tilt_val.set(75)
        self.lbl_pan_val.config(text="90°")
        self.lbl_tilt_val.config(text="75°")
        self._send_payload(
            {"action": "camera", "axis": "center", "pan": 90, "tilt": 75},
            "/camera",
            {"axis": "pan", "val": 0}
        )

    def send_text_command(self):
        text = self.ent_cmd.get().strip()
        if not text:
            return
        self.ent_cmd.delete(0, "end")
        self.log(f"You: {text}")
        self._send_payload(
            {"action": "think", "q": text},
            "/ai/think",
            {"q": text}
        )

    def _start_background_threads(self):
        threading.Thread(target=self._camera_stream_loop, daemon=True).start()
        threading.Thread(target=self._telemetry_poll_loop, daemon=True).start()

    def _camera_stream_loop(self):
        while self.running:
            if self.is_connected and HAS_PIL:
                try:
                    base = self.robot_host.get().rstrip("/")
                    url = f"{base}/stream"
                    req = urllib.request.Request(url)
                    token = self.auth_token.get()
                    if token:
                        req.add_header("Authorization", f"Bearer {token}")
                    with urllib.request.urlopen(req, timeout=2) as resp:
                        data = resp.read()
                        if data:
                            from io import BytesIO
                            img = Image.open(BytesIO(data))
                            cw = self.cam_canvas.winfo_width() or 480
                            ch = self.cam_canvas.winfo_height() or 360
                            img = img.resize((max(10, cw), max(10, ch)), Image.Resampling.BILINEAR)
                            photo = ImageTk.PhotoImage(img)
                            self.cam_image_tk = photo
                            self.root.after(0, lambda p=photo: self._update_canvas(p))
                except Exception:
                    pass
            time.sleep(0.1)

    def _update_canvas(self, photo):
        self.cam_canvas.delete("all")
        self.cam_canvas.create_image(0, 0, anchor="nw", image=photo)

    def _telemetry_poll_loop(self):
        while self.running:
            if self.is_connected:
                mode = self.transport_mode.get()
                if mode == "HTTP":
                    try:
                        base = self.robot_host.get().rstrip("/")
                        req = urllib.request.Request(f"{base}/health")
                        with urllib.request.urlopen(req, timeout=3) as resp:
                            res = json.loads(resp.read().decode("utf-8"))
                            status_txt = f"Status: {res.get('status', 'online')} | Version: {res.get('version', 'v7.4')}"
                            self.root.after(0, lambda s=status_txt: self.lbl_telemetry.config(text=s))
                    except Exception:
                        pass
                elif mode == "SERIAL" and self.ser and self.ser.is_open:
                    try:
                        line = self.ser.readline()
                        if line:
                            msg = json.loads(line.decode("utf-8", errors="replace").strip())
                            if msg.get("type") == "telemetry":
                                d = msg.get("data", {})
                                status_txt = f"Status: {d.get('status', 'online')} | Version: {d.get('version', 'v7.4')}"
                                self.root.after(0, lambda s=status_txt: self.lbl_telemetry.config(text=s))
                            elif msg.get("type") == "response" and "speech" in msg.get("result", {}):
                                sp = msg["result"]["speech"]
                                self.root.after(0, lambda s=sp: self.log(f"Robot: {s}"))
                    except Exception:
                        pass
            time.sleep(2.0)


def main():
    root = tk.Tk()
    app = PathfinderApp(root)
    root.protocol("WM_DELETE_WINDOW", lambda: (setattr(app, "running", False), root.destroy()))
    root.mainloop()

if __name__ == "__main__":
    main()
