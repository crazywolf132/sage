package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/crazywolf132/sage/internal/config"
)

// OpenAILLM is an implementation of LLM using OpenAI's API
type OpenAILLM struct {
	apiKey  string
	model   string
	baseURL string
}

// NewOpenAILLM creates a new OpenAI LLM client
func NewOpenAILLM() *OpenAILLM {
	model := config.Get("ai.model", false)
	if model == "" {
		model = "gpt-4"
	}

	baseURL := config.Get("ai.base_url", false)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = config.Get("ai.api_key", false)
	}

	return &OpenAILLM{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Complete implements the LLM interface
func (o *OpenAILLM) Complete(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("%s/chat/completions", o.baseURL)

	req := chatRequest{
		Model: o.model,
		Messages: []chatMessage{
			{Role: "system", Content: "You are a helpful AI assistant."},
			{Role: "user", Content: prompt},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.apiKey))

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no completion choices returned")
	}

	return chatResp.Choices[0].Message.Content, nil
}
