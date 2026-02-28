// Package llm provides a minimal multi-provider LLM client.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorewood/timbers/internal/output"
)

// Provider represents an LLM provider.
type Provider string

// Supported LLM providers.
const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
	ProviderGoogle    Provider = "google"
	ProviderLocal     Provider = "local"
)

// Request represents an LLM completion request.
type Request struct {
	System      string  // System prompt
	Prompt      string  // User prompt
	Temperature float64 // Temperature (0 uses default)
	MaxTokens   int     // Max tokens (0 uses default)
}

// Response represents an LLM completion response.
type Response struct {
	Content string // Generated content
	Model   string // Model used
}

// HTTPDoer defines the HTTP operations required by Client.
// This allows injection of test doubles for testing.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Client is a provider-agnostic LLM client.
type Client struct {
	provider   Provider
	model      string
	apiKey     string
	httpClient HTTPDoer
}

// New creates a new LLM client for the given model.
// Model can be a combined format like "claude-haiku", "gemini-flash", "gpt-5-nano".
// Provider is inferred from the model name if not specified.
func New(model string, provider Provider) (*Client, error) {
	// Parse combined provider-model format (e.g., "claude-haiku", "gemini-flash")
	if provider == "" {
		provider, model = parseProviderPrefix(model)
	}

	if provider == "" {
		provider = inferProvider(model)
	}

	model = resolveModelAlias(model, provider)

	apiKey, err := getAPIKey(provider)
	if err != nil {
		return nil, err
	}

	return &Client{
		provider: provider,
		model:    model,
		apiKey:   apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}, nil
}

// Complete generates a completion for the given request.
func (c *Client) Complete(ctx context.Context, req Request) (*Response, error) {
	switch c.provider {
	case ProviderAnthropic:
		return c.completeAnthropic(ctx, req)
	case ProviderOpenAI:
		return c.completeOpenAI(ctx, req)
	case ProviderGoogle:
		return c.completeGoogle(ctx, req)
	case ProviderLocal:
		return c.completeLocal(ctx, req)
	default:
		return nil, output.NewUserError(fmt.Sprintf("unsupported provider: %s", c.provider))
	}
}

// providerPrefix maps explicit prefixes to providers for combined format parsing.
var providerPrefixes = map[string]Provider{
	"claude-":    ProviderAnthropic,
	"anthropic-": ProviderAnthropic,
	"gemini-":    ProviderGoogle,
	"google-":    ProviderGoogle,
	"openai-":    ProviderOpenAI,
	"local-":     ProviderLocal,
}

// parseProviderPrefix extracts provider from combined format like "claude-haiku".
// Returns empty provider if no prefix matches.
func parseProviderPrefix(model string) (Provider, string) {
	modelLower := strings.ToLower(model)
	for prefix, provider := range providerPrefixes {
		if strings.HasPrefix(modelLower, prefix) {
			return provider, model[len(prefix):]
		}
	}
	return "", model
}

// providerPattern maps model substrings to providers.
type providerPattern struct {
	substring string
	provider  Provider
}

// providerPatterns checked in order; first match wins.
var providerPatterns = []providerPattern{
	{"claude", ProviderAnthropic},
	{"haiku", ProviderAnthropic},
	{"sonnet", ProviderAnthropic},
	{"opus", ProviderAnthropic},
	{"gpt", ProviderOpenAI},
	{"nano", ProviderOpenAI},
	{"o1", ProviderOpenAI},
	{"o3", ProviderOpenAI},
	{"o4", ProviderOpenAI},
	{"gemini", ProviderGoogle},
	{"flash", ProviderGoogle},
	{"local", ProviderLocal},
	{"qwen", ProviderLocal},
	{"llama", ProviderLocal},
	{"mistral", ProviderLocal},
	{"phi", ProviderLocal},
}

// inferProvider guesses the provider from the model name.
func inferProvider(model string) Provider {
	modelLower := strings.ToLower(model)
	for _, p := range providerPatterns {
		if strings.Contains(modelLower, p.substring) {
			return p.provider
		}
	}
	return ProviderAnthropic
}

// Model aliases - just convenient shorthands, users can pass full names directly.
var modelAliases = map[Provider]map[string]string{
	ProviderAnthropic: {
		"haiku":  "claude-haiku-4-5-20251001",
		"sonnet": "claude-sonnet-4-5-20250929",
		"opus":   "claude-opus-4-6",
	},
	ProviderOpenAI: {
		"nano": "gpt-5-nano",
		"mini": "gpt-5-mini",
		"gpt":  "gpt-5.2",
	},
	ProviderGoogle: {
		"flash":      "gemini-3-flash-preview",
		"flash-lite": "gemini-2.5-flash-lite",
		"pro":        "gemini-3-pro-preview",
	},
	ProviderLocal: {
		"local": "default",
	},
}

// resolveModelAlias expands shorthand aliases, passes through unknown names.
func resolveModelAlias(model string, provider Provider) string {
	if aliases, ok := modelAliases[provider]; ok {
		if resolved, ok := aliases[strings.ToLower(model)]; ok {
			return resolved
		}
	}
	return model
}

// envVarForProvider maps providers to their API key environment variables.
var envVarForProvider = map[Provider]string{
	ProviderAnthropic: "ANTHROPIC_API_KEY",
	ProviderOpenAI:    "OPENAI_API_KEY",
	ProviderGoogle:    "GOOGLE_API_KEY",
	ProviderLocal:     "", // Local provider doesn't require an API key
}

func getAPIKey(provider Provider) (string, error) {
	envVar, ok := envVarForProvider[provider]
	if !ok {
		return "", output.NewUserError(fmt.Sprintf("unsupported provider: %s", provider))
	}

	// Local provider doesn't require an API key
	if envVar == "" {
		return "not-needed", nil
	}

	key := os.Getenv(envVar)
	if key == "" {
		return "", output.NewUserError(envVar + " environment variable not set")
	}
	return key, nil
}

// LocalServerURL returns the URL for the local LLM server.
// Defaults to http://localhost:1234/v1 (LM Studio default).
func LocalServerURL() string {
	if url := os.Getenv("LOCAL_LLM_URL"); url != "" {
		return url
	}
	return "http://localhost:1234/v1"
}

// doRequest performs an HTTP POST request with JSON body.
func (c *Client) doRequest(ctx context.Context, url string, body any, headers map[string]string) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, output.NewSystemErrorWithCause("failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, output.NewSystemErrorWithCause("failed to create request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, output.NewSystemErrorWithCause("request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, output.NewSystemErrorWithCause("failed to read response", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Truncate error body to prevent sensitive data leakage and memory issues
		errBody := string(respBody)
		if len(errBody) > 500 {
			errBody = errBody[:500]
		}
		return nil, output.NewSystemError(fmt.Sprintf("API error (status %d): %s", resp.StatusCode, errBody))
	}

	return respBody, nil
}

// SupportedProviders returns a list of supported providers.
func SupportedProviders() []string {
	return []string{string(ProviderAnthropic), string(ProviderOpenAI), string(ProviderGoogle), string(ProviderLocal)}
}

// cloudProviders lists providers that require API keys, in display order.
// Update this when adding a new cloud provider to envVarForProvider.
var cloudProviders = []Provider{ProviderAnthropic, ProviderOpenAI, ProviderGoogle}

// APIKeyEnvVars returns the environment variable names for cloud provider API keys.
func APIKeyEnvVars() []string {
	var vars []string
	for _, p := range cloudProviders {
		if v := envVarForProvider[p]; v != "" {
			vars = append(vars, v)
		}
	}
	return vars
}
