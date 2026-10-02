#!/bin/bash
# THE-PATHFINDER-EYE: Comprehensive Hardware Verification v6.4 (Native I2C)
# NO PYTHON REQUIRED. Direct OS-level verification.

echo "🏎️ Testing Drive System (4WD via i2cset)..."
# Register 0x01: [motor_id, direction, speed]
# Forward speed 120
for i in {0..3}; do i2cset -y 1 0x2B 0x01 $i 0 120 i; done
sleep 0.5
# Stop
for i in {0..3}; do i2cset -y 1 0x2B 0x01 $i 0 0 i; done

sleep 0.3

# Rotate Left (L backward, R forward)
i2cset -y 1 0x2B 0x01 0 1 120 i
i2cset -y 1 0x2B 0x01 1 1 120 i
i2cset -y 1 0x2B 0x01 2 0 120 i
i2cset -y 1 0x2B 0x01 3 0 120 i
sleep 0.5
# Stop
for i in {0..3}; do i2cset -y 1 0x2B 0x01 $i 0 0 i; done

echo "📐 Testing Camera Gimbal (Native i2cset)..."
# Register 0x02: [servo_id, angle]
# Pan Sweep (ID 1)
i2cset -y 1 0x2B 0x02 1 45 i; sleep 0.8
i2cset -y 1 0x2B 0x02 1 135 i; sleep 0.8
i2cset -y 1 0x2B 0x02 1 90 i

# Tilt Sweep (ID 2 - Calibrated center 60 for this test)
i2cset -y 1 0x2B 0x02 2 100 i; sleep 0.8
i2cset -y 1 0x2B 0x02 2 60 i; sleep 0.8
i2cset -y 1 0x2B 0x02 2 30 i; sleep 0.8
i2cset -y 1 0x2B 0x02 2 60 i

echo "✅ Native Hardware Test Complete."
