package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var (
	trackingMode        = false
	aiActive            = false
	commandBusy         int32
	lastSpokeTime       time.Time
	remoteControlActive bool
	deepThoughtActive   bool
)

// MinFreeMBFor70B is the minimum MB of free memory required to attempt
// loading the 70B model. Llama-3.1-70B-Instruct-Q4_K_S requires about
// 22000 MB during prompt-eval; we pad to 20480 MB (20 GB) floor. On an
// 8 GB Pi 5 this guard will always fire, which is correct — deep thought
// must never try to load 70B on a board that physically cannot hold it.
// Bumping this constant is the only way to enable 70B on bigger hardware.
const MinFreeMBFor70B = 20480

// wakeWords are STT variants of the wake word "instruction".
// These are common misrecognitions from the speech-to-text engine.
// Each is matched as a whole word only to avoid false positives.
var wakeWords = map[string]bool{
	"instruction": true, "instruct": true,
	"destruction": true, "restruction": true,
}

func isWakeWord(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	words := strings.Fields(t)
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:'\"-()[]{}")
		if wakeWords[word] {
			return true
		}
	}
	return false
}

func handleCommandSequence() {
	defer atomic.StoreInt32(&commandBusy, 0)
	// AUDIO_POLICY.md rule 4: 5-second listening window. We sleep briefly
	// before opening the listen so the wake-confirmation ("yes") finishes.
	time.Sleep(1200 * time.Millisecond)

	// authorityLevel / speakerName are recomputed on every retry attempt
	// inside the loop so a face re-detection mid-sequence doesn't run
	// commands against a stale identity.
	level := LevelGuest
	name := "Guest"

	// One 5-second capture window. Single retry only, which keeps the
	// total worst-case window within policy.
	for attempt := 0; attempt < 2; attempt++ {
		// Re-snapshot the speaker for this attempt. visionDB.GetCurrentSpeaker
		// is cheap (returns from the latest detection frame) but callers
		// that touched face verification since the previous attempt will
		// see a fresh authority.
		if sp, err := visionDB.GetCurrentSpeaker(); err == nil {
			if figure, recognized := authority.VerifyFigure(sp.FaceID); recognized {
				level = figure.Level
				name = figure.Name
			} else {
				level = LevelGuest
				name = "Guest"
			}
		}

		go indicateCommandAck()
		samples, err := captureAudio(PerCommandListenSec)
		if err != nil {
			break
		}
		cmdText, _ := transcribeAudio(samples)
		cmd := strings.ToLower(cmdText)
		if cmd == "" {
			continue
		}

		if isWakeWord(cmd) {
			if ttsEngine != nil {
				_ = ttsEngine.SpeakCritical("resetting")
			}
			attempt = -1
			continue
		}

		if !authority.CanExecuteCommand(level, cmd) {
			if ttsEngine != nil {
				_ = ttsEngine.SpeakCritical("Permission denied")
			}
			return
		}

		if processDirectCommand(cmd, level, name) {
			return
		}
	}

	go indicateWarning()
	if ttsEngine != nil {
		_ = ttsEngine.SpeakCritical("I did not understand.")
	}
}

// processDirectCommand handles robot commands using semantic routing.
// Falls back to keyword extraction when routing confidence is low.
// Returns true if command was handled, false if unrecognized.
func processDirectCommand(cmd string, level AuthorityLevel, name string) bool {
	// Try semantic routing — routes to basic_chat, thinking, or tool_call.
	route, _ := ClassifyRoute(cmd)

	// Only use router for tool_call; basic_chat/thinking go to AI brain.
	if route != RouteToolCall {
		// Fallback: check for exit/stop/sleep which are always valid.
		lower := strings.ToLower(cmd)
		if strings.Contains(lower, "exit") || strings.Contains(lower, "stop") ||
			strings.Contains(lower, "sleep") || strings.Contains(lower, "shut down") {
			return handleExitCommand()
		}
		return false // let AI brain handle it
	}

	// Route is tool_call — extract structured action + target.
	cmdParsed := ExtractCommand(cmd)
	infoLog.Printf("ROUTER: action=%q target=%q raw=%q", cmdParsed.Action, cmdParsed.Target, cmd)

	switch cmdParsed.Action {
	case "test":
		if ttsEngine != nil {
			_ = ttsEngine.SpeakCritical("understood")
		}
		go handleFullHardwareTest_Direct()
		return true

	case "move":
		return handleMoveAction(cmdParsed)

	case "look":
		return handleLookAction(cmdParsed)

	case "play":
		return handlePlayAction(cmdParsed)

	case "read":
		return handleReadAction(cmdParsed)

	case "activate":
		return handleActivateAction(cmdParsed)

	case "deactivate":
		return handleDeactivateAction(cmdParsed)

	case "attention":
		if ttsEngine != nil {
			_ = ttsEngine.SpeakCritical("understood. activating AI intelligence.")
		}
		exec.Command("sudo", "systemctl", "start", "leafcutter.service").Run()
		go startAIConversationLoop(name)
		return true

	case "remote":
		if ttsEngine != nil {
			_ = ttsEngine.SpeakCritical("understood. remote control mode enabled.")
		}
		remoteControlActive = true
		_ = os.WriteFile("/tmp/stream_active", []byte("1"), 0644)
		return true

	case "deep":
		if ttsEngine != nil {
			_ = ttsEngine.SpeakCritical("entering deep thought. suspending vision systems.")
		}
		deepThoughtActive = true
		go enterDeepThought()
		return true

	case "translate":
		if ttsEngine != nil {
			_ = ttsEngine.SpeakCritical("understood. japanese mode active. say stop to exit.")
		}
		go startJapaneseTranslationLoop()
		return true

	default:
		// Fallback: check for exit/stop/sleep.
		lower := strings.ToLower(cmd)
		if strings.Contains(lower, "exit") || strings.Contains(lower, "stop") ||
			strings.Contains(lower, "sleep") || strings.Contains(lower, "shut down") {
			return handleExitCommand()
		}
		return false
	}
}

