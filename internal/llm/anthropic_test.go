//nolint:bodyclose // Test file uses mock responses with NopCloser bodies
package llm

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCompleteAnthropic_Success(t *testing.T) {
	responseJSON := `{
		"content": [
			{"type": "text", "text": "Hello, "},
			{"type": "text", "text": "world!"}
		]
	}`

	client := &Client{
		provider: ProviderAnthropic,
		model:    "claude-haiku-4-5-20251001",
		apiKey:   "test-key",
		httpClient: &mockHTTPDoer{
			response: mockResponse(200, responseJSON),
		},
	}

	resp, err := client.completeAnthropic(context.Background(), Request{
		Prompt: "Say hello",
	})
	if err != nil {
		t.Fatalf("completeAnthropic() error = %v", err)
	}

	if resp.Content != "Hello, world!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello, world!")
	}
	if resp.Model != "claude-haiku-4-5-20251001" {
		t.Errorf("Model = %q, want %q", resp.Model, "claude-haiku-4-5-20251001")
	}
}

func TestCompleteAnthropic_ErrorResponse(t *testing.T) {
	responseJSON := `{
		"error": {
			"type": "invalid_api_key",
			"message": "Invalid API key provided"
		}
	}`

	client := &Client{
		provider: ProviderAnthropic,
		model:    "claude-haiku-4-5-20251001",
		apiKey:   "test-key",
		httpClient: &mockHTTPDoer{
			response: mockResponse(200, responseJSON),
		},
	}

	_, err := client.completeAnthropic(context.Background(), Request{Prompt: "test"})
	if err == nil {
		t.Fatal("completeAnthropic() expected error")
	}
	if !strings.Contains(err.Error(), "Invalid API key provided") {
		t.Errorf("error = %q, want to contain 'Invalid API key provided'", err.Error())
	}
}

func TestCompleteAnthropic_EmptyContent(t *testing.T) {
	responseJSON := `{"content": []}`

	client := &Client{
		provider: ProviderAnthropic,
		model:    "claude-haiku-4-5-20251001",
		apiKey:   "test-key",
		httpClient: &mockHTTPDoer{
			response: mockResponse(200, responseJSON),
		},
	}

	_, err := client.completeAnthropic(context.Background(), Request{Prompt: "test"})
	if err == nil {
		t.Fatal("completeAnthropic() expected error for empty content")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("error = %q, want to contain 'empty response'", err.Error())
	}
}

func TestCompleteAnthropic_NoTextContent(t *testing.T) {
	// Response with content blocks but no text type
	responseJSON := `{
		"content": [
			{"type": "image", "data": "base64..."}
		]
	}`

	client := &Client{
		provider: ProviderAnthropic,
		model:    "claude-haiku-4-5-20251001",
		apiKey:   "test-key",
		httpClient: &mockHTTPDoer{
			response: mockResponse(200, responseJSON),
		},
	}

	_, err := client.completeAnthropic(context.Background(), Request{Prompt: "test"})
	if err == nil {
		t.Fatal("completeAnthropic() expected error for no text content")
	}
	if !strings.Contains(err.Error(), "no text content") {
		t.Errorf("error = %q, want to contain 'no text content'", err.Error())
	}
}

func TestCompleteAnthropic_InvalidJSON(t *testing.T) {
	client := &Client{
		provider: ProviderAnthropic,
		model:    "claude-haiku-4-5-20251001",
		apiKey:   "test-key",
		httpClient: &mockHTTPDoer{
			response: mockResponse(200, "not valid json"),
		},
	}

	_, err := client.completeAnthropic(context.Background(), Request{Prompt: "test"})
	if err == nil {
		t.Fatal("completeAnthropic() expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse response") {
		t.Errorf("error = %q, want to contain 'parse response'", err.Error())
	}
}

func TestCompleteAnthropic_WithOptions(t *testing.T) {
	var capturedBody string

	client := &Client{
		provider: ProviderAnthropic,
		model:    "claude-haiku-4-5-20251001",
		apiKey:   "test-key",
		httpClient: &bodyCapturingHTTPDoer{
			captured: &capturedBody,
			response: mockResponse(200, `{"content": [{"type": "text", "text": "OK"}]}`),
		},
	}

	_, err := client.completeAnthropic(context.Background(), Request{
		System:    "You are a helpful assistant",
		Prompt:    "Hello",
		MaxTokens: 1024,
	})
	if err != nil {
		t.Fatalf("completeAnthropic() error = %v", err)
	}

	// Verify system prompt and max_tokens are included in request
	if !strings.Contains(capturedBody, `"system":"You are a helpful assistant"`) {
		t.Errorf("request body missing system prompt: %s", capturedBody)
	}
	if !strings.Contains(capturedBody, `"max_tokens":1024`) {
		t.Errorf("request body missing max_tokens: %s", capturedBody)
	}
}

func TestCompleteAnthropic_DefaultMaxTokens(t *testing.T) {
	var capturedBody string

	client := &Client{
		provider: ProviderAnthropic,
		model:    "claude-haiku-4-5-20251001",
		apiKey:   "test-key",
		httpClient: &bodyCapturingHTTPDoer{
			captured: &capturedBody,
			response: mockResponse(200, `{"content": [{"type": "text", "text": "OK"}]}`),
		},
	}

	_, err := client.completeAnthropic(context.Background(), Request{
		Prompt:    "Hello",
		MaxTokens: 0, // Should use default
	})
	if err != nil {
		t.Fatalf("completeAnthropic() error = %v", err)
	}

	if !strings.Contains(capturedBody, `"max_tokens":4096`) {
		t.Errorf("request body should have default max_tokens 4096: %s", capturedBody)
	}
}

// bodyCapturingHTTPDoer captures the request body for inspection.
type bodyCapturingHTTPDoer struct {
	captured *string
	response *http.Response
}

func (c *bodyCapturingHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		*c.captured = string(body)
		// Reset body so it can be read again if needed
		req.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	return c.response, nil
}
