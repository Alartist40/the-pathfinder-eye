package main

import (
	"time"
)

func enableUltrasonic() error {
	return i2cWrite(0x07, []byte{1})
}

func disableUltrasonic() error {
	return i2cWrite(0x07, []byte{0})
}

func readDistanceMM() (int, error) {
	i2cMutex.Lock()
	defer i2cMutex.Unlock()

	// Read high byte (0x1B)
	bufHigh := make([]byte, 1)
	if _, err := i2cFile.Write([]byte{0x1B}); err != nil { return 0, err }
	if _, err := i2cFile.Read(bufHigh); err != nil { return 0, err }

	// Read low byte (0x1A)
	bufLow := make([]byte, 1)
	if _, err := i2cFile.Write([]byte{0x1A}); err != nil { return 0, err }
	if _, err := i2cFile.Read(bufLow); err != nil { return 0, err }

	dist := (int(bufHigh[0]) << 8) | int(bufLow[0])
	return dist, nil
}

func getDistanceCM() (float64, error) {
    enableUltrasonic()
    time.Sleep(60 * time.Millisecond)
    mm, err := readDistanceMM()
    disableUltrasonic()
    if err != nil { return 0, err }
    return float64(mm) / 10.0, nil
}

var followModeActive bool

func startFollowMode() {
    followModeActive = true
    for followModeActive {
        dist, err := getDistanceCM()
        if err == nil && dist > 0 {
            if dist < 15 {
                // Too close, step back
                for i := byte(0); i < 4; i++ { moveMotor(i, 1, 120) }
            } else if dist > 25 && dist < 100 {
                // Follow
                for i := byte(0); i < 4; i++ { moveMotor(i, 0, 120) }
            } else {
                stopAllMotors()
            }
        }
        time.Sleep(100 * time.Millisecond)
    }
    stopAllMotors()
}

func checkSafetyDistance() {
    for {
        if !followModeActive {
            dist, err := getDistanceCM()
            if err == nil && dist > 0 && dist < 15 {
                // Emergency step back
                for i := byte(0); i < 4; i++ { moveMotor(i, 1, 150) }
                time.Sleep(500 * time.Millisecond)
                stopAllMotors()
            }
        }
        time.Sleep(200 * time.Millisecond)
    }
}