// handleExitCommand cleans up all active modes.
func handleExitCommand() bool {
	if ttsEngine != nil {
		_ = ttsEngine.SpeakCritical("system idle.")
	}
	stopAllMotors()
	birdwatchActive = false
	followModeActive = false
	aiActive = false
	remoteControlActive = false
	_ = os.Remove("/tmp/stream_active")
	if deepThoughtActive {
		go exitDeepThought()
	}
	_ = exec.Command("sudo", "systemctl", "stop", "leafcutter.service").Run()
	return true
}

// handleMoveAction executes movement commands (forward, backward, left, right, about_turn).
// Motors are run in a background goroutine so the command loop is not blocked.
func handleMoveAction(cmd ParsedCommand) bool {
	speed := byte(150)
	if sp, ok := cmd.Modifiers["speed"]; ok {
		if v, err := parsePositiveInt(sp); err == nil && v > 0 {
			speed = byte(v)
		}
	}
	go func() {
		switch cmd.Target {
		case "forward":
			for i := byte(0); i < 4; i++ {
				_ = moveMotor(i, 0, speed)
			}
			time.Sleep(2 * time.Second)
			stopAllMotors()
		case "backward":
			for i := byte(0); i < 4; i++ {
				_ = moveMotor(i, 1, speed)
			}
			time.Sleep(2 * time.Second)
			stopAllMotors()
		case "left":
			_ = moveMotor(0, 1, speed)
			_ = moveMotor(1, 0, speed)
			_ = moveMotor(2, 0, speed)
			_ = moveMotor(3, 1, speed)
			time.Sleep(1 * time.Second)
			stopAllMotors()
		case "right":
			_ = moveMotor(0, 0, speed)
			_ = moveMotor(1, 1, speed)
			_ = moveMotor(2, 1, speed)
			_ = moveMotor(3, 0, speed)
			time.Sleep(1 * time.Second)
			stopAllMotors()
		case "about_turn":
			_ = moveMotor(0, 1, speed)
			_ = moveMotor(1, 0, speed)
			_ = moveMotor(2, 1, speed)
			_ = moveMotor(3, 0, speed)
			time.Sleep(1500 * time.Millisecond)
			stopAllMotors()
		}
	}()
	return true
}

// handleLookAction positions the camera gimbal (pan/tilt).
func handleLookAction(cmd ParsedCommand) bool {
	switch cmd.Target {
	case "up":
		_ = setServo(2, 170)
	case "down":
		_ = setServo(2, 30)
	case "left":
		_ = setServo(1, 150)
	case "right":
		_ = setServo(1, 30)
	case "center":
		_ = setServo(1, 90)
		_ = setServo(2, 75)
	default:
		return false
	}
	return true
}

// handlePlayAction plays audio (pathfinder/adventurer songs).
func handlePlayAction(cmd ParsedCommand) bool {
	file := "Pathfinder Song.mp3"
	if cmd.Target == "adventurer_song" || strings.Contains(cmd.Target, "adventurer") {
		file = "Adventurer Song.mp3"
	}
	go func() {
		exec.Command("mpg123", "/home/pi/the-pathfinder-eye_ai/resources/"+file).Run()
		lastSpokeTime = time.Now()
	}()
	return true
}

