//nolint:bodyclose // Test file uses mock responses with NopCloser bodies
package llm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
)

// mockHTTPDoer implements HTTPDoer for testing.
type mockHTTPDoer struct {
	response *http.Response
	err      error
}

func (m *mockHTTPDoer) Do(*http.Request) (*http.Response, error) {
	return m.response, m.err
}

// mockResponse creates a mock HTTP response with the given status and body.
// The body uses io.NopCloser so no explicit close is required.
func mockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

func TestParseProviderPrefix(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		wantProvider Provider
		wantModel    string
	}{
		{
			name:         "claude prefix",
			model:        "claude-haiku",
			wantProvider: ProviderAnthropic,
			wantModel:    "haiku",
		},
		{
			name:         "anthropic prefix",
			model:        "anthropic-sonnet",
			wantProvider: ProviderAnthropic,
			wantModel:    "sonnet",
		},
		{
			name:         "gemini prefix",
			model:        "gemini-flash",
			wantProvider: ProviderGoogle,
			wantModel:    "flash",
		},
		{
			name:         "google prefix",
			model:        "google-pro",
			wantProvider: ProviderGoogle,
			wantModel:    "pro",
		},
		{
			name:         "openai prefix",
			model:        "openai-gpt-5",
			wantProvider: ProviderOpenAI,
			wantModel:    "gpt-5",
		},
		{
			name:         "local prefix",
			model:        "local-llama",
			wantProvider: ProviderLocal,
			wantModel:    "llama",
		},
		{
			name:         "no prefix - full model name",
			model:        "claude-3-haiku-20240307",
			wantProvider: ProviderAnthropic,
			wantModel:    "3-haiku-20240307",
		},
		{
			name:         "no matching prefix",
			model:        "gpt-4-turbo",
			wantProvider: "",
			wantModel:    "gpt-4-turbo",
		},
		{
			name:         "case insensitive - uppercase",
			model:        "CLAUDE-haiku",
			wantProvider: ProviderAnthropic,
			wantModel:    "haiku",
		},
		{
			name:         "case insensitive - mixed case",
			model:        "Gemini-Flash",
			wantProvider: ProviderGoogle,
			wantModel:    "Flash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, model := parseProviderPrefix(tt.model)
			if provider != tt.wantProvider {
				t.Errorf("parseProviderPrefix(%q) provider = %q, want %q", tt.model, provider, tt.wantProvider)
			}
			if model != tt.wantModel {
				t.Errorf("parseProviderPrefix(%q) model = %q, want %q", tt.model, model, tt.wantModel)
			}
		})
	}
}

func TestInferProvider(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		wantProvider Provider
	}{
		// Anthropic patterns
		{name: "claude model", model: "claude-3-opus", wantProvider: ProviderAnthropic},
		{name: "haiku model", model: "haiku-latest", wantProvider: ProviderAnthropic},
		{name: "sonnet model", model: "sonnet-3.5", wantProvider: ProviderAnthropic},
		{name: "opus model", model: "opus-4", wantProvider: ProviderAnthropic},

		// OpenAI patterns
		{name: "gpt model", model: "gpt-4-turbo", wantProvider: ProviderOpenAI},
		{name: "nano model", model: "nano-v2", wantProvider: ProviderOpenAI},
		{name: "o1 model", model: "o1-preview", wantProvider: ProviderOpenAI},
		{name: "o3 model", model: "o3-mini", wantProvider: ProviderOpenAI},
		{name: "o4 model", model: "o4-latest", wantProvider: ProviderOpenAI},

		// Google patterns
		{name: "gemini model", model: "gemini-pro", wantProvider: ProviderGoogle},
		{name: "flash model", model: "flash-lite", wantProvider: ProviderGoogle},

		// Local patterns
		{name: "local model", model: "local-default", wantProvider: ProviderLocal},
		{name: "qwen model", model: "qwen-72b", wantProvider: ProviderLocal},
		{name: "llama model", model: "llama-3-8b", wantProvider: ProviderLocal},
		{name: "mistral model", model: "mistral-7b", wantProvider: ProviderLocal},
		{name: "phi model", model: "phi-3", wantProvider: ProviderLocal},

		// Case insensitive
		{name: "uppercase", model: "GPT-4", wantProvider: ProviderOpenAI},
		{name: "mixed case", model: "Claude-Opus", wantProvider: ProviderAnthropic},

		// Default to Anthropic for unknown
		{name: "unknown model", model: "unknown-model-xyz", wantProvider: ProviderAnthropic},
		{name: "empty string", model: "", wantProvider: ProviderAnthropic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferProvider(tt.model)
			if got != tt.wantProvider {
				t.Errorf("inferProvider(%q) = %q, want %q", tt.model, got, tt.wantProvider)
			}
		})
	}
}

