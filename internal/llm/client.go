package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nsourov/gcai/internal/prompt"
)

const defaultTimeout = 45 * time.Second

type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

type chatRequest struct {
	Model       string           `json:"model"`
	Messages    []prompt.Message `json:"messages"`
	Temperature float64          `json:"temperature,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) GenerateCommitMessage(ctx context.Context, messages []prompt.Message) (string, error) {
	if c.BaseURL == "" {
		return "", fmt.Errorf("base URL is required")
	}
	if c.APIKey == "" {
		return "", fmt.Errorf("API key is required")
	}
	if c.Model == "" {
		return "", fmt.Errorf("model is required")
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	baseURL := strings.TrimRight(c.BaseURL, "/")
	url := baseURL + "/chat/completions"
	reqBody := chatRequest{
		Model:       c.Model,
		Messages:    messages,
		Temperature: 0.2,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed.Error != nil && parsed.Error.Message != "" {
			return "", fmt.Errorf("API error: %s", parsed.Error.Message)
		}
		return "", fmt.Errorf("API error: status %d", resp.StatusCode)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("API returned no choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("API returned empty content")
	}

	// The tool expects a single-line subject by default.
	lines := strings.Split(content, "\n")
	return strings.TrimSpace(lines[0]), nil
}
