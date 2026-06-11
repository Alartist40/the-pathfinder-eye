package main

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"
)

var (
	trackingMode = false
	aiActive     = false
	commandBusy  int32
	lastSpokeTime time.Time
	remoteControlActive bool
	deepThoughtActive bool
)

func isWakeWord(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	keywords := []string{"instruction", "instruct", "destruction", "restruction", "direction", "introduction"}
	for _, kw := range keywords {
		if strings.Contains(t, kw) { return true }
	}
	return false
}

func handleCommandSequence() {
	defer atomic.StoreInt32(&commandBusy, 0)
	time.Sleep(1200 * time.Millisecond)

	speaker, _ := visionDB.GetCurrentSpeaker()
	figure, recognized := authority.VerifyFigure(speaker.FaceID)
	level := LevelGuest
	name := "Guest"
	if recognized {
		level = figure.Level
		name = figure.Name
	}
	
	for i := 0; i < 2; i++ {
		go indicateCommandAck()
		samples, err := captureAudio(6)
		if err != nil { break }
		cmdText, _ := transcribeAudio(samples)
		cmd := strings.ToLower(cmdText)
		if cmd == "" { continue }

        if isWakeWord(cmd) {
            if ttsEngine != nil { _ = ttsEngine.SpeakCritical("resetting") }
            i = -1 
            continue
        }

		if !authority.CanExecuteCommand(level, cmd) {
			if ttsEngine != nil { _ = ttsEngine.SpeakCritical("Permission denied") }
			return
		}

		if processDirectCommand(cmd, level, name) {
			return
		}
	}
	
	go indicateWarning()
	if ttsEngine != nil { _ = ttsEngine.SpeakCritical("I did not understand.") }
}

func processDirectCommand(cmd string, level AuthorityLevel, name string) bool {
	if strings.Contains(cmd, "test") || strings.Contains(cmd, "diagnostic") {
		if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood") }
		go handleFullHardwareTest_Direct()
		return true
	}

    if strings.Contains(cmd, "about turn") || strings.Contains(cmd, "turn about") {
		if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood") }
		_ = moveMotor(0, 1, 150)
        _ = moveMotor(1, 0, 150)
        _ = moveMotor(2, 1, 150)
        _ = moveMotor(3, 0, 150)
		time.Sleep(1500 * time.Millisecond)
        stopAllMotors()
		return true
	}

    if strings.Contains(cmd, "follow") {
		if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood. follow mode active.") }
		go startFollowMode()
		return true
	}

    if strings.Contains(cmd, "bird") || strings.Contains(cmd, "third") || strings.Contains(cmd, "beard") || strings.Contains(cmd, "word") {
        if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood. bird watch active.") }
        birdwatchActive = true
        return true
    }

	if strings.Contains(cmd, "song") {
		file := "Pathfinder Song.mp3"
        var speechResponse string = "understood. playing pathfinder soundtrack."
		if strings.Contains(cmd, "adventurer") || strings.Contains(cmd, "adventure") { 
            file = "Adventurer Song.mp3" 
            speechResponse = "understood. playing adventurer soundtrack."
        }
        if ttsEngine != nil { _ = ttsEngine.SpeakCritical(speechResponse) }
		go func() {
            exec.Command("mpg123", "/home/pi/the-pathfinder-eye_ai/resources/"+file).Run()
            lastSpokeTime = time.Now()
        }()
		return true
	}

	if strings.Contains(cmd, "law") || strings.Contains(cmd, "pledge") || strings.Contains(cmd, "aim") || strings.Contains(cmd, "motto") {
		target := "Pathfinder Law.md"
        var speechResponse string = "understood. reading pathfinder document."
		if strings.Contains(cmd, "pledge") { target = "Pathfinder Pledge.md" }
		if strings.Contains(cmd, "aim") { target = "Pathfinder Aim.md" }
		if strings.Contains(cmd, "motto") { target = "Pathfinder Motto.md" }
		if strings.Contains(cmd, "adventurer") || strings.Contains(cmd, "adventure") { 
			target = strings.Replace(target, "Pathfinder", "Adventurer", 1)
            speechResponse = "understood. reading adventurer document."
		}
        if ttsEngine != nil { _ = ttsEngine.SpeakCritical(speechResponse) }
		go readDocument("/home/pi/the-pathfinder-eye_ai/resources/"+target)
		return true
	}

	if strings.Contains(cmd, "translate") {
		if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood. japanese mode active. say stop to exit.") }
		go startJapaneseTranslationLoop()
		return true
	}
	
	if strings.Contains(cmd, "forward") {
		if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood") }
		for i := byte(0); i < 4; i++ { _ = moveMotor(i, 0, 150) }
		time.Sleep(2 * time.Second); stopAllMotors()
		return true
	}
	if strings.Contains(cmd, "back") {
		if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood") }
		for i := byte(0); i < 4; i++ { _ = moveMotor(i, 1, 150) }
		time.Sleep(2 * time.Second); stopAllMotors()
		return true
	}
	if strings.Contains(cmd, "left") {
		if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood") }
		_ = moveMotor(0, 1, 150); _ = moveMotor(1, 0, 150); _ = moveMotor(2, 0, 150); _ = moveMotor(3, 1, 150)
		time.Sleep(1 * time.Second); stopAllMotors()
		return true
	}
	if strings.Contains(cmd, "right") {
		if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood") }
		_ = moveMotor(0, 0, 150); _ = moveMotor(1, 1, 150); _ = moveMotor(2, 1, 150); _ = moveMotor(3, 0, 150)
		time.Sleep(1 * time.Second); stopAllMotors()
		return true
	}

    if strings.Contains(cmd, "remote control") || strings.Contains(cmd, "control mode") {
        if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood. remote control mode enabled.") }
        remoteControlActive = true
        _ = os.WriteFile("/tmp/stream_active", []byte("1"), 0644)
        return true
    }

    if strings.Contains(cmd, "deep thought") || strings.Contains(cmd, "thinking mode") {
        if ttsEngine != nil { _ = ttsEngine.SpeakCritical("Entering deep thought. Suspending vision systems.") }
        deepThoughtActive = true
        go enterDeepThought()
        return true
    }

    if strings.Contains(cmd, "exit") || strings.Contains(cmd, "stop") || strings.Contains(cmd, "sleep") {
        if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood. system idle.") }
        stopAllMotors()
        birdwatchActive = false
        followModeActive = false
        aiActive = false
        remoteControlActive = false
        _ = os.Remove("/tmp/stream_active")
        if deepThoughtActive {
            go exitDeepThought()
        }
        exec.Command("sudo", "systemctl", "stop", "leafcutter.service").Run()
        return true
    }
	
	if strings.Contains(cmd, "look up") {
		if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood") }
		_ = setServo(2, 170); return true
	}
	if strings.Contains(cmd, "look down") {
		if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood") }
		_ = setServo(2, 30); return true
	}
	if strings.Contains(cmd, "look left") {
		if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood") }
		_ = setServo(1, 150); return true
	}
	if strings.Contains(cmd, "look right") {
		if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood") }
		_ = setServo(1, 30); return true
	}
	if strings.Contains(cmd, "center") {
		if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood") }
		_ = setServo(1, 90); _ = setServo(2, 75); return true
	}

	if strings.Contains(cmd, "attention") {
		if ttsEngine != nil { _ = ttsEngine.SpeakCritical("understood. activating AI intelligence.") }
		exec.Command("sudo", "systemctl", "start", "leafcutter.service").Run()
        go startAIConversationLoop(name)
		return true
	}
	return false
}