func TestResolveModelAlias(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		provider  Provider
		wantModel string
	}{
		// Anthropic aliases
		{name: "anthropic haiku alias", model: "haiku", provider: ProviderAnthropic, wantModel: "claude-haiku-4-5-20251001"},
		{name: "anthropic sonnet alias", model: "sonnet", provider: ProviderAnthropic, wantModel: "claude-sonnet-4-5-20250929"},
		{name: "anthropic opus alias", model: "opus", provider: ProviderAnthropic, wantModel: "claude-opus-4-6"},

		// OpenAI aliases
		{name: "openai nano alias", model: "nano", provider: ProviderOpenAI, wantModel: "gpt-5-nano"},
		{name: "openai mini alias", model: "mini", provider: ProviderOpenAI, wantModel: "gpt-5-mini"},
		{name: "openai gpt alias", model: "gpt", provider: ProviderOpenAI, wantModel: "gpt-5.2"},

		// Google aliases
		{name: "google flash alias", model: "flash", provider: ProviderGoogle, wantModel: "gemini-3-flash-preview"},
		{name: "google flash-lite alias", model: "flash-lite", provider: ProviderGoogle, wantModel: "gemini-2.5-flash-lite"},
		{name: "google pro alias", model: "pro", provider: ProviderGoogle, wantModel: "gemini-3-pro-preview"},

		// Local aliases
		{name: "local alias", model: "local", provider: ProviderLocal, wantModel: "default"},

		// Case insensitive lookup
		{name: "uppercase alias", model: "HAIKU", provider: ProviderAnthropic, wantModel: "claude-haiku-4-5-20251001"},
		{name: "mixed case alias", model: "Sonnet", provider: ProviderAnthropic, wantModel: "claude-sonnet-4-5-20250929"},

		// Pass through unknown models
		{name: "unknown anthropic model", model: "claude-3-opus-20240229", provider: ProviderAnthropic, wantModel: "claude-3-opus-20240229"},
		{name: "unknown openai model", model: "gpt-4-turbo", provider: ProviderOpenAI, wantModel: "gpt-4-turbo"},
		{name: "unknown google model", model: "gemini-1.5-pro", provider: ProviderGoogle, wantModel: "gemini-1.5-pro"},
		{name: "unknown local model", model: "custom-model", provider: ProviderLocal, wantModel: "custom-model"},

		// Unknown provider passes through
		{name: "unknown provider", model: "some-model", provider: Provider("unknown"), wantModel: "some-model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveModelAlias(tt.model, tt.provider)
			if got != tt.wantModel {
				t.Errorf("resolveModelAlias(%q, %q) = %q, want %q", tt.model, tt.provider, got, tt.wantModel)
			}
		})
	}
}

