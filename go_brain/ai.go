/**
 * THE-PATHFINDER-EYE : AI Cortex Module (v5.9 - OpenAI Protocol Compliant)
 */

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type AIResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type AIBrain struct {
	ModelPath string
	Client    *http.Client
	Tools     *ToolRegistry
}

func newAIBrain(modelPath string) *AIBrain {
	return &AIBrain{
		ModelPath: modelPath,
		Client:    &http.Client{Timeout: 45 * time.Second},
		Tools:     initHardwareTools(),
	}
}

func (a *AIBrain) EnsureLocalLLM(state bool) {
	if state {
		_ = exec.Command("sudo", "systemctl", "start", "leafcutter").Run()
	} else {
		_ = exec.Command("sudo", "systemctl", "stop", "leafcutter").Run()
	}
}

func (a *AIBrain) Process(userInput string, worldContext string) (string, error) {
	if os.Getenv("CLOUD_API_ENABLED") == "true" {
		url := os.Getenv("CLOUD_API_URL")
		apiKey := os.Getenv("CLOUD_API_KEY")

		lowInput := strings.ToLower(userInput)
		visionKeywords := []string{"see", "look at", "describe", "what is this", "scan"}
		needsVision := false
		for _, kw := range visionKeywords {
			if strings.Contains(lowInput, kw) {
				needsVision = true
				break
			}
		}

		model := os.Getenv("CLOUD_MODEL_NAME")
		if needsVision {
			model = os.Getenv("CLOUD_VISION_MODEL")
		}

		infoLog.Printf("AI_ROUTER: Attempting Cloud (%s)...", model)

		var res string
		var err error
		if needsVision {
			res, err = a.ProcessVision(userInput, worldContext, url, apiKey, model)
		} else {
			res, err = a.processText(userInput, worldContext, url, apiKey, model)
		}

		if err == nil {
			go a.EnsureLocalLLM(false)
			return res, nil
		}
		errorLog.Printf("AI_ROUTER: Cloud failed: %v", err)
		// Above `err` may carry HTTP body or auth header echo. The redact
		// pass would catch a sk-… bearer, but log lines are unwrapped so
		// we re-redact here before it lands in the journal.
		safeLogf("ERROR", "AI_ROUTER: Cloud failed: %s", redactOnce(err.Error()))
	}

	infoLog.Println("AI_ROUTER: Falling back to local intelligence...")
	a.EnsureLocalLLM(true)
	return a.processText(userInput, worldContext, "http://localhost:8081/v1/chat/completions", "", "hermes")
}

func (a *AIBrain) processText(userInput, worldContext, url, apiKey, model string) (string, error) {
	systemPrompt := `You are THE-PATHFINDER-EYE, an autonomous robot.
Use tools to interact. WORLD STATE: ` + worldContext

	messages := []map[string]interface{}{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": userInput},
	}
	return a.iterateAgent(messages, url, apiKey, model)
}

func (a *AIBrain) ProcessVision(userInput, worldContext, url, apiKey, model string) (string, error) {
	imgPath := "/tmp/vision_feed.jpg"
	imgData, err := ioutil.ReadFile(imgPath)
	if err != nil {
		return "", fmt.Errorf("no vision feed")
	}
	base64Img := base64.StdEncoding.EncodeToString(imgData)

	systemPrompt := `Analyze image and act. WORLD STATE: ` + worldContext

	messages := []map[string]interface{}{
		{"role": "system", "content": systemPrompt},
		{
			"role": "user",
			"content": []map[string]interface{}{
				{"type": "text", "text": userInput},
				{"type": "image_url", "image_url": map[string]string{"url": "data:image/jpeg;base64," + base64Img}},
			},
		},
	}
	return a.iterateAgent(messages, url, apiKey, model)
}

func (a *AIBrain) iterateAgent(messages []map[string]interface{}, url, apiKey, model string) (string, error) {
	rawTools := a.Tools.Schemas()
	formattedTools := make([]map[string]interface{}, len(rawTools))
	for i, t := range rawTools {
		formattedTools[i] = map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		}
	}

	finalSpeech := ""
	prevSpoken := ""
	for iter := 0; iter < 3; iter++ {
		payload := map[string]interface{}{
			"model":       model,
			"messages":    messages,
			"tools":       formattedTools,
			"temperature": 0.1,
		}

		jsonData, _ := json.Marshal(payload)
		prevSpoken = finalSpeech

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		resp, err := a.Client.Do(req)
		if err != nil {
			return "", fmt.Errorf("connection failed")
		}

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			// Capture the response body for the log so an HTTP failure
			// hints at the upstream reason. Redact first — the body
			// may contain echoes of the API key or bearer header.
			safeLogf("ERROR", "AI_ROUTER: API %d: %s",
				resp.StatusCode, redactOnce(string(body)))
			return "", fmt.Errorf("API %d: %s", resp.StatusCode, string(body))
		}

		var chatResponse struct {
			Choices []struct {
				Message struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(body, &chatResponse); err != nil {
			return "", fmt.Errorf("json error")
		}
		if len(chatResponse.Choices) == 0 {
			break
		}

		aiMsg := chatResponse.Choices[0].Message
		cleaned := a.cleanAIGarbage(aiMsg.Content)
		// Track the latest *cleaned* assistant content. We don't
		// concatenate across iterations because the brain and the
		// TTS pipeline want the final answer, not the entire
		// multi-turn trace.
		if cleaned != "" {
			finalSpeech = cleaned
			_ = speak(cleaned)
		}

		if len(aiMsg.ToolCalls) == 0 {
			break
		}

		// CRITICAL: Append the assistant message exactly as received
		messages = append(messages, map[string]interface{}{
			"role":       "assistant",
			"content":    aiMsg.Content,
			"tool_calls": aiMsg.ToolCalls,
		})

		// Execute tools and append 'tool' role results
		for _, tc := range aiMsg.ToolCalls {
			safeLogf("", "CORTEX_TOOL: executing %s", tc.Function.Name)
			result, err := a.Tools.Execute(tc.Function.Name, json.RawMessage(tc.Function.Arguments))
			if err != nil {
				result = "Error: " + err.Error()
			}
			messages = append(messages, map[string]interface{}{
				"role":         "tool",
				"tool_call_id": tc.ID,
				"content":      redactOnce(result),
			})
		}
	}
	// If we accumulated no final speech but did say something earlier
	// (e.g. an interim reply while a tool loop rounds out), keep the
	// most recent assistant content so the TTS doesn't get a blank reply.
	if finalSpeech == "" {
		finalSpeech = prevSpoken
	}
	return strings.TrimSpace(finalSpeech), nil
}

func (a *AIBrain) cleanAIGarbage(raw string) string {
	lines := strings.Split(raw, "\n")
	var goodLines []string
	for _, line := range lines {
		tLine := strings.TrimSpace(line)
		if tLine == "" || strings.HasPrefix(tLine, "/") || strings.HasPrefix(tLine, "```") {
			continue
		}
		if strings.Contains(strings.ToLower(tLine), "i can assist you") {
			continue
		}
		goodLines = append(goodLines, tLine)
	}
	return strings.Join(goodLines, " ")
}
