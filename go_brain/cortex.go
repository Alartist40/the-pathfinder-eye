/**
 * THE-PATHFINDER-EYE : Integrated AI Cortex (v6.4 - Vocal Feedback)
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

func (c *AICortex) StartUnifiedAwareness() {
	infoLog.Println("CORTEX: Gated Awareness active.")
	
	for {
		if atomic.LoadInt32(&commandBusy) == 1 {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// 1. PASSIVE LISTENING (Low Impact)
		samples, err := captureAudio(2) 
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		// 2. VOLUME GATE (Optimized for fan noise)
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
		
		// ANTI-FEEDBACK: Cooldown after speaking
		if time.Since(lastSpokeTime) < 2*time.Second {
			continue
		}

		if isWakeWord(text) || strings.Contains(lowerText, "pathfinder") {
			atomic.StoreInt32(&commandBusy, 1)
			
			go indicateWakeWord()
			if ttsEngine != nil { _ = ttsEngine.SpeakCritical("yes") }
			
			// Pause to clear room audio
			time.Sleep(1200 * time.Millisecond)
			go c.handleActiveConversation()
		}
	}
}

func (c *AICortex) handleActiveConversation() {
	defer atomic.StoreInt32(&commandBusy, 0)
	
	var fullCommand []string
	silenceCount := 0
	
	for i := 0; i < 6; i++ {
		go indicateCommandAck()
		samples, _ := captureAudio(2)
		
		if isQuiet(samples, 0.015) {
			silenceCount++
			if silenceCount >= 2 && len(fullCommand) > 0 {
				break
			}
			continue
		}
		
		silenceCount = 0
		text, _ := transcribeAudio(samples)
		if text != "" {
			fullCommand = append(fullCommand, text)
		}
	}
	
	finalText := strings.Join(fullCommand, " ")
	if finalText == "" {
		if ttsEngine != nil { _ = ttsEngine.Speak("I didn't hear anything.") }
		return
	}
	
	infoLog.Printf("CORTEX: Received: '%s'", finalText)
	
	// IMMEDIATE FEEDBACK: "Understood"
	if ttsEngine != nil { _ = ttsEngine.SpeakCritical("Understood") }

	go indicateProcessing()
	worldState := GetWorldStatePrompt()
	
	_, err := aiBrain.Process(finalText, worldState)
	if err != nil {
		infoLog.Printf("CORTEX_AGENT_ERROR: %v", err)
		if ttsEngine != nil { _ = ttsEngine.Speak("My neural link is struggling.") }
	}
	lastSpokeTime = time.Now()
}

func isQuiet(samples []float32, threshold float32) bool {
	var max float32
	for _, s := range samples {
		if s > max { max = s }
		if s < -max { max = -s }
	}
	return max < threshold
}
