/**
 * THE-PATHFINDER-EYE : Voice Management Module (v3.1 - Protected Feedback)
 */

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

type TTSEngine struct {
	Device         string
	Speed          float32
	Volume         int
	currentTTS     *exec.Cmd
	isCritical     bool // If true, this speech cannot be interrupted
	mu             sync.Mutex
}

var (
	whisperCtx   whisper.Context
	isVoiceReady bool
	ttsEngine    *TTSEngine
	ttsQueue     chan ttsMessage
)

type ttsMessage struct {
	text     string
	critical bool
}

func initVoice(modelPath string) error {
	infoLog.Printf("VOICE_INIT: Loading model from %s", modelPath)
	model, err := whisper.New(modelPath)
	if err != nil { return err }
	ctx, err := model.NewContext()
	if err != nil { return err }
	_ = ctx.SetLanguage("en")
        ctx.SetThreads(2)
	whisperCtx = ctx
	isVoiceReady = true
	return nil
}

var preferredMicDevice string

func captureAudio(durationSeconds int) ([]float32, error) {
	tempFile := "/tmp/capture.wav"
	devices := []string{"plughw:0,0", "plughw:1,0", "plughw:0,0", "default"}
	
	if preferredMicDevice != "" {
		devices = append([]string{preferredMicDevice}, devices...)
	}

	for _, dev := range devices {
		_ = os.Remove(tempFile)
		cmd := exec.Command("arecord", "-D", dev, "-d", fmt.Sprintf("%d", durationSeconds),
			"-f", "S16_LE", "-r", "16000", "-c", "1", tempFile)
		
		if err := cmd.Run(); err == nil {
			if fi, err := os.Stat(tempFile); err == nil && fi.Size() > 1024 {
				preferredMicDevice = dev
				return readWavSamples(tempFile)
			}
		}
	}
	return nil, fmt.Errorf("no working mic")
}

func readWavSamples(tempFile string) ([]float32, error) {
	data, err := ioutil.ReadFile(tempFile)
	if err != nil { return nil, err }
	samples := make([]float32, (len(data)-44)/2)
	for i := 0; i < len(samples); i++ {
		val := int16(data[44+i*2]) | int16(data[44+i*2+1])<<8
		samples[i] = float32(val) / 32768.0
	}
	return samples, nil
}

func transcribeAudio(samples []float32) (string, error) {
	if !isVoiceReady { return "", fmt.Errorf("deaf") }
	_ = whisperCtx.Process(samples, nil, nil, nil)
	var res string
	for {
		s, err := whisperCtx.NextSegment()
		if err != nil { break }
		res += s.Text
	}
	return res, nil
}

func translateJapaneseToEnglish(samples []float32) (string, error) {
    if !isVoiceReady { return "", fmt.Errorf("deaf") }
    whisperCtx.SetTranslate(true) 
    _ = whisperCtx.SetLanguage("ja") 
    if err := whisperCtx.Process(samples, nil, nil, nil); err != nil { return "", err }
    var res string
    for {
        s, err := whisperCtx.NextSegment()
        if err != nil { break }
        res += s.Text
    }
    return res, nil
}

func initTTS() (*TTSEngine, error) {
	engine := &TTSEngine{ Device: "plughw:0,0", Speed: 1.0, Volume: 170 }
	ttsQueue = make(chan ttsMessage, 10)
	go ttsWorker(engine)
	return engine, nil
}

func ttsWorker(t *TTSEngine) {
	for msg := range ttsQueue {
		_ = t.executeSpeak(msg.text, msg.critical)
	}
}

func (t *TTSEngine) executeSpeak(text string, critical bool) error {
	if text == "" { return nil }

	// Only kill if the CURRENT speech is NOT critical
	t.mu.Lock()
	if t.isCritical {
		t.mu.Unlock()
		// Wait for critical speech to finish naturally
		for {
			t.mu.Lock()
			if !t.isCritical { t.mu.Unlock(); break }
			t.mu.Unlock()
			runtime.Gosched()
		}
	} else {
		t.mu.Unlock()
		killTTS()
	}

	t.mu.Lock()
	t.isCritical = critical
	t.mu.Unlock()

	wavPath := "/tmp/speech.wav"
	cmd := exec.Command("/home/pi/piper/piper/piper", 
		"--model", "/home/pi/piper/en_US-lessac-medium.onnx", 
		"--output_file", wavPath,
	)
	stdin, _ := cmd.StdinPipe()
	go func() { defer stdin.Close(); stdin.Write([]byte(text)) }()
	if err := cmd.Run(); err != nil { return err }

	playCmd := exec.Command("aplay", "-D", t.Device, wavPath)
	t.mu.Lock()
	t.currentTTS = playCmd
	t.mu.Unlock()

	_ = playCmd.Run()

	t.mu.Lock()
	t.currentTTS = nil
	t.isCritical = false
	t.mu.Unlock()

	return nil
}

func (t *TTSEngine) Speak(text string) error {
	if t == nil { return nil }
	ttsQueue <- ttsMessage{text: text, critical: false}
	return nil
}

func (t *TTSEngine) SpeakCritical(text string) error {
	if t == nil { return nil }
	ttsQueue <- ttsMessage{text: text, critical: true}
	return nil
}

func killTTS() {
	if ttsEngine == nil { return }
	ttsEngine.mu.Lock()
	// Never kill if it's marked critical (feedback like "Yes" or "Understood")
	if ttsEngine.isCritical {
		ttsEngine.mu.Unlock()
		return
	}
	if ttsEngine.currentTTS != nil && ttsEngine.currentTTS.Process != nil {
		_ = ttsEngine.currentTTS.Process.Kill()
		_ = ttsEngine.currentTTS.Wait()
		ttsEngine.currentTTS = nil
	}
	ttsEngine.mu.Unlock()
	_ = exec.Command("pkill", "-9", "aplay").Run()
}

func speak(text string) error {
	if ttsEngine == nil { return nil }
	return ttsEngine.Speak(text)
}

func speakCritical(text string) error {
	if ttsEngine == nil { return nil }
	return ttsEngine.SpeakCritical(text)
}
