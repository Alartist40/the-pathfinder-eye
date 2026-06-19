package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	I2C_DEVICE = "/dev/i2c-1"
	I2C_SLAVE  = 0x0703
	I2C_ADDR   = 0x2B
)

var (
	i2cFile         *os.File
	i2cMutex        sync.Mutex
	infoLog         *log.Logger
	errorLog        *log.Logger
	startTime       time.Time
	calibration     *ServoCalibration
	version         = "v7.4-STARLING-CORTEX"
	aiMutex         sync.Mutex
	visionDB        *VisionDB
	dendrite        *Dendrite
	authority       *AuthorityManager
	aiBrain         *AIBrain
	executor        *ActionExecutor
	birdDB          *BirdWatchDB
	tracker         *BirdTracker
	gimbal          *GimbalTracker
	cortex          *AICortex
	birdwatchActive bool
	startupOnce     sync.Once
	shutdownCtx     context.Context
	shutdownCancel  context.CancelFunc
)

// LED color constants for WS2812 (from original Yahboom library)
const (
	LEDColorRed    = 0
	LEDColorGreen  = 1
	LEDColorBlue   = 2
	LEDColorYellow = 3
)

// setLEDAll controls all 10 WS2812 LEDs at once (register 0x03)
func setLEDAll(state, color byte) error {
	if state > 1 {
		state = 1
	}
	if color > 3 {
		color = 3
	}
	return i2cWrite(0x03, []byte{state, color})
}

// setLEDAlone controls a single WS2812 LED (register 0x04)
func setLEDAlone(number, state, color byte) error {
	if number < 1 {
		number = 1
	}
	if number > 10 {
		number = 10
	}
	if state > 1 {
		state = 1
	}
	if color > 3 {
		color = 3
	}
	return i2cWrite(0x04, []byte{number, state, color})
}

// --- LED SEMANTIC INDICATORS ---

func indicateReady() {
	_ = setLEDAll(1, LEDColorGreen)
	time.Sleep(200 * time.Millisecond)
	_ = setLEDAll(0, 0)
}

func indicateSuccess() {
	for i := 0; i < 2; i++ {
		_ = setLEDAll(0, 0)
		time.Sleep(50 * time.Millisecond)
		_ = setLEDAll(1, LEDColorGreen)
		time.Sleep(120 * time.Millisecond)
	}
	_ = setLEDAll(0, 0)
}

func indicateError() {
	for i := 0; i < 3; i++ {
		_ = setLEDAll(1, LEDColorRed)
		time.Sleep(150 * time.Millisecond)
		_ = setLEDAll(0, 0)
		time.Sleep(100 * time.Millisecond)
	}
	_ = setLEDAll(0, 0)
}

func indicateWarning() {
	_ = setLEDAll(1, LEDColorYellow)
	time.Sleep(200 * time.Millisecond)
	_ = setLEDAll(0, 0)
}

func indicateProcessing() {
	_ = setLEDAll(1, LEDColorBlue)
	time.Sleep(200 * time.Millisecond)
	_ = setLEDAll(0, 0)
}

func indicateCommandAck() {
	_ = setLEDAll(1, LEDColorYellow)
	time.Sleep(150 * time.Millisecond)
	_ = setLEDAll(0, 0)
}

func indicateWakeWord() {
	for i := 0; i < 2; i++ {
		_ = setLEDAll(0, 0)
		time.Sleep(50 * time.Millisecond)
		_ = setLEDAll(1, LEDColorGreen)
		time.Sleep(100 * time.Millisecond)
	}
	_ = setLEDAll(0, 0)
}

func indicateSleep() { _ = setLEDAll(0, 0) }

func indicateStartup() {
	colors := []byte{LEDColorRed, LEDColorGreen, LEDColorBlue, LEDColorYellow}
	for _, c := range colors {
		_ = setLEDAll(1, c)
		time.Sleep(200 * time.Millisecond)
	}
	_ = setLEDAll(0, 0)
}

func startListeningHeartbeat() {
	ticker := time.NewTicker(15 * time.Second)
	for range ticker.C {
		_ = setLEDAlone(1, 1, LEDColorGreen)
		time.Sleep(80 * time.Millisecond)
		_ = setLEDAlone(1, 0, 0)
	}
}

type ServoCalibration struct {
	Version      string    `json:"version"`
	CalibratedAt time.Time `json:"calibrated_at"`
	Pan          ServoAxis `json:"pan"`
	Tilt         ServoAxis `json:"tilt"`
}

type ServoAxis struct {
	Current int `json:"current"`
	Min     int `json:"min"`
	Max     int `json:"max"`
	Center  int `json:"center"`
}

func initLogging() error {
	logPath := "../logs/go_brain.log"
	_ = os.MkdirAll("../logs", 0755)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		infoLog = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
		errorLog = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
		return nil
	}
	infoLog = log.New(logFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLog = log.New(logFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	return nil
}

func initI2C() error {
	f, err := os.OpenFile(I2C_DEVICE, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), I2C_SLAVE, uintptr(I2C_ADDR)); errno != 0 {
		f.Close()
		return fmt.Errorf("ioctl failed: %v", errno)
	}
	i2cFile = f
	return nil
}

func i2cWrite(reg byte, data []byte) error {
	i2cMutex.Lock()
	defer i2cMutex.Unlock()
	if i2cFile == nil {
		return fmt.Errorf("i2c nil")
	}
	buf := append([]byte{reg}, data...)
	_, err := i2cFile.Write(buf)
	return err
}

func setServo(id, angle byte) error {
	if angle > 180 {
		angle = 180
	}
	if id == 2 && angle > 100 {
		angle = 100
	}
	return i2cWrite(0x02, []byte{id, angle})
}

