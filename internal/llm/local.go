package llm

import (
	"context"
	"encoding/json"

	"github.com/gorewood/timbers/internal/output"
)

// Local LLM server API types (OpenAI-compatible format).
// Works with LM Studio, Ollama, and other OpenAI-compatible servers.

type localRequest struct {
	Model       string         `json:"model"`
	Messages    []localMessage `json:"messages"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Temperature float64        `json:"temperature,omitempty"`
}

type localMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type localResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) completeLocal(ctx context.Context, req Request) (*Response, error) {
	body := c.buildLocalRequest(req)
	url := LocalServerURL() + "/chat/completions"

	respBody, err := c.doRequest(ctx, url, body, nil)
	if err != nil {
		return nil, err
	}

	return parseLocalResponse(respBody, c.model)
}

func (c *Client) buildLocalRequest(req Request) localRequest {
	messages := []localMessage{}
	if req.System != "" {
		messages = append(messages, localMessage{Role: "system", Content: req.System})
	}
	messages = append(messages, localMessage{Role: "user", Content: req.Prompt})

	// Use empty string to let the server use its loaded model
	model := c.model
	if model == "default" || model == "local" {
		model = ""
	}

	body := localRequest{Model: model, Messages: messages}
	if req.MaxTokens > 0 {
		body.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		body.Temperature = req.Temperature
	}
	return body
}

func parseLocalResponse(respBody []byte, model string) (*Response, error) {
	var result localResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, output.NewSystemErrorWithCause("failed to parse response", err)
	}

	if result.Error != nil {
		return nil, output.NewSystemError("API error: " + result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return nil, output.NewSystemError("empty response from API")
	}

	responseModel := model
	if responseModel == "" || responseModel == "default" {
		responseModel = "local"
	}

	return &Response{Content: result.Choices[0].Message.Content, Model: responseModel}, nil
}
