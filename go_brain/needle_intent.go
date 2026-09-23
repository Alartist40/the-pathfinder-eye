package main

// needle_intent.go — Needle 2 integration for pathfinder-eye.
//
// Replaces the legacy TRM classifier and unifies tool-calling and intent classification.
// Calls out to the local Needle 2 service on localhost:8082/complete.
// Needle 2 (45M params, ~28MB RAM session) returns structured function calls with confidence.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	needleDefaultURL        = "http://127.0.0.1:8082/complete"
	needleConfidenceFloor   = 0.5
	needleMaxConsecutiveErr = 3
	needleRequestTimeout    = 3 * time.Second
)

// needleAvailability tracks how the Needle 2 service is behaving.
// When the service fails 3 times in a row, we mark it down to avoid added latency.
type needleAvailability struct {
	consecutiveErrors int32
	markedDown        int32
	mu                sync.Mutex
	lastErr           error
}

func (a *needleAvailability) recordSuccess() {
	atomic.StoreInt32(&a.consecutiveErrors, 0)
	atomic.StoreInt32(&a.markedDown, 0)
	a.mu.Lock()
	a.lastErr = nil
	a.mu.Unlock()
}

func (a *needleAvailability) recordFailure(err error) {
	a.mu.Lock()
	a.lastErr = err
	a.mu.Unlock()
	count := atomic.AddInt32(&a.consecutiveErrors, 1)
	if count >= needleMaxConsecutiveErr {
		atomic.StoreInt32(&a.markedDown, 1)
	}
}

func (a *needleAvailability) isMarkedDown() bool {
	return atomic.LoadInt32(&a.markedDown) == 1
}

var needleHealth = &needleAvailability{}

// NeedleRequest is the JSON payload posted to /complete.
type NeedleRequest struct {
	Input string `json:"input"`
}

// NeedleResponse is the JSON returned by Needle 2 /complete.
type NeedleResponse struct {
	Type          string `json:"type"`
	Success       bool   `json:"success"`
	FunctionCalls []struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function_calls"`
	Reasoning  string  `json:"reasoning,omitempty"`
	Confidence float32 `json:"confidence"`
}

var (
	cachedNeedleSchemas []Tool
	needleSchemaOnce    sync.Once
)

func getCachedToolSchemas() []Tool {
	needleSchemaOnce.Do(func() {
		cachedNeedleSchemas = loadToolSchemas()
	})
	return cachedNeedleSchemas
}

func loadToolSchemas() []Tool {
	path := "../config/needle_tools.json"
	if _, err := os.Stat(path); err != nil {
		path = "config/needle_tools.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var rawTools []struct {
		Function struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			Parameters  map[string]any `json:"parameters"`
		} `json:"function"`
	}
	_ = json.Unmarshal(data, &rawTools)
	var tools []Tool
	for _, rt := range rawTools {
		tools = append(tools, Tool{
			Name:        rt.Function.Name,
			Description: rt.Function.Description,
			Parameters:  rt.Function.Parameters,
		})
	}
	return tools
}

// processCommandNeedle attempts to classify and execute a transcribed command
// using Needle 2.
func processCommandNeedle(cmd string) (handled bool, result string) {
	return processCommandNeedleWithAuth(cmd, LevelGuest, "Guest")
}

// processCommandNeedleWithAuth attempts to classify and execute a transcribed command
// using Needle 2 with authority verification.
func processCommandNeedleWithAuth(cmd string, level AuthorityLevel, name string) (handled bool, result string) {
	if needleHealth.isMarkedDown() {
		return false, ""
	}

	reqPayload := NeedleRequest{Input: cmd}
	body, err := json.Marshal(reqPayload)
	if err != nil {
		needleHealth.recordFailure(err)
		return false, ""
	}

	client := &http.Client{Timeout: needleRequestTimeout}
	resp, err := client.Post(needleDefaultURL, "application/json", bytes.NewReader(body))
	if err != nil {
		needleHealth.recordFailure(err)
		return false, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable {
		return false, ""
	}

	if resp.StatusCode != http.StatusOK {
		needleHealth.recordFailure(errors.New("needle non-200"))
		return false, ""
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		needleHealth.recordFailure(err)
		return false, ""
	}

	var nResp NeedleResponse
	if err := json.Unmarshal(data, &nResp); err != nil {
		needleHealth.recordFailure(err)
		return false, ""
	}

	if !nResp.Success || len(nResp.FunctionCalls) == 0 {
		return false, ""
	}

	if nResp.Confidence < needleConfidenceFloor {
		return false, ""
	}

	needleHealth.recordSuccess()

	var toolResults []string
	for _, fc := range nResp.FunctionCalls {
		switch fc.Name {
		case "activate", "deactivate":
			var arg struct {
				Mode string `json:"mode"`
			}
			_ = json.Unmarshal(fc.Arguments, &arg)
			if arg.Mode != "" {
				parsed := ParsedCommand{
					Action:    fc.Name,
					Target:    arg.Mode,
					Modifiers: make(map[string]string),
				}
				if dispatchAction(parsed, level, name) {
					toolResults = append(toolResults, fmt.Sprintf("%s %s", fc.Name, arg.Mode))
				}
			}
		default:
			// Hardware tool execution (move, look, light, play_resource, read_document)
			safeLogf("", "NEEDLE2_EXECUTE: tool=%s args=%s", fc.Name, string(fc.Arguments))
			res, execErr := initHardwareTools().Execute(fc.Name, fc.Arguments)
			if execErr != nil {
				toolResults = append(toolResults, fmt.Sprintf("%s error: %v", fc.Name, execErr))
			} else if res != "" {
				toolResults = append(toolResults, res)
			}
		}
	}

	if len(toolResults) == 0 {
		return false, ""
	}

	finalMsg := strings.Join(toolResults, ". ")
	_ = speak(finalMsg)
	return true, finalMsg
}

// MarkNeedleServiceUp resets the availability counter when the service recovers.
func MarkNeedleServiceUp() {
	needleHealth.recordSuccess()
}
