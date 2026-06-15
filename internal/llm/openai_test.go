//nolint:bodyclose // Test file uses mock responses with NopCloser bodies
package llm

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCompleteOpenAI_Success(t *testing.T) {
	responseJSON := `{
		"choices": [
			{
				"message": {
					"content": "Hello, world!"
				}
			}
		]
	}`

	client := &Client{
		provider: ProviderOpenAI,
		model:    "gpt-5-nano",
		apiKey:   "test-key",
		httpClient: &mockHTTPDoer{
			response: mockResponse(200, responseJSON),
		},
	}

	resp, err := client.completeOpenAI(context.Background(), Request{
		Prompt: "Say hello",
	})
	if err != nil {
		t.Fatalf("completeOpenAI() error = %v", err)
	}

	if resp.Content != "Hello, world!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello, world!")
	}
	if resp.Model != "gpt-5-nano" {
		t.Errorf("Model = %q, want %q", resp.Model, "gpt-5-nano")
	}
}

func TestCompleteOpenAI_ErrorResponse(t *testing.T) {
	responseJSON := `{
		"error": {
			"message": "Invalid API key"
		}
	}`

	client := &Client{
		provider: ProviderOpenAI,
		model:    "gpt-5-nano",
		apiKey:   "test-key",
		httpClient: &mockHTTPDoer{
			response: mockResponse(200, responseJSON),
		},
	}

	_, err := client.completeOpenAI(context.Background(), Request{Prompt: "test"})
	if err == nil {
		t.Fatal("completeOpenAI() expected error")
	}
	if !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("error = %q, want to contain 'Invalid API key'", err.Error())
	}
}

func TestCompleteOpenAI_EmptyChoices(t *testing.T) {
	responseJSON := `{"choices": []}`

	client := &Client{
		provider: ProviderOpenAI,
		model:    "gpt-5-nano",
		apiKey:   "test-key",
		httpClient: &mockHTTPDoer{
			response: mockResponse(200, responseJSON),
		},
	}

	_, err := client.completeOpenAI(context.Background(), Request{Prompt: "test"})
	if err == nil {
		t.Fatal("completeOpenAI() expected error for empty choices")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("error = %q, want to contain 'empty response'", err.Error())
	}
}