func moveMotor(id, dir, speed byte) error {
	return i2cWrite(0x01, []byte{id, dir, speed})
}

func stopAllMotors() {
	for i := byte(0); i < 4; i++ {
		moveMotor(i, 0, 0)
	}
}

func loadCalibration() *ServoCalibration {
	path := "../config/servo_calibration.json"
	data, err := ioutil.ReadFile(path)
	if err == nil {
		var c ServoCalibration
		if err := json.Unmarshal(data, &c); err == nil {
			return &c
		}
	}
	return &ServoCalibration{
		Version: "1.0",
		Pan:     ServoAxis{Current: 90, Center: 90, Min: 0, Max: 180},
		Tilt:    ServoAxis{Current: 75, Center: 75, Min: 0, Max: 100},
	}
}

func getSystemStats() map[string]interface{} {
	cmdRAM := exec.Command("sh", "-c", "free -m | grep Mem | awk '{print $3}'")
	outRAM, _ := cmdRAM.Output()
	cmdTemp := exec.Command("vcgencmd", "measure_temp")
	outTemp, _ := cmdTemp.Output()
	temp := strings.TrimPrefix(strings.TrimSuffix(string(outTemp), "\n"), "temp=")
	return map[string]interface{}{
		"ram_used": strings.TrimSpace(string(outRAM)) + "MB",
		"temp":     temp,
		"uptime":   time.Since(startTime).String(),
	}
}

func main() {
	startTime = time.Now()
	shutdownCtx, shutdownCancel = context.WithCancel(context.Background())
	initLogging()
	initI2C()
	calibration = loadCalibration()

	visionDB, _ = initVisionDB()
	go visionPollerLoop()
	var err error
	dendrite, err = initDendrite()
	if err != nil {
		log.Fatalf("Failed to init dendrite: %v", err)
	}
	authority, _ = initAuthority(dendrite)

	birdDB, _ = initBirdWatchDB("../db/birdwatch.sqlite")
	tracker = NewBirdTracker()
	gimbal = NewGimbalTracker()

	modelPath := "../models/Hermes-3-Llama-3.1-8B.Q4_K_M.gguf"
	aiBrain = newAIBrain(modelPath)
	executor = &ActionExecutor{dendrite: dendrite}

	ttsEngine, _ = initTTS()
	voiceModel := "../models/ggml-tiny.en.bin"

	cortex = newCortex()
	go func() {
		_ = initVoice(voiceModel)
		go cortex.StartUnifiedAwareness()
		go startListeningHeartbeat()
	}()

	// Audio setup and startup announcement
	go func() {
		startupOnce.Do(func() {
			_ = exec.Command("/home/pi/the-pathfinder-eye_ai/scripts/set_audio.sh").Run()
			time.Sleep(1 * time.Second)
			if ttsEngine != nil {
				_ = ttsEngine.SpeakCritical("Pathfinder Eye online")
				time.Sleep(500 * time.Millisecond)
				_ = ttsEngine.SpeakCritical("Motors and servos ready")
				time.Sleep(500 * time.Millisecond)
				_ = ttsEngine.SpeakCritical("Say Instruction to begin")
				time.Sleep(500 * time.Millisecond)
				_ = ttsEngine.SpeakCritical("Ready")
			}
			go indicateStartup()
		})
	}()

	// Graceful shutdown: cancel context on signal, then clean up
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		infoLog.Println("SHUTDOWN: signal received, cleaning up...")
		shutdownCancel()
		stopAllMotors()
		if i2cFile != nil {
			i2cFile.Close()
		}
		if ttsEngine != nil {
			close(ttsQueue)
		}
		if visionDB != nil {
			visionDB.Close()
		}
		if dendrite != nil {
			dendrite.Close()
		}
		if birdDB != nil {
			birdDB.Close()
		}
		infoLog.Println("SHUTDOWN: complete")
		os.Exit(0)
	}()

	http.HandleFunc("/move", authWrap(handleMove))
	http.HandleFunc("/camera", authWrap(handleCamera))
	http.HandleFunc("/ai/think", authWrap(handleAIThink))
	http.HandleFunc("/stream", authWrap(func(w http.ResponseWriter, r *http.Request) {
		data, err := ioutil.ReadFile("/tmp/vision_feed.jpg")
		if err != nil || len(data) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"no vision feed","hint":"is camera-feed.service running?"}`))
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(data)
	}))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "online", "version": version})
	})

	infoLog.Println("--- THE-PATHFINDER-EYE STARLING CORTEX READY ---")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleAIThink(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	worldCtx := GetWorldStatePrompt()

	route, confidence := ClassifyRoute(q)
	infoLog.Printf("AI_THINK: route=%s confidence=%.2f query=%q", route, confidence, q)

	speech, _ := aiBrain.Process(q, worldCtx)
	json.NewEncoder(w).Encode(map[string]string{
		"speech":     speech,
		"route":      route.String(),
		"confidence": fmt.Sprintf("%.2f", confidence),
	})
}

func handleMove(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	s := byte(150)
	switch dir {
	case "forward":
		for i := byte(0); i < 4; i++ {
			moveMotor(i, 0, s)
		}
	case "backward":
		for i := byte(0); i < 4; i++ {
			moveMotor(i, 1, s)
		}
	case "stop":
		stopAllMotors()
	}
}

func handleCamera(w http.ResponseWriter, r *http.Request) {
	axis := r.URL.Query().Get("axis")
	val := 0
	fmt.Sscanf(r.URL.Query().Get("val"), "%d", &val)
	if axis == "pan" {
		calibration.Pan.Current += val
	} else {
		calibration.Tilt.Current -= val
	}
	_ = setServo(1, byte(calibration.Pan.Current))
	_ = setServo(2, byte(calibration.Tilt.Current))
}
