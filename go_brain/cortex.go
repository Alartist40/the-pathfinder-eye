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
			time.Sleep(200 * time.Millisecond)
			continue
		}

		// 1. PASSIVE LISTENING (Low Impact)
		// AUDIO_POLICY.md rule 3: 3-second uninterruptible window.
		samples, err := captureAudio(PerWakeWordListenSec)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// 2. VOLUME GATE (Optimized to 0.003 to allow quiet speech while filtering line static)
		if isQuiet(samples, 0.003) {
			continue
		}

		// 3. WAKE WORD DETECTION
		text, err := transcribeAudio(samples)
		if err != nil || text == "" {
			continue
		}
		text = strings.TrimSpace(text)
		lowerText := strings.ToLower(text)
		infoLog.Printf("VOICE_DETECTED: %q", text)

		// ANTI-FEEDBACK: If the robot just finished talking, ignore the detection
		// (PostSpeechCooldownSec — bound to AUDIO_POLICY.md rule 1.)
		if time.Since(lastSpokeTime) < time.Duration(PostSpeechCooldownSec)*time.Second {
			continue
		}

		// Check for wake words
		if isWakeWord(text) || strings.Contains(lowerText, "pathfinder") ||
			strings.Contains(lowerText, "computer") || strings.Contains(lowerText, "robot") {
			atomic.StoreInt32(&commandBusy, 1)

			// Visual indicator: Solid Blue for active listening
			_ = setLEDAll(1, LEDColorBlue)
			if ttsEngine != nil {
				_ = ttsEngine.SpeakCritical("yes")
			}

			// Small pause to let "Yes" finish playing and clearing from the air
			time.Sleep(1000 * time.Millisecond)

			go c.handleActiveConversation()
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
				lastSpokeTime = time.Now()
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

	// Try direct command dispatch first (offline, instant response)
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
		lastSpokeTime = time.Now()
		return
	}

	// Fall back to AI brain (local leafcutter LLM)
	_ = setLEDAll(1, LEDColorYellow)
	worldState := GetWorldStatePrompt()

	speech, err := aiBrain.Process(finalText, worldState)
	if err == nil && speech != "" {
		indicateSuccess()
		_ = speak(speech)
		lastSpokeTime = time.Now()
	} else if err != nil {
		infoLog.Printf("CORTEX_AGENT_ERROR: %v", err)
		indicateWarning()
		if ttsEngine != nil {
			_ = ttsEngine.Speak("My neural link is struggling.")
		}
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
