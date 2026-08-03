package llm

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/gorewood/timbers/internal/output"
)

// defaultOpenAIBaseURL is the public OpenAI API root. Users can override the
// base URL via the OPENAI_BASE_URL environment variable to target
// OpenAI-compatible gateways such as AWS Bedrock Mantle, Azure OpenAI,
// OpenRouter, LiteLLM, or vLLM. This matches the convention used by the
// official OpenAI SDKs.
const defaultOpenAIBaseURL = "https://api.openai.com/v1"

// openAIChatCompletionsURL returns the chat-completions endpoint, honoring
// OPENAI_BASE_URL when set. The env var is expected to point at the API root
// (e.g. "https://example.com/v1"); the "/chat/completions" path is appended
// here. Trailing slashes on the base URL are tolerated.
func openAIChatCompletionsURL() string {
	base := os.Getenv("OPENAI_BASE_URL")
	if base == "" {
		base = defaultOpenAIBaseURL
	}
	return strings.TrimRight(base, "/") + "/chat/completions"
}

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

	respBody, err := c.doRequest(ctx, openAIChatCompletionsURL(), body, map[string]string{
		"Authorization": "Bearer " + c.apiKey,
	})
	if err != nil {
		return nil, err
	}

	var result openaiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, output.NewSystemErrorWithCause("failed to parse response", err)
	}

	if result.Error != nil {
		return nil, output.NewSystemError("API error: " + result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return nil, output.NewSystemError("empty response from API")
	}

	return &Response{Content: result.Choices[0].Message.Content, Model: c.model}, nil
}