func TestGetAPIKey(t *testing.T) {
	// Save and restore environment
	origAnthropic := os.Getenv("ANTHROPIC_API_KEY")
	origOpenAI := os.Getenv("OPENAI_API_KEY")
	origGoogle := os.Getenv("GOOGLE_API_KEY")
	t.Cleanup(func() {
		_ = os.Setenv("ANTHROPIC_API_KEY", origAnthropic)
		_ = os.Setenv("OPENAI_API_KEY", origOpenAI)
		_ = os.Setenv("GOOGLE_API_KEY", origGoogle)
	})

	tests := []struct {
		name     string
		provider Provider
		envVar   string
		envValue string
		wantKey  string
		wantErr  bool
	}{
		{
			name:     "anthropic key set",
			provider: ProviderAnthropic,
			envVar:   "ANTHROPIC_API_KEY",
			envValue: "sk-ant-test123",
			wantKey:  "sk-ant-test123",
			wantErr:  false,
		},
		{
			name:     "anthropic key not set",
			provider: ProviderAnthropic,
			envVar:   "ANTHROPIC_API_KEY",
			envValue: "",
			wantErr:  true,
		},
		{
			name:     "openai key set",
			provider: ProviderOpenAI,
			envVar:   "OPENAI_API_KEY",
			envValue: "sk-openai-test456",
			wantKey:  "sk-openai-test456",
			wantErr:  false,
		},
		{
			name:     "openai key not set",
			provider: ProviderOpenAI,
			envVar:   "OPENAI_API_KEY",
			envValue: "",
			wantErr:  true,
		},
		{
			name:     "google key set",
			provider: ProviderGoogle,
			envVar:   "GOOGLE_API_KEY",
			envValue: "AIza-test789",
			wantKey:  "AIza-test789",
			wantErr:  false,
		},
		{
			name:     "google key not set",
			provider: ProviderGoogle,
			envVar:   "GOOGLE_API_KEY",
			envValue: "",
			wantErr:  true,
		},
		{
			name:     "local provider - no key needed",
			provider: ProviderLocal,
			wantKey:  "not-needed",
			wantErr:  false,
		},
		{
			name:     "unsupported provider",
			provider: Provider("unsupported"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars
			_ = os.Unsetenv("ANTHROPIC_API_KEY")
			_ = os.Unsetenv("OPENAI_API_KEY")
			_ = os.Unsetenv("GOOGLE_API_KEY")

			// Set the specific env var for this test
			if tt.envVar != "" && tt.envValue != "" {
				_ = os.Setenv(tt.envVar, tt.envValue)
			}

			key, err := getAPIKey(tt.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAPIKey(%q) error = %v, wantErr %v", tt.provider, err, tt.wantErr)
				return
			}
			if key != tt.wantKey {
				t.Errorf("getAPIKey(%q) = %q, want %q", tt.provider, key, tt.wantKey)
			}
		})
	}
}

func TestDoRequest_Success(t *testing.T) {
	client := &Client{
		httpClient: &mockHTTPDoer{
			response: mockResponse(http.StatusOK, `{"result": "success"}`),
		},
	}

	body, err := client.doRequest(context.Background(), "https://example.com/api", map[string]string{"key": "value"}, nil)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}

	expected := `{"result": "success"}`
	if string(body) != expected {
		t.Errorf("doRequest() body = %q, want %q", string(body), expected)
	}
}