// handleReadAction reads documents aloud (law, pledge, aim, motto).
func handleReadAction(cmd ParsedCommand) bool {
	var file string
	switch cmd.Target {
	case "law":
		file = "Pathfinder Law.md"
	case "pledge":
		file = "Pathfinder Pledge.md"
	case "aim":
		file = "Pathfinder Aim.md"
	case "motto":
		file = "Pathfinder Motto.md"
	case "adventurer_law":
		file = "Adventurer Law.md"
	case "adventurer_pledge":
		file = "Adventurer Pledge.md"
	default:
		// Try pathfinder prefix.
		file = "Pathfinder " + strings.Title(cmd.Target) + ".md"
	}
	go readDocument("/home/pi/the-pathfinder-eye_ai/resources/" + file)
	return true
}

// handleActivateAction starts modes: birdwatch, follow, security.
func handleActivateAction(cmd ParsedCommand) bool {
	switch cmd.Target {
	case "birdwatch":
		if ttsEngine != nil {
			_ = ttsEngine.SpeakCritical("bird watch active.")
		}
		birdwatchActive = true
	case "follow":
		if ttsEngine != nil {
			_ = ttsEngine.SpeakCritical("follow mode active.")
		}
		go startFollowMode()
	default:
		return false
	}
	return true
}

// handleDeactivateAction stops active modes.
func handleDeactivateAction(cmd ParsedCommand) bool {
	switch cmd.Target {
	case "birdwatch":
		birdwatchActive = false
	case "follow":
		followModeActive = false
	case "security":
		// security mode stop handled by exit path
	default:
		return false
	}
	return true
}

func parsePositiveInt(s string) (int, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if v <= 0 {
		return 0, fmt.Errorf("not positive")
	}
	return v, nil
}

func enterDeepThought() {
	// RAM guard — refuse to load the 70B model if we don't have room.
	// Llama-3.1-70B-Instruct-Q4_K_S is ~38 GB on disk; even the IQ4_NL
	// variant peaks around 18-22 GB resident during prompt-eval. On an
	// 8 GB Pi 5 we cannot run it: blocking this prevents a hard freeze.
	if _, memFreeMB, _ := getMemInfo(); memFreeMB < MinFreeMBFor70B {
		infoLog.Printf("DEEP_THOUGHT: refused — only %d MB free, need ≥ %d MB",
			memFreeMB, MinFreeMBFor70B)
		if ttsEngine != nil {
			_ = ttsEngine.SpeakCritical(
				"I don't have enough memory for deep thought right now.")
		}
		return
	}

	exec.Command("sudo", "systemctl", "stop", "pathfinder-vision").Run()
	exec.Command("sudo", "systemctl", "stop", "leafcutter").Run()
	time.Sleep(1 * time.Second)

	// Use the structured swap helper instead of sed-editing the unit file.
	// sed-based mutation was racy (concurrent service edits would clobber
	// eachother) and persisted the swap to disk.
	const deepModel = "/home/pi/the-pathfinder-eye_ai/models/Meta-Llama-3.1-70B-Instruct-Q4_K_S.gguf"
	if err := SwapLeafcutterModel(deepModel); err != nil {
		infoLog.Printf("DEEP_THOUGHT: swap failed: %v — reverting", err)
		_ = RevertLeafcutterSwap()
		if ttsEngine != nil {
			_ = ttsEngine.SpeakCritical(
				"Deep thought swap failed. Staying on the normal model.")
		}
		return
	}

	// Re-check after the swap: if RAM has dropped under our floor, revert.
	time.Sleep(2 * time.Second)
	if _, memFreeMB, _ := getMemInfo(); memFreeMB < MinFreeMBFor70B {
		infoLog.Printf("DEEP_THOUGHT: post-swap RAM %d MB below floor, reverting", memFreeMB)
		go RevertLeafcutterSwap()
		return
	}

	if ttsEngine != nil {
		_ = ttsEngine.SpeakCritical("Deep thought initialized. I am ready for complex reasoning.")
	}
}

func exitDeepThought() {
	deepThoughtActive = false
	// Restore the canonical unit (drop in is rejected and daemon-reloaded).
	// Previously this also drove a sed patch to put the 8B model back; now
	// the helper handles all of it via RevertLeafcutterSwap.
	exec.Command("sudo", "systemctl", "stop", "leafcutter").Run()
	time.Sleep(1 * time.Second)
	if err := RevertLeafcutterSwap(); err != nil {
		infoLog.Printf("DEEP_THOUGHT_REVERT: %v", err)
		// Best-effort: still restart on the canonical unit.
		exec.Command("sudo", "systemctl", "start", "leafcutter").Run()
	}
	exec.Command("sudo", "systemctl", "start", "pathfinder-vision").Run()
	if ttsEngine != nil {
		_ = ttsEngine.SpeakCritical("Deep thought suspended. Vision systems online.")
	}
}

