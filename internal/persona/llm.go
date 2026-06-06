package persona

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
)

type client struct {
	base  string
	style string
	key   string
	model string
	temp  float64
	http  *http.Client
}

func newClient(g config.Generated) *client {
	style := g.APIStyle
	if style == "" {
		style = "openai"
	}
	return &client{
		base:  strings.TrimRight(g.APIBase, "/"),
		style: style,
		key:   os.Getenv(g.APIKeyEnv),
		model: g.Model,
		temp:  g.Temperature,
		http:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *client) generate(instruction, context string) (string, error) {
	if c.style == "ollama" {
		return c.generateOllama(instruction, context)
	}
	return c.generateOpenAI(instruction, context)
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Temperature float64       `json:"temperature"`
	Messages    []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *client) generateOpenAI(instruction, context string) (string, error) {
	payload, err := json.Marshal(chatRequest{
		Model:       c.model,
		Temperature: c.temp,
		Messages: []chatMessage{
			{Role: "system", Content: instruction},
			{Role: "user", Content: context},
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, c.base+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("api error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("api returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

type ollamaRequest struct {
	Model    string         `json:"model"`
	Stream   bool           `json:"stream"`
	Think    bool           `json:"think"`
	Options  map[string]any `json:"options"`
	Messages []chatMessage  `json:"messages"`
}

type ollamaResponse struct {
	Message chatMessage `json:"message"`
	Error   string      `json:"error"`
}

func (c *client) generateOllama(instruction, context string) (string, error) {
	payload, err := json.Marshal(ollamaRequest{
		Model:   c.model,
		Stream:  false,
		Think:   false,
		Options: map[string]any{"temperature": c.temp},
		Messages: []chatMessage{
			{Role: "system", Content: instruction},
			{Role: "user", Content: context},
		},
	})
	if err != nil {
		return "", err
	}

	resp, err := c.http.Post(c.base+"/api/chat", "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed ollamaResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.Error != "" {
		return "", fmt.Errorf("ollama error: %s", parsed.Error)
	}
	return parsed.Message.Content, nil
}