func TestDoRequest_ErrorStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
	}{
		{
			name:       "400 bad request",
			statusCode: http.StatusBadRequest,
			body:       `{"error": "invalid request"}`,
			wantErr:    "API error (status 400)",
		},
		{
			name:       "401 unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error": "invalid api key"}`,
			wantErr:    "API error (status 401)",
		},
		{
			name:       "429 rate limited",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error": "rate limit exceeded"}`,
			wantErr:    "API error (status 429)",
		},
		{
			name:       "500 server error",
			statusCode: http.StatusInternalServerError,
			body:       `{"error": "internal server error"}`,
			wantErr:    "API error (status 500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				httpClient: &mockHTTPDoer{
					response: mockResponse(tt.statusCode, tt.body),
				},
			}

			_, err := client.doRequest(context.Background(), "https://example.com/api", nil, nil)
			if err == nil {
				t.Fatal("doRequest() expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("doRequest() error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestDoRequest_NetworkError(t *testing.T) {
	networkErr := errors.New("connection refused")
	client := &Client{
		httpClient: &mockHTTPDoer{
			err: networkErr,
		},
	}

	_, err := client.doRequest(context.Background(), "https://example.com/api", nil, nil)
	if err == nil {
		t.Fatal("doRequest() expected error")
	}
	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("doRequest() error = %q, want to contain 'request failed'", err.Error())
	}
}

func TestDoRequest_ErrorTruncation(t *testing.T) {
	// Create error body longer than 500 characters
	longError := strings.Repeat("x", 600)
	client := &Client{
		httpClient: &mockHTTPDoer{
			response: mockResponse(http.StatusBadRequest, longError),
		},
	}

	_, err := client.doRequest(context.Background(), "https://example.com/api", nil, nil)
	if err == nil {
		t.Fatal("doRequest() expected error")
	}

	errMsg := err.Error()
	// The truncated error should contain at most 500 chars of the body
	// Full error would be over 600, truncated should be shorter
	if len(errMsg) > 600 {
		t.Errorf("doRequest() error message not truncated, len = %d", len(errMsg))
	}
	// Verify truncation happened (message doesn't contain the full 600 x's)
	if strings.Count(errMsg, "x") >= 600 {
		t.Error("doRequest() error body not truncated")
	}
}

func TestDoRequest_Headers(t *testing.T) {
	var capturedReq *http.Request
	client := &Client{
		httpClient: &capturingHTTPDoer{
			capturedReq: &capturedReq,
			response:    mockResponse(http.StatusOK, `{}`),
		},
	}

	headers := map[string]string{
		"Authorization": "Bearer test-token",
		"X-Custom":      "custom-value",
	}

	_, err := client.doRequest(context.Background(), "https://example.com/api", nil, headers)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}

	if capturedReq == nil {
		t.Fatal("request was not captured")
	}

	// Check Content-Type is always set
	if ct := capturedReq.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want 'application/json'", ct)
	}

	// Check custom headers
	if auth := capturedReq.Header.Get("Authorization"); auth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want 'Bearer test-token'", auth)
	}
	if custom := capturedReq.Header.Get("X-Custom"); custom != "custom-value" {
		t.Errorf("X-Custom = %q, want 'custom-value'", custom)
	}
}

// capturingHTTPDoer captures the request for inspection.
type capturingHTTPDoer struct {
	capturedReq **http.Request
	response    *http.Response
}

func (c *capturingHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	*c.capturedReq = req
	return c.response, nil
}

func TestLocalServerURL(t *testing.T) {
	// Save and restore
	orig := os.Getenv("LOCAL_LLM_URL")
	t.Cleanup(func() {
		if orig != "" {
			_ = os.Setenv("LOCAL_LLM_URL", orig)
		} else {
			_ = os.Unsetenv("LOCAL_LLM_URL")
		}
	})

	tests := []struct {
		name     string
		envValue string
		wantURL  string
	}{
		{
			name:     "default when unset",
			envValue: "",
			wantURL:  "http://localhost:1234/v1",
		},
		{
			name:     "custom URL",
			envValue: "http://localhost:8080/api",
			wantURL:  "http://localhost:8080/api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				_ = os.Setenv("LOCAL_LLM_URL", tt.envValue)
			} else {
				_ = os.Unsetenv("LOCAL_LLM_URL")
			}

			got := LocalServerURL()
			if got != tt.wantURL {
				t.Errorf("LocalServerURL() = %q, want %q", got, tt.wantURL)
			}
		})
	}
}

func TestSupportedProviders(t *testing.T) {
	providers := SupportedProviders()

	expected := []string{"anthropic", "openai", "google", "local"}
	if len(providers) != len(expected) {
		t.Errorf("SupportedProviders() length = %d, want %d", len(providers), len(expected))
	}

	for _, exp := range expected {
		if !slices.Contains(providers, exp) {
			t.Errorf("SupportedProviders() missing %q", exp)
		}
	}
}

func TestComplete_UnsupportedProvider(t *testing.T) {
	client := &Client{
		provider: Provider("unsupported"),
	}

	_, err := client.Complete(context.Background(), Request{Prompt: "test"})
	if err == nil {
		t.Fatal("Complete() expected error for unsupported provider")
	}
	if !strings.Contains(err.Error(), "unsupported provider") {
		t.Errorf("Complete() error = %q, want to contain 'unsupported provider'", err.Error())
	}
}
