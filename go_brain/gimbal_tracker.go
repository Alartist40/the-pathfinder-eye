package main

import (
	"sync"
)

type GimbalTracker struct {
	currentPan  int
	currentTilt int
	targetPan   int
	targetTilt  int
	mu          sync.Mutex
	smooth      bool
}

func NewGimbalTracker() *GimbalTracker {
	return &GimbalTracker{
		currentPan:  90,
		currentTilt: 110,
		targetPan:   90,
		targetTilt:  110,
		smooth:      true,
	}
}

func (gt *GimbalTracker) TrackObject(detX, detY, detW, detH, frameW, frameH int) {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	centerX := detX + detW/2
	centerY := detY + detH/2

	// Map frame coordinates to gimbal angles
	gt.targetPan = 90 + (centerX-frameW/2)*90/frameW
	gt.targetTilt = 110 + (centerY-frameH/2)*70/frameH

	if gt.targetPan < 30 { gt.targetPan = 30 } else if gt.targetPan > 150 { gt.targetPan = 150 }
	if gt.targetTilt < 30 { gt.targetTilt = 30 } else if gt.targetTilt > 180 { gt.targetTilt = 180 }

	gt.updateGimbal()
}

func (gt *GimbalTracker) updateGimbal() {
	if gt.smooth {
		if gt.currentPan < gt.targetPan { gt.currentPan += 2 } else if gt.currentPan > gt.targetPan { gt.currentPan -= 2 }
		if gt.currentTilt < gt.targetTilt { gt.currentTilt += 2 } else if gt.currentTilt > gt.targetTilt { gt.currentTilt -= 2 }
	} else {
		gt.currentPan = gt.targetPan
		gt.currentTilt = gt.targetTilt
	}

	_ = setServo(1, byte(gt.currentPan))
	_ = setServo(2, byte(gt.currentTilt))
}

func (gt *GimbalTracker) GetCurrentPosition() (pan, tilt int) {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	return gt.currentPan, gt.currentTilt
}
