# THE-PATHFINDER-EYE : SD Card Recovery Guide (v7.3)

> **Last Updated:** 2026-05-14
> **Status:** Recovery Toolkit Ready

---

## ⚠️ What I (the AI) Can and Cannot Do

| I CAN do | I CANNOT do |
|---|---|
| Write setup scripts & documentation | Physically flash the SD card |
| Fix code bugs & model path mismatches | Boot the Pi or test I2C/camera/motors |
| Create systemd services for auto-start | Download 5.3GB models over slow WiFi for you |
| Analyze the codebase for issues | Confirm hardware actually works |

**You** must do the physical steps (flashing, booting, plugging in cables). I have prepared everything else.

---

## 🛠️ Step 1: Flash the SD Card (YOU do this)

1. **Download Raspberry Pi OS (64-bit, Lite)** from https://downloads.raspberrypi.org/raspios_lite_arm64_latest
2. **Flash it** using Raspberry Pi Imager or `dd`:
   ```bash
   # Example with dd (BE CAREFUL with of=)
   sudo dd if=2024-11-19-raspios-bookworm-arm64-lite.img of=/dev/mmcblk0 bs=4M status=progress conv=fsync
   ```
3. **Enable SSH and I2C before first boot:**
   - Create an empty file named `ssh` in the boot partition
   - Add to `config.txt` in the boot partition:
     ```
     dtparam=i2c_arm=on
     dtparam=spi=on
     start_x=1
     gpu_mem=128
     ```
4. **Create `wpa_supplicant.conf`** in the boot partition for WiFi (optional but recommended):
   ```
   ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev
   update_config=1
   country=US

   network={
       ssid="YOUR_WIFI_SSID"
       psk="YOUR_WIFI_PASSWORD"
       key_mgmt=WPA-PSK
   }
   ```
5. **Boot the Pi**, find its IP (`arp-scan` or router admin page), and SSH in:
   ```bash
   ssh pi@<pi-ip>
   # Default password: raspberry (change it immediately)
   ```

---

## 📦 Step 2: Copy This Repo to the Pi (YOU do this)

Option A — **Clone from GitHub** (if you pushed it):
```bash
git clone https://github.com/Alartist40/the-pathfinder-eye.git /home/pi/the-pathfinder-eye_ai
```

Option B — **Copy from this machine** (if the Pi is on the same network):
```bash
# On your Linux machine (NOT the Pi):
rsync -avz --progress /home/xander/Documents/THE-PATHFINDER-EYE/ pi@<pi-ip>:/home/pi/the-pathfinder-eye_ai/
```

