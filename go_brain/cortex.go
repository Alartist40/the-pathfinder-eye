/**
 * THE-PATHFINDER-EYE : Integrated AI Cortex (v6.3 - Echo Suppression)
 */

package main

import (
	"strings"
	"sync/atomic"
	"time"
)

type AICortex struct {
	Active bool
}

func newCortex() *AICortex {
	return &AICortex{Active: true}
}

func (c *AICortex) StartUnifiedAwareness() {
	if c == nil {
		infoLog.Println("CORTEX: WARNING - cortex is nil, awareness disabled")
		return
	}
	infoLog.Println("CORTEX: Gated Awareness with Echo Suppression active.")

	for {
		if atomic.LoadInt32(&commandBusy) == 1 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Anti-feedback: If robot is speaking or finished within cooldown, wait
		if ttsEngine != nil && ttsEngine.isSpeaking() {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if time.Since(lastSpokeTime) < time.Duration(PostSpeechCooldownSec)*time.Second {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		// 1. PASSIVE LISTENING (Low Impact)
		// AUDIO_POLICY.md rule 3: 3-second uninterruptible window.
		samples, err := captureAudio(PerWakeWordListenSec)
		if err != nil {
			time.Sleep(300 * time.Millisecond)
			continue
		}

		// 2. VOLUME GATE (0.002 to allow soft speech while filtering silence)
		if isQuiet(samples, 0.002) {
			continue
		}

		// 3. WAKE WORD DETECTION
		text, err := transcribeAudio(samples)
		if err != nil || text == "" {
			continue
		}
		text = strings.TrimSpace(text)
		infoLog.Printf("VOICE_DETECTED: %q", text)

		// Double-check anti-feedback
		if time.Since(lastSpokeTime) < time.Duration(PostSpeechCooldownSec)*time.Second {
			continue
		}

		// Check for wake words
		if isWakeWord(text) {
			atomic.StoreInt32(&commandBusy, 1)

			// Visual indicator: Solid Blue for active listening
			_ = setLEDAll(1, LEDColorBlue)
			if ttsEngine != nil {
				_ = ttsEngine.SpeakCritical("yes")
			}

			// Wait for "Yes" to finish playing before recording command
			waitForTTS()

			c.handleActiveConversation()
		} else {
			// Direct command execution in passive window (e.g. "move forward", "stop", "lights on")
			level := LevelGuest
			name := "Guest"
			if sp, err := visionDB.GetCurrentSpeaker(); err == nil {
				if figure, recognized := authority.VerifyFigure(sp.FaceID); recognized {
					level = figure.Level
					name = figure.Name
				}
			}
			if processDirectCommand(text, level, name) {
				atomic.StoreInt32(&commandBusy, 1)
				_ = setLEDAll(1, LEDColorGreen)
				time.Sleep(600 * time.Millisecond)
				_ = setLEDAll(0, 0)
				atomic.StoreInt32(&commandBusy, 0)
			}
		}
	}
}

func (c *AICortex) handleActiveConversation() {
	defer func() {
		_ = setLEDAll(0, 0)
		atomic.StoreInt32(&commandBusy, 0)
	}()

	infoLog.Println("CORTEX: Actively listening for natural command...")

	// Visual indicator: Solid Blue during recording window so user knows mic is capturing
	_ = setLEDAll(1, LEDColorBlue)

	// AUDIO_POLICY.md rule 4: 5-second uninterruptible window.
	var fullCommand []string
	samples, _ := captureAudio(PerCommandListenSec)

	// Visual indicator: Yellow while transcribing/processing
	_ = setLEDAll(1, LEDColorYellow)

	text, _ := transcribeAudio(samples)
	if text != "" {
		fullCommand = append(fullCommand, text)
	}

	finalText := strings.Join(fullCommand, " ")
	if finalText == "" {
		indicateWarning()
		if ttsEngine != nil {
			_ = ttsEngine.Speak("I didn't hear anything.")
		}
		return
	}

	infoLog.Printf("CORTEX: Executing: '%s'", finalText)
	safeLogf("", "CORTEX: speech payload redacted: %s",
		redactOnce(finalText))

	// Try direct command dispatch (Needle 2 + Rule Parser)
	level := LevelGuest
	name := "Guest"
	if sp, err := visionDB.GetCurrentSpeaker(); err == nil {
		if figure, recognized := authority.VerifyFigure(sp.FaceID); recognized {
			level = figure.Level
			name = figure.Name
		}
	}
	if processDirectCommand(finalText, level, name) {
		indicateSuccess()
		return
	}
}

func isQuiet(samples []float32, threshold float32) bool {
	var max float32
	for _, s := range samples {
		if s > max {
			max = s
		}
		if s < -max {
			max = -s
		}
	}
	return max < threshold
}
