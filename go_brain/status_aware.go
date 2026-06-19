/**
 * THE-PATHFINDER-EYE : System Status Module (v1.0)
 * Quirky, human-like responses based on actual robot health
 *
 * This makes the robot feel ALIVE - it complains when struggling
 * and feels happy when everything runs smooth.
 */

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type SystemStatus struct {
	CPUPercent  float64
	CPUCores    int
	RAMUsedMB   int
	RAMTotalMB  int
	RAMPercent  float64
	TempCelsius float64
	LoadAvg1Min float64
	DiskUsedGB  int
	DiskTotalGB int
}

func GetSystemStatus() *SystemStatus {
	s := &SystemStatus{}
	s.CPUPercent = getCPUUsage()
	s.CPUCores = runtime.NumCPU()
	s.LoadAvg1Min = getLoadAverage()
	s.RAMUsedMB, s.RAMTotalMB, s.RAMPercent = getMemInfo()
	s.TempCelsius = getCPUTemp()
	s.DiskUsedGB, s.DiskTotalGB = getDiskInfo()
	return s
}

func (s *SystemStatus) GetHealthReport() string {
	var issues []string
	var mood string

	// Analyze issues
	if s.TempCelsius >= 80 {
		issues = append(issues, "feeling very hot")
	}
	if s.TempCelsius >= 70 && s.TempCelsius < 80 {
		issues = append(issues, "feeling warm")
	}
	if s.TempCelsius >= 60 && s.TempCelsius < 70 {
		issues = append(issues, "a bit toasty")
	}

	if s.RAMPercent >= 90 {
		issues = append(issues, "completely overwhelmed by memories")
	}
	if s.RAMPercent >= 80 && s.RAMPercent < 90 {
		issues = append(issues, "my brain is full")
	}
	if s.RAMPercent >= 70 && s.RAMPercent < 80 {
		issues = append(issues, "thinking hard")
	}

	if s.CPUPercent >= 90 {
		issues = append(issues, "my head is spinning")
	}
	if s.CPUPercent >= 80 && s.CPUPercent < 90 {
		issues = append(issues, "I have a headache")
	}
	if s.CPUPercent >= 70 && s.CPUPercent < 80 {
		issues = append(issues, "feeling mentally busy")
	}

	if s.LoadAvg1Min >= 3.0 {
		issues = append(issues, "everything at once is too much")
	}
	if s.LoadAvg1Min >= 2.0 && s.LoadAvg1Min < 3.0 {
		issues = append(issues, "I'm juggling a lot")
	}

	if len(issues) == 0 {
		mood = "I'm doing great! Everything feels smooth and easy."
	} else if len(issues) == 1 {
		mood = "I'm " + issues[0] + "."
	} else if len(issues) == 2 {
		mood = "I'm " + issues[0] + " and " + issues[1] + "."
	} else {
		mood = "I'm " + issues[0] + ", " + issues[1] + ", and " + issues[2] + "."
	}

	return fmt.Sprintf("System check: CPU at %.0f%% (load: %.2f), RAM at %.0f%%, temperature %.0f°C. %s",
		s.CPUPercent, s.LoadAvg1Min, s.RAMPercent, s.TempCelsius, mood)
}

// === Platform-specific collectors ===

func getCPUUsage() float64 {
	// Read from /proc/stat
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	line := scanner.Text()

	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0
	}

	var total, idle uint64
	for i := 1; i < len(fields); i++ {
		val, _ := strconv.ParseUint(fields[i], 10, 64)
		total += val
		if i == 4 { // idle is field 5 (index 4 after "cpu")
			idle = val
		}
	}

	return (1.0 - float64(idle)/float64(total)) * 100
}

func getLoadAverage() float64 {
	file, err := os.Open("/proc/loadavg")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 1 {
			val, _ := strconv.ParseFloat(fields[0], 64)
			return val
		}
	}
	return 0
}