func enterDeepThought() {
    exec.Command("sudo", "systemctl", "stop", "pathfinder-vision").Run()
    exec.Command("sudo", "systemctl", "stop", "leafcutter").Run()
    time.Sleep(1 * time.Second)
    // Switch to 70B
    exec.Command("sudo", "sed", "-i", "s|--model .*|--model /home/pi/the-pathfinder-eye_ai/models/Meta-Llama-3.1-70B-Instruct-Q4_K_S.gguf --port 8081 --engine native-streaming|", "/etc/systemd/system/leafcutter.service").Run()
    exec.Command("sudo", "systemctl", "daemon-reload").Run()
    exec.Command("sudo", "systemctl", "start", "leafcutter").Run()
    if ttsEngine != nil { _ = ttsEngine.SpeakCritical("Deep thought initialized. I am ready for complex reasoning.") }
}

func exitDeepThought() {
    deepThoughtActive = false
    exec.Command("sudo", "systemctl", "stop", "leafcutter").Run()
    time.Sleep(1 * time.Second)
    // Switch back to 8B
    exec.Command("sudo", "sed", "-i", "s|--model .*|--model /home/pi/the-pathfinder-eye_ai/models/Hermes-3-Llama-3.1-8B.Q4_K_M.gguf --port 8081 --engine native-streaming|", "/etc/systemd/system/leafcutter.service").Run()
    exec.Command("sudo", "systemctl", "daemon-reload").Run()
    exec.Command("sudo", "systemctl", "start", "leafcutter").Run()
    exec.Command("sudo", "systemctl", "start", "pathfinder-vision").Run()
    if ttsEngine != nil { _ = ttsEngine.SpeakCritical("Deep thought suspended. Vision systems online.") }
}

func startAIConversationLoop(userName string) {
    atomic.StoreInt32(&commandBusy, 1)
    defer atomic.StoreInt32(&commandBusy, 0)
    
    aiActive = true
    time.Sleep(5 * time.Second)
    
    for aiActive {
        go indicateProcessing()
        samples, err := captureAudio(8)
        if err != nil { break }
        
        text, _ := transcribeAudio(samples)
        if text == "" { continue }
        
        if strings.Contains(strings.ToLower(text), "sleep") || strings.Contains(strings.ToLower(text), "stop") || strings.Contains(strings.ToLower(text), "exit") {
            if ttsEngine != nil { _ = ttsEngine.SpeakCritical("Understood. Entering power save mode.") }
            exec.Command("sudo", "systemctl", "stop", "leafcutter.service").Run()
            aiActive = false
            if deepThoughtActive { go exitDeepThought() }
            break
        }

        worldContext := GetWorldStatePrompt()
        if ttsEngine != nil { _ = ttsEngine.Speak("One moment, I am thinking...") }

        speech, err := aiBrain.Process(text, worldContext)
        if err == nil && speech != "" {
            go indicateSuccess()
            if ttsEngine != nil { _ = ttsEngine.Speak(speech) }
        } else if err != nil {
            if ttsEngine != nil { _ = ttsEngine.Speak("I am having trouble with my neural link right now.") }
        }
    }
}

func readDocument(path string) {
	file, err := os.Open(path)
	if err != nil { return }
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") { continue }
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
        if err != nil { break }
        text, _ := transcribeAudio(samples)
        if strings.Contains(strings.ToLower(text), "stop") {
            if ttsEngine != nil { _ = ttsEngine.SpeakCritical("exiting translation mode") }
            break
        }
        translation, _ := translateJapaneseToEnglish(samples)
        translation = strings.TrimSpace(translation)
        if translation != "" {
            go indicateSuccess()
            if ttsEngine != nil { _ = ttsEngine.Speak(translation) }
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
