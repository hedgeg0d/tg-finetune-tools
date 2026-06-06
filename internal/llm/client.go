package llm

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

type Client struct {
	base  string
	style string
	key   string
	model string
	temp  float64
	http  *http.Client
}

func New(g config.Generated) *Client {
	style := g.APIStyle
	if style == "" {
		style = "openai"
	}
	return &Client{
		base:  strings.TrimRight(g.APIBase, "/"),
		style: style,
		key:   os.Getenv(g.APIKeyEnv),
		model: g.Model,
		temp:  g.Temperature,
		http:  &http.Client{Timeout: 180 * time.Second},
	}
}

func (c *Client) Generate(instruction, content string) (string, error) {
	if c.style == "ollama" {
		return c.generateOllama(instruction, content)
	}
	return c.generateOpenAI(instruction, content)
}

func (c *Client) Embed(text string) ([]float64, error) {
	if c.style == "ollama" {
		return c.embedOllama(text)
	}
	return c.embedOpenAI(text)
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

func (c *Client) generateOpenAI(instruction, content string) (string, error) {
	payload, _ := json.Marshal(chatRequest{
		Model:       c.model,
		Temperature: c.temp,
		Messages: []chatMessage{
			{Role: "system", Content: instruction},
			{Role: "user", Content: content},
		},
	})

	body, err := c.post(c.base+"/chat/completions", payload)
	if err != nil {
		return "", err
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

type ollamaChatRequest struct {
	Model    string         `json:"model"`
	Stream   bool           `json:"stream"`
	Think    bool           `json:"think"`
	Options  map[string]any `json:"options"`
	Messages []chatMessage  `json:"messages"`
}

type ollamaChatResponse struct {
	Message chatMessage `json:"message"`
	Error   string      `json:"error"`
}

func (c *Client) generateOllama(instruction, content string) (string, error) {
	payload, _ := json.Marshal(ollamaChatRequest{
		Model:   c.model,
		Stream:  false,
		Think:   false,
		Options: map[string]any{"temperature": c.temp},
		Messages: []chatMessage{
			{Role: "system", Content: instruction},
			{Role: "user", Content: content},
		},
	})

	body, err := c.post(c.base+"/api/chat", payload)
	if err != nil {
		return "", err
	}

	var parsed ollamaChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.Error != "" {
		return "", fmt.Errorf("ollama error: %s", parsed.Error)
	}
	return parsed.Message.Content, nil
}

type openAIEmbedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) embedOpenAI(text string) ([]float64, error) {
	payload, _ := json.Marshal(map[string]any{"model": c.model, "input": text})
	body, err := c.post(c.base+"/embeddings", payload)
	if err != nil {
		return nil, err
	}
	var parsed openAIEmbedResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("api error: %s", parsed.Error.Message)
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("api returned no embedding")
	}
	return parsed.Data[0].Embedding, nil
}

type ollamaEmbedResponse struct {
	Embedding []float64 `json:"embedding"`
	Error     string    `json:"error"`
}

func (c *Client) embedOllama(text string) ([]float64, error) {
	payload, _ := json.Marshal(map[string]any{"model": c.model, "prompt": text})
	body, err := c.post(c.base+"/api/embeddings", payload)
	if err != nil {
		return nil, err
	}
	var parsed ollamaEmbedResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.Error != "" {
		return nil, fmt.Errorf("ollama error: %s", parsed.Error)
	}
	if len(parsed.Embedding) == 0 {
		return nil, fmt.Errorf("ollama returned no embedding")
	}
	return parsed.Embedding, nil
}

func (c *Client) post(url string, payload []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}