func TestCompleteOpenAI_InvalidJSON(t *testing.T) {
	client := &Client{
		provider: ProviderOpenAI,
		model:    "gpt-5-nano",
		apiKey:   "test-key",
		httpClient: &mockHTTPDoer{
			response: mockResponse(200, "not valid json"),
		},
	}

	_, err := client.completeOpenAI(context.Background(), Request{Prompt: "test"})
	if err == nil {
		t.Fatal("completeOpenAI() expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse response") {
		t.Errorf("error = %q, want to contain 'parse response'", err.Error())
	}
}

func TestCompleteOpenAI_WithSystemPrompt(t *testing.T) {
	var capturedBody string

	client := &Client{
		provider: ProviderOpenAI,
		model:    "gpt-5-nano",
		apiKey:   "test-key",
		httpClient: &openaiBodyCapturingHTTPDoer{
			captured: &capturedBody,
			response: mockResponse(200, `{"choices": [{"message": {"content": "OK"}}]}`),
		},
	}

	_, err := client.completeOpenAI(context.Background(), Request{
		System: "You are a helpful assistant",
		Prompt: "Hello",
	})
	if err != nil {
		t.Fatalf("completeOpenAI() error = %v", err)
	}

	// Verify system message is included
	if !strings.Contains(capturedBody, `"role":"system"`) {
		t.Errorf("request body missing system role: %s", capturedBody)
	}
	if !strings.Contains(capturedBody, `"content":"You are a helpful assistant"`) {
		t.Errorf("request body missing system content: %s", capturedBody)
	}
}

func TestCompleteOpenAI_WithoutSystemPrompt(t *testing.T) {
	var capturedBody string

	client := &Client{
		provider: ProviderOpenAI,
		model:    "gpt-5-nano",
		apiKey:   "test-key",
		httpClient: &openaiBodyCapturingHTTPDoer{
			captured: &capturedBody,
			response: mockResponse(200, `{"choices": [{"message": {"content": "OK"}}]}`),
		},
	}

	_, err := client.completeOpenAI(context.Background(), Request{
		Prompt: "Hello",
		// No system prompt
	})
	if err != nil {
		t.Fatalf("completeOpenAI() error = %v", err)
	}

	// Verify no system message is included
	if strings.Contains(capturedBody, `"role":"system"`) {
		t.Errorf("request body should not have system role when empty: %s", capturedBody)
	}
}

func TestCompleteOpenAI_WithOptions(t *testing.T) {
	var capturedBody string

	client := &Client{
		provider: ProviderOpenAI,
		model:    "gpt-5-nano",
		apiKey:   "test-key",
		httpClient: &openaiBodyCapturingHTTPDoer{
			captured: &capturedBody,
			response: mockResponse(200, `{"choices": [{"message": {"content": "OK"}}]}`),
		},
	}

	_, err := client.completeOpenAI(context.Background(), Request{
		Prompt:      "Hello",
		MaxTokens:   2048,
		Temperature: 0.7,
	})
	if err != nil {
		t.Fatalf("completeOpenAI() error = %v", err)
	}

	// Verify max_tokens and temperature are included
	if !strings.Contains(capturedBody, `"max_tokens":2048`) {
		t.Errorf("request body missing max_tokens: %s", capturedBody)
	}
	if !strings.Contains(capturedBody, `"temperature":0.7`) {
		t.Errorf("request body missing temperature: %s", capturedBody)
	}
}

func TestCompleteOpenAI_ZeroOptionsOmitted(t *testing.T) {
	var capturedBody string

	client := &Client{
		provider: ProviderOpenAI,
		model:    "gpt-5-nano",
		apiKey:   "test-key",
		httpClient: &openaiBodyCapturingHTTPDoer{
			captured: &capturedBody,
			response: mockResponse(200, `{"choices": [{"message": {"content": "OK"}}]}`),
		},
	}

	_, err := client.completeOpenAI(context.Background(), Request{
		Prompt:      "Hello",
		MaxTokens:   0, // Should be omitted
		Temperature: 0, // Should be omitted
	})
	if err != nil {
		t.Fatalf("completeOpenAI() error = %v", err)
	}

	// Verify zero values are omitted (not included in request)
	if strings.Contains(capturedBody, `"max_tokens"`) {
		t.Errorf("request body should omit zero max_tokens: %s", capturedBody)
	}
	if strings.Contains(capturedBody, `"temperature"`) {
		t.Errorf("request body should omit zero temperature: %s", capturedBody)
	}
}

func TestOpenAIChatCompletionsURL_DefaultsToOpenAI(t *testing.T) {
	// Ensure no override leaks in from the surrounding environment.
	t.Setenv("OPENAI_BASE_URL", "")

	got := openAIChatCompletionsURL()
	want := "https://api.openai.com/v1/chat/completions"
	if got != want {
		t.Errorf("openAIChatCompletionsURL() = %q, want %q", got, want)
	}
}

func TestOpenAIChatCompletionsURL_RespectsBaseURL(t *testing.T) {
	cases := []struct {
		name string
		base string
		want string
	}{
		{
			name: "no trailing slash",
			base: "https://example.com/v1",
			want: "https://example.com/v1/chat/completions",
		},
		{
			name: "trailing slash",
			base: "https://example.com/v1/",
			want: "https://example.com/v1/chat/completions",
		},
		{
			name: "multiple trailing slashes",
			base: "https://example.com/v1///",
			want: "https://example.com/v1/chat/completions",
		},
		{
			name: "non-v1 root",
			base: "https://bedrock.example.com",
			want: "https://bedrock.example.com/chat/completions",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OPENAI_BASE_URL", tc.base)
			if got := openAIChatCompletionsURL(); got != tc.want {
				t.Errorf("openAIChatCompletionsURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestCompleteOpenAI_BaseURLOverride proves the env var actually changes where
// the request is sent: a local httptest server stands in for the gateway and
// asserts it received the chat-completions call.
func TestCompleteOpenAI_BaseURLOverride(t *testing.T) {
	var (
		hitPath string
		hitAuth string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitPath = r.URL.Path
		hitAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"routed"}}]}`))
	}))
	defer srv.Close()

	t.Setenv("OPENAI_BASE_URL", srv.URL+"/v1")

	client := &Client{
		provider:   ProviderOpenAI,
		model:      "gpt-5-nano",
		apiKey:     "test-key",
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	resp, err := client.completeOpenAI(context.Background(), Request{Prompt: "hi"})
	if err != nil {
		t.Fatalf("completeOpenAI() error = %v", err)
	}
	if resp.Content != "routed" {
		t.Errorf("Content = %q, want %q", resp.Content, "routed")
	}
	if hitPath != "/v1/chat/completions" {
		t.Errorf("server received path = %q, want %q", hitPath, "/v1/chat/completions")
	}
	if hitAuth != "Bearer test-key" {
		t.Errorf("server received Authorization = %q, want %q", hitAuth, "Bearer test-key")
	}
}

// openaiBodyCapturingHTTPDoer captures the request body for inspection.
type openaiBodyCapturingHTTPDoer struct {
	captured *string
	response *http.Response
}

func (c *openaiBodyCapturingHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		*c.captured = string(body)
		req.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	return c.response, nil
}
