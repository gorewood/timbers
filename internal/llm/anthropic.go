package llm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gorewood/timbers/internal/output"
)

// Anthropic API types.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) completeAnthropic(ctx context.Context, req Request) (*Response, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	body := anthropicRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    req.System,
		Messages:  []anthropicMessage{{Role: "user", Content: req.Prompt}},
	}

	respBody, err := c.doRequest(ctx, "https://api.anthropic.com/v1/messages", body, map[string]string{
		"x-api-key":         c.apiKey,
		"anthropic-version": "2023-06-01",
	})
	if err != nil {
		return nil, err
	}

	var result anthropicResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, output.NewSystemErrorWithCause("failed to parse response", err)
	}

	if result.Error != nil {
		return nil, output.NewSystemError("API error: " + result.Error.Message)
	}

	if len(result.Content) == 0 {
		return nil, output.NewSystemError("empty response from API")
	}

	var content strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			content.WriteString(block.Text)
		}
	}

	if content.Len() == 0 {
		return nil, output.NewSystemError("response contained no text content")
	}

	return &Response{Content: content.String(), Model: c.model}, nil
}
