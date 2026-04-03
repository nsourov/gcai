package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	Error json.RawMessage `json:"error,omitempty"`
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
	applyOpenRouterHeaders(req, baseURL)

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
		snippet := strings.TrimSpace(string(respBytes))
		if len(snippet) > 400 {
			snippet = snippet[:400] + "…"
		}
		return "", fmt.Errorf("decode response: %w (body: %s)", err, snippet)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if msg := formatAPIError(parsed.Error, respBytes); msg != "" {
			return "", fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, msg)
		}
		snippet := strings.TrimSpace(string(respBytes))
		if len(snippet) > 500 {
			snippet = snippet[:500] + "…"
		}
		if snippet != "" {
			return "", fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, snippet)
		}
		return "", fmt.Errorf("API error: HTTP %d", resp.StatusCode)
	}
	if len(parsed.Choices) == 0 {
		if msg := formatAPIError(parsed.Error, respBytes); msg != "" {
			return "", fmt.Errorf("API returned no choices: %s", msg)
		}
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

// formatAPIError extracts a human-readable message from common OpenAI-compatible error shapes,
// including OpenRouter-style nested metadata.
func formatAPIError(errField json.RawMessage, fullBody []byte) string {
	if len(errField) == 0 {
		return ""
	}
	var asString string
	if err := json.Unmarshal(errField, &asString); err == nil && strings.TrimSpace(asString) != "" {
		return strings.TrimSpace(asString)
	}
	var nested struct {
		Message  string          `json:"message"`
		Code     json.RawMessage `json:"code"`
		Type     string          `json:"type"`
		Param    string          `json:"param"`
		Metadata json.RawMessage `json:"metadata"`
	}
	if err := json.Unmarshal(errField, &nested); err != nil {
		s := strings.TrimSpace(string(errField))
		if len(s) > 400 {
			return s[:400] + "…"
		}
		return s
	}
	var parts []string
	if nested.Message != "" {
		parts = append(parts, nested.Message)
	}
	if nested.Type != "" {
		parts = append(parts, "type="+nested.Type)
	}
	if nested.Param != "" {
		parts = append(parts, "param="+nested.Param)
	}
	if len(nested.Code) > 0 && string(nested.Code) != "null" {
		parts = append(parts, "code="+strings.TrimSpace(string(nested.Code)))
	}
	if len(nested.Metadata) > 0 && string(nested.Metadata) != "null" {
		var meta struct {
			ProviderName string `json:"provider_name"`
			Raw          string `json:"raw"`
			HTTPStatus   int    `json:"http_status"`
		}
		if err := json.Unmarshal(nested.Metadata, &meta); err == nil {
			if meta.ProviderName != "" {
				parts = append(parts, "provider="+meta.ProviderName)
			}
			if meta.Raw != "" {
				raw := strings.TrimSpace(meta.Raw)
				if len(raw) > 350 {
					raw = raw[:350] + "…"
				}
				parts = append(parts, "detail="+raw)
			}
			if meta.HTTPStatus != 0 {
				parts = append(parts, fmt.Sprintf("upstream_HTTP=%d", meta.HTTPStatus))
			}
		} else {
			m := strings.TrimSpace(string(nested.Metadata))
			if len(m) > 200 {
				m = m[:200] + "…"
			}
			if m != "" {
				parts = append(parts, "metadata="+m)
			}
		}
	}
	out := strings.Join(parts, "; ")
	if out == "" {
		s := strings.TrimSpace(string(fullBody))
		if len(s) > 400 {
			s = s[:400] + "…"
		}
		return s
	}
	return out
}

func applyOpenRouterHeaders(req *http.Request, baseURL string) {
	host := ""
	if u, err := url.Parse(baseURL); err == nil && u.Host != "" {
		host = strings.ToLower(u.Host)
	} else {
		host = strings.ToLower(baseURL)
	}
	if !strings.Contains(host, "openrouter.ai") {
		return
	}
	// OpenRouter recommends these for attribution; some setups behave better with them set.
	if req.Header.Get("Referer") == "" {
		req.Header.Set("Referer", "https://github.com/nsourov/gcai")
	}
	if req.Header.Get("X-OpenRouter-Title") == "" {
		req.Header.Set("X-OpenRouter-Title", "gcai")
	}
}