func getMemInfo() (used int, total int, percent float64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0
	}
	defer file.Close()

	var memTotal, memFree, buffers, cached uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseUint(fields[1], 10, 64) // KB
		switch fields[0] {
		case "MemTotal:":
			memTotal = val
		case "MemFree:":
			memFree = val
		case "Buffers:":
			buffers = val
		case "Cached:":
			cached = val
		}
	}

	if memTotal == 0 {
		return 0, 0, 0
	}

	available := memFree + buffers + cached
	used = int((memTotal - available) / 1024) // MB
	total = int(memTotal / 1024)              // MB
	percent = float64(memTotal-available) / float64(memTotal) * 100
	return
}

func getCPUTemp() float64 {
	// Try common thermal zone paths
	paths := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/thermal/thermal_zone1/temp",
		"/sys/class/hwmon/hwmon0/temp1_input",
		"/sys/class/hwmon/hwmon1/temp1_input",
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		trimmed := strings.TrimSpace(string(data))
		if val, err := strconv.ParseFloat(trimmed, 64); err == nil {
			// Temperature is usually in millidegrees
			if val > 1000 {
				return val / 1000.0
			}
			return val
		}
	}

	// Fallback: use vcgencmd on Raspberry Pi
	cmd := exec.Command("vcgencmd", "measure_temp")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	// Output format: temp=42.5'C
	outputStr := strings.ReplaceAll(string(output), "temp=", "")
	outputStr = strings.ReplaceAll(outputStr, "'C", "")
	outputStr = strings.TrimSpace(outputStr)
	if val, err := strconv.ParseFloat(outputStr, 64); err == nil {
		return val
	}

	return 0
}

func getDiskInfo() (used int, total int) {
	// Simple df -h parsing
	cmd := exec.Command("df", "-B1", "/")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, 0
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return 0, 0
	}

	usedBlocks, _ := strconv.ParseInt(fields[2], 10, 64)
	totalBlocks, _ := strconv.ParseInt(fields[3], 10, 64)

	used = int(usedBlocks / (1024 * 1024 * 1024))   // GB
	total = int(totalBlocks / (1024 * 1024 * 1024)) // GB
	return
}

// QuirkyStatusResponse returns a playful, human-like status response
// with specific complaints based on actual system metrics
func QuirkyStatusResponse() string {
	status := GetSystemStatus()

	// Priority-based complaints: temp > CPU > RAM > load
	if status.TempCelsius >= 85 {
		return "I'm overheating! Please turn on a fan or I might pass out!"
	}
	if status.TempCelsius >= 75 {
		return "I'm feeling really hot... my circuits are sweating! Can we cool down the room?"
	}
	if status.TempCelsius >= 65 {
		return "I'm getting a bit warm. Nothing serious, but I wouldn't say no to some AC."
	}

	if status.RAMPercent >= 90 {
		return "My brain is completely full! I'm forgetting things... too much to process!"
	}
	if status.RAMPercent >= 80 {
		return "My mind is crowded. I'm trying to remember everything but it's getting messy in here!"
	}
	if status.RAMPercent >= 70 {
		return "I'm thinking hard about a lot of things right now. Ask me again in a bit!"
	}

	if status.CPUPercent >= 85 {
		return "I have such a bad headache... too many thoughts at once!"
	}
	if status.CPUPercent >= 70 {
		return "My head is spinning a little. Give me a moment to catch up!"
	}

	if status.LoadAvg1Min >= 3.0 {
		return "Everything is happening at once! I'm trying my best but it's overwhelming!"
	}
	if status.LoadAvg1Min >= 2.0 {
		return "I'm juggling a lot of tasks right now. Bear with me!"
	}

	// All good!
	responses := []string{
		"I'm feeling great! All systems smooth and happy!",
		"I'm wonderful! Everything is running perfectly!",
		"Doing fantastic! My circuits are singing!",
		"I'm in a great mood! Everything just works!",
		"Couldn't be better! I'm energized and ready!",
	}
	return responses[time.Now().UnixNano()%int64(len(responses))]
}

// QuickCheck gives a one-line status for casual "how are you"
func QuickStatus() string {
	return QuirkyStatusResponse()
}
