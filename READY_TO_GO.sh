#!/bin/bash
# THE-PATHFINDER-EYE Startup Script v6.7-PRODUCTION
# Performs comprehensive hardware and software health checks before launch.

set -e  # Exit on any failure to ensure system integrity

LOG_FILE="/home/pi/the-pathfinder-eye_ai/logs/startup.log"
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

# Ensure logging environment is ready
mkdir -p /home/pi/the-pathfinder-eye_ai/logs

echo "[$TIMESTAMP] ========================================" | tee -a "$LOG_FILE"
echo "[$TIMESTAMP] THE-PATHFINDER-EYE BOOT SEQUENCE START" | tee -a "$LOG_FILE"
echo "[$TIMESTAMP] ========================================" | tee -a "$LOG_FILE"

# 1. Verify I2C Physical Layer
echo "[$TIMESTAMP] CHECK 1/8: Verifying I2C bus connectivity..." | tee -a "$LOG_FILE"
if ! ls /dev/i2c-1 > /dev/null 2>&1; then
    echo "[$TIMESTAMP] CRITICAL ERROR: /dev/i2c-1 not found! Enable I2C in raspi-config." | tee -a "$LOG_FILE"
    exit 1
fi
echo "[$TIMESTAMP] ✓ I2C physical bus verified." | tee -a "$LOG_FILE"

# 2. Verify Yahboom Controller Address
echo "[$TIMESTAMP] CHECK 2/8: Scanning for I2C Slave 0x2B..." | tee -a "$LOG_FILE"
if ! i2cdetect -y 1 | grep -q "2b"; then
    echo "[$TIMESTAMP] CRITICAL ERROR: Yahboom motor controller not detected at 0x2B. Check power/cables." | tee -a "$LOG_FILE"
    exit 1
fi
echo "[$TIMESTAMP] ✓ Yahboom controller (0x2B) online." | tee -a "$LOG_FILE"

# 3. Verify Camera Hardware
echo "[$TIMESTAMP] CHECK 3/8: Verifying Camera Sensor..." | tee -a "$LOG_FILE"
if ! ls /dev/video0 > /dev/null 2>&1; then
    echo "[$TIMESTAMP] WARNING: /dev/video0 not found. Vision Engine will fail to start." | tee -a "$LOG_FILE"
else
    echo "[$TIMESTAMP] ✓ Camera sensor verified." | tee -a "$LOG_FILE"
fi

# 4. Verify AI Engine (LeafcutterLLM)
echo "[$TIMESTAMP] CHECK 4/8: Handshaking with LeafcutterLLM (Port 8081)..."
if ! curl -s http://localhost:8081/health | grep -q "ok"; then
    echo "[$TIMESTAMP] CRITICAL ERROR: LeafcutterLLM server not running. Run bash INTEGRATE_LEAFCUTTER.sh" | tee -a "$LOG_FILE"
    exit 1
fi
echo "[$TIMESTAMP] ✓ LeafcutterLLM engine detected." | tee -a "$LOG_FILE"

# 5. Build/Verify Go Brain Orchestrator
echo "[$TIMESTAMP] CHECK 5/8: Finalizing Go Brain compilation..." | tee -a "$LOG_FILE"
export PATH=$PATH:/usr/local/go/bin
cd /home/pi/the-pathfinder-eye_ai/go_brain
if go build -o brain main.go; then
    echo "[$TIMESTAMP] ✓ Go Brain binary compiled successfully." | tee -a "$LOG_FILE"
else
    echo "[$TIMESTAMP] CRITICAL ERROR: Go compilation failed." | tee -a "$LOG_FILE"
    exit 1
fi

# 6. Start the Master Brain
echo "[$TIMESTAMP] CHECK 6/8: Spawning Orchestrator Process..." | tee -a "$LOG_FILE"
sudo killall brain 2>/dev/null || true
./brain >> "$LOG_FILE" 2>&1 &
GO_PID=$!
echo "[$TIMESTAMP] ✓ Master Brain started (PID: $GO_PID)." | tee -a "$LOG_FILE"

# 7. Wait for System Readiness (Health Check)
echo "[$TIMESTAMP] CHECK 7/8: Performing System Health Handshake..." | tee -a "$LOG_FILE"
READY=0
for i in {1..20}; do
    if curl -s http://localhost:8080/health | grep -q "online"; then
        READY=1
        echo "[$TIMESTAMP] ✓ System Handshake SUCCESS." | tee -a "$LOG_FILE"
        break
    fi
    sleep 1
done

if [ $READY -eq 0 ]; then
    echo "[$TIMESTAMP] CRITICAL ERROR: Master Brain failed to report ONLINE status." | tee -a "$LOG_FILE"
    kill $GO_PID
    exit 1
fi

# 8. Start Vision Engine
echo "[$TIMESTAMP] CHECK 8/8: Initializing Vision Pipeline..." | tee -a "$LOG_FILE"
# Check if vision binary exists
if [ -f "/home/pi/the-pathfinder-eye_ai/rust_vision/target/release/rust_vision" ]; then
    /home/pi/the-pathfinder-eye_ai/rust_vision/target/release/rust_vision >> "/home/pi/the-pathfinder-eye_ai/logs/vision.log" 2>&1 &
    echo "[$TIMESTAMP] ✓ Rust Vision Engine started." | tee -a "$LOG_FILE"
else
    echo "[$TIMESTAMP] WARNING: Rust Vision binary not found. Run cargo build first." | tee -a "$LOG_FILE"
fi

# Final Summary
IP_ADDR=$(hostname -I | awk '{print $1}')
echo "[$TIMESTAMP] ========================================" | tee -a "$LOG_FILE"
echo "[$TIMESTAMP] THE-PATHFINDER-EYE STATUS: OPERATIONAL" | tee -a "$LOG_FILE"
echo "[$TIMESTAMP] DASHBOARD: http://$IP_ADDR:8080" | tee -a "$LOG_FILE"
echo "[$TIMESTAMP] ========================================" | tee -a "$LOG_FILE"