Option C — **USB/SD card direct copy** (mount the Pi's SD card on this machine and copy files to `/home/pi/`).

---

## 🤖 Step 3: Run the Automated Setup (mostly automated)

Once the repo is on the Pi:

```bash
cd /home/pi/the-pathfinder-eye_ai
chmod +x setup-pi.sh
./setup-pi.sh
```

This script will:
- ✅ Install all system packages (I2C, audio, OpenCV, build tools)
- ✅ Install Go 1.23 and Rust (if missing)
- ✅ Enable I2C and camera interfaces
- ✅ Build whisper.cpp and download the 500MB `ggml-small.bin` model
- ✅ Clone/build llama.cpp (shared libraries)
- ✅ Build the LeafcutterLLM Rust server (FFI to llama.cpp)
- ✅ Build the Go Brain (`go build -o ../brain .`)
- ✅ Build the Rust Vision Engine (YOLOv8/v5 dual detection & face tracking)
- ✅ Install systemd services
- ⚠️ **PROMPT you** about downloading the ~1.5GB Qwen3.5-2B-Q4_K_M model (you can skip and copy it via USB)

**Estimated time:** 15-30 minutes (excluding the LLM model download).

---

## 💾 Step 4: The Model (YOU decide how)

The default production LLM model is **~1.5GB** (`Qwen3.5-2B-Q4_K_M.gguf`), perfectly sized for RPi 5 thermal and memory headroom. The setup script will ask if you want to download it. Options:

### Option A: Download on the Pi (automatic)
Say `y` when the script prompts.

### Option B: Download on a fast machine, copy via USB
1. On a fast machine with good internet:
   ```bash
   wget -O Qwen3.5-2B-Q4_K_M.gguf "https://huggingface.co/bartowski/Qwen_Qwen3.5-2B-GGUF/resolve/main/Qwen_Qwen3.5-2B-Q4_K_M.gguf?download=true"
   ```
2. Copy the file to the Pi's SD card or via SCP:
   ```bash
   scp Qwen3.5-2B-Q4_K_M.gguf pi@<pi-ip>:/home/pi/the-pathfinder-eye_ai/models/
   ```

### Option C: Use a different model
Edit `go_brain/main.go` and change the `modelPath` to any `.gguf` file llama.cpp supports (e.g. `Qwen3.5-4B-Q4_K_M.gguf`). Then rebuild:
```bash
cd /home/pi/the-pathfinder-eye_ai/go_brain
go build -o ../brain .
```

---

## 🔧 Step 5: Start Services

```bash
# Start the AI engine
sudo systemctl start leafcutter

# Wait 10-20 seconds for the model to load into RAM
sleep 15
curl http://localhost:8081/health

# Start the robot brain
sudo systemctl start pathfinder-eye

# Check if it's alive
curl http://localhost:8080/health

# Enable auto-start on boot
sudo systemctl enable leafcutter
sudo systemctl enable pathfinder-eye
```

---

## 🎮 Step 6: Hardware Test

```bash
cd /home/pi/the-pathfinder-eye_ai
bash quick_test_hardware.sh
```

You should hear/see:
1. Motors spin forward briefly
2. Motors spin left briefly
3. Camera gimbal pans left → right → center
4. Camera gimbal tilts up → center → down → center

If any of this fails:
- **Motors don't spin:** Check I2C wiring, run `i2cdetect -y 1` — you should see `2b`
- **Gimbal doesn't move:** Check servo power (separate 5V rail usually)
- **Camera not found:** Run `libcamera-hello` to test; may need `dtoverlay=imx708` in `/boot/firmware/config.txt`

---

## 🌐 Step 7: Dashboard

Open a browser on any device on the same network:
```
http://<pi-ip>:8080
```

You should see the live camera feed, system stats, and drive controls.

---

## 🐛 Known Issues & Fixes

### Issue: `go build` fails with whisper.cpp binding errors
**Fix:** The CGO flags in `setup-pi.sh` set `-I` and `-L` paths. If it still fails:
```bash
export CGO_CFLAGS="-I/usr/local/lib/whisper"
export CGO_LDFLAGS="-L/usr/local/lib/whisper -lwhisper -lm -lopenblas"
cd /home/pi/the-pathfinder-eye_ai/go_brain
go build -o ../brain .
```

### Issue: Rust Vision fails to build (OpenCV Rust bindings)
**Fix:** The Rust Vision Engine uses OpenCV DNN for YOLOv8/v5 detection and face tracking. If `opencv` crate fails to build on a headless Lite install, install `libopencv-dev` and ensure `PKG_CONFIG_PATH` is set. If needed, the Go Brain will continue running hardware and voice control even if the vision sub-process is absent.

```bash
sudo apt-get install -y libopencv-dev clang
cd /home/pi/the-pathfinder-eye_ai/rust_vision
cargo build --release
```

### Issue: LeafcutterLLM uses too much RAM
**Fix:** The `main.go` safety monitor will auto-kill Leafcutter if RAM > 92%. You can also edit `leafcutter.service` to add:
```
MemoryMax=12G
```

### Issue: No audio / TTS silent
**Fix:**
```bash
# Check audio device
aplay -l
# Set volume
amixer set Speaker 100%
# Test TTS manually
espeak-ng "Hello"
```

---

## 📋 Quick Reference Cheat Sheet

```bash
# View logs
sudo journalctl -u pathfinder-eye -f
sudo journalctl -u leafcutter -f
tail -f /home/pi/the-pathfinder-eye_ai/logs/go_brain.log

# Restart everything
sudo systemctl restart leafcutter
sleep 15
sudo systemctl restart pathfinder-eye

# Stop everything
sudo systemctl stop pathfinder-eye leafcutter

# Hardware scan
i2cdetect -y 1

# RAM check
free -h
vcgencmd measure_temp

# Camera test
libcamera-hello -t 5000
```

---

## 🔄 If You Need to Re-Flash Again

1. Keep a backup of `/home/pi/the-pathfinder-eye_ai/models/*.gguf` (the big models) on an external drive
2. Keep a backup of `db/` (your robot's memory — DENDRITE graph, vision history, etc.)
3. Re-flash, copy the repo, run `./setup-pi.sh`, copy models back, restore `db/`, start services

---

**Questions?** Check `docs/` for protocol specs, or run the hardware test script.