func startAIConversationLoop(userName string) {
	atomic.StoreInt32(&commandBusy, 1)
	defer func() {
		atomic.StoreInt32(&commandBusy, 0)
		aiActive = false
	}()

	aiActive = true
	time.Sleep(5 * time.Second)

	// Loop-local transcript with auto-compression. We keep the whole
	// transcript here rather than threading `messages` through
	// ai.go's iterateAgent so the compactor is decoupled from the
	// LLM call shape — a future redesign of iterateAgent can't
	// accidentally break compaction.
	compactor := newCompactor(dendrite)
	transcript := make([]transcriptMsg, 0, 32)

	for aiActive {
		go indicateProcessing()
		samples, err := captureAudio(PerCommandListenSec)
		if err != nil {
			break
		}

		text, _ := transcribeAudio(samples)
		if text == "" {
			continue
		}

		// Robot-native exit keywords (same as before). Kept here
		// rather than in iter because voice_commands.go owns the
		// command vocabulary near the recognition layer.
		if strings.Contains(strings.ToLower(text), "sleep") || strings.Contains(strings.ToLower(text), "stop") || strings.Contains(strings.ToLower(text), "exit") {
			if ttsEngine != nil {
				_ = ttsEngine.SpeakCritical("Understood. Entering power save mode.")
			}
			exec.Command("sudo", "systemctl", "stop", "leafcutter.service").Run()
			aiActive = false
			if deepThoughtActive {
				go exitDeepThought()
			}
			break
		}

		// Append user turn BEFORE the LLM call so the compactor can
		// see the prompt's true size on the next iteration.
		transcript = append(transcript, transcriptMsg{Role: "user", Body: text})

		worldContext := GetWorldStatePrompt()
		if ttsEngine != nil {
			_ = ttsEngine.Speak("One moment, I am thinking...")
		}

		speech, err := aiBrain.Process(text, worldContext)
		if err == nil && speech != "" {
			transcript = append(transcript, transcriptMsg{Role: "assistant", Body: speech})
			// Compress AFTER the assistant turn records so the next
			// iteration's user prompt sees a trimmed history.
			transcript = compactor.MaybeCompress(transcript)
			go indicateSuccess()
			if ttsEngine != nil {
				_ = ttsEngine.Speak(speech)
			}
		} else if err != nil {
			if ttsEngine != nil {
				_ = ttsEngine.Speak("I am having trouble with my neural link right now.")
			}
			safeLogf("ERROR", "AIConversationLoop brain error: %s", redactOnce(err.Error()))
		}
	}
}

func readDocument(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if ttsEngine != nil {
			_ = ttsEngine.Speak(line)
			time.Sleep(2 * time.Second)
		}
	}
	lastSpokeTime = time.Now()
}

func startJapaneseTranslationLoop() {
	atomic.StoreInt32(&commandBusy, 1)
	defer atomic.StoreInt32(&commandBusy, 0)

	for {
		go indicateProcessing()
		samples, err := captureAudio(7)
		if err != nil {
			break
		}
		text, _ := transcribeAudio(samples)
		if strings.Contains(strings.ToLower(text), "stop") {
			if ttsEngine != nil {
				_ = ttsEngine.SpeakCritical("exiting translation mode")
			}
			break
		}
		translation, _ := translateJapaneseToEnglish(samples)
		translation = strings.TrimSpace(translation)
		if translation != "" {
			go indicateSuccess()
			if ttsEngine != nil {
				_ = ttsEngine.Speak(translation)
			}
		}
	}
	lastSpokeTime = time.Now()
}

func handleFullHardwareTest_Direct() {
	go indicateProcessing()
	_ = ttsEngine.SpeakCritical("Initiating hardware sequence.")
	s := byte(150)
	for i := byte(0); i < 4; i++ {
		go indicateProcessing()
		_ = moveMotor(i, 0, s)
		time.Sleep(400 * time.Millisecond)
		_ = moveMotor(i, 0, 0)
		time.Sleep(200 * time.Millisecond)
	}
	stopAllMotors()
	_ = setServo(1, 45)
	time.Sleep(400 * time.Millisecond)
	_ = setServo(1, 135)
	time.Sleep(400 * time.Millisecond)
	_ = setServo(1, 90)
	_ = setServo(2, 75)
	go indicateSuccess()
	_ = ttsEngine.SpeakCritical("Test complete.")
	lastSpokeTime = time.Now()
}
