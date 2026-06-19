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
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// 1. PASSIVE LISTENING (Low Impact)
		// AUDIO_POLICY.md rule 3: 3-second uninterruptible window.
		samples, err := captureAudio(PerWakeWordListenSec)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		// 2. VOLUME GATE (Optimized to 0.02 to avoid fan noise)
		if isQuiet(samples, 0.02) {
			continue
		}

		// 3. WAKE WORD DETECTION
		text, err := transcribeAudio(samples)
		if err != nil || text == "" {
			continue
		}
		text = strings.TrimSpace(text)
		lowerText := strings.ToLower(text)

		// ANTI-FEEDBACK: If the robot just finished talking, ignore the detection
		// (PostSpeechCooldownSec — bound to AUDIO_POLICY.md rule 1.)
		if time.Since(lastSpokeTime) < time.Duration(PostSpeechCooldownSec)*time.Second {
			continue
		}

		// Check for wake words
		if isWakeWord(text) || strings.Contains(lowerText, "pathfinder") {
			atomic.StoreInt32(&commandBusy, 1)

			go indicateWakeWord()
			if ttsEngine != nil {
				_ = ttsEngine.SpeakCritical("yes")
			}

			// Small pause to let "Yes" finish playing and clearing from the air
			time.Sleep(1500 * time.Millisecond)

			go c.handleActiveConversation()
		}
	}
}

func (c *AICortex) handleActiveConversation() {
	defer atomic.StoreInt32(&commandBusy, 0)

	infoLog.Println("CORTEX: Actively listening for natural command...")

	// AUDIO_POLICY.md rule 4: 5-second uninterruptible window.
	// One capture of the full window instead of multiple chunks — the prior
	// chunked version silently doubled the policy time.
	var fullCommand []string
	samples, _ := captureAudio(PerCommandListenSec)
	text, _ := transcribeAudio(samples)
	if text != "" {
		fullCommand = append(fullCommand, text)
	}

	finalText := strings.Join(fullCommand, " ")
	if finalText == "" {
		if ttsEngine != nil {
			_ = ttsEngine.Speak("I didn't hear anything.")
		}
		return
	}

	infoLog.Printf("CORTEX: Executing: '%s'", finalText)
	// Speech transcripts may carry secrets (e.g. someone reading an
	// API key aloud). Redact before the LLM call records them.
	safeLogf("", "CORTEX: speech payload redacted: %s",
		redactOnce(finalText))
	go indicateProcessing()
	worldState := GetWorldStatePrompt()

	speech, err := aiBrain.Process(finalText, worldState)
	if err == nil && speech != "" {
		_ = speak(speech)
		lastSpokeTime = time.Now() // Update cooldown
	} else if err != nil {
		infoLog.Printf("CORTEX_AGENT_ERROR: %v", err)
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
