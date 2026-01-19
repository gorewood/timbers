package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// OpenAI API types.
type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) completeOpenAI(ctx context.Context, req Request) (*Response, error) {
	messages := []openaiMessage{}
	if req.System != "" {
		messages = append(messages, openaiMessage{Role: "system", Content: req.System})
	}
	messages = append(messages, openaiMessage{Role: "user", Content: req.Prompt})

	body := openaiRequest{Model: c.model, Messages: messages}
	if req.MaxTokens > 0 {
		body.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		body.Temperature = req.Temperature
	}

	respBody, err := c.doRequest(ctx, "https://api.openai.com/v1/chat/completions", body, map[string]string{
		"Authorization": "Bearer " + c.apiKey,
	})
	if err != nil {
		return nil, err
	}

	var result openaiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return nil, errors.New("empty response from API")
	}

	return &Response{Content: result.Choices[0].Message.Content, Model: c.model}, nil
}
