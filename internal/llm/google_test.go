package llm

import (
	"strings"
	"testing"
)

func TestParseGoogleResponse_Success(t *testing.T) {
	responseJSON := `{
		"candidates": [
			{
				"content": {
					"parts": [
						{"text": "Hello, "},
						{"text": "world!"}
					]
				}
			}
		]
	}`

	resp, err := parseGoogleResponse([]byte(responseJSON), "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("parseGoogleResponse() error = %v", err)
	}

	if resp.Content != "Hello, world!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello, world!")
	}
	if resp.Model != "gemini-2.5-flash" {
		t.Errorf("Model = %q, want %q", resp.Model, "gemini-2.5-flash")
	}
}

func TestParseGoogleResponse_SinglePart(t *testing.T) {
	responseJSON := `{
		"candidates": [
			{
				"content": {
					"parts": [{"text": "Single response"}]
				}
			}
		]
	}`

	resp, err := parseGoogleResponse([]byte(responseJSON), "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("parseGoogleResponse() error = %v", err)
	}

	if resp.Content != "Single response" {
		t.Errorf("Content = %q, want %q", resp.Content, "Single response")
	}
}

func TestParseGoogleResponse_ErrorResponse(t *testing.T) {
	responseJSON := `{
		"error": {
			"message": "Invalid API key"
		}
	}`

	_, err := parseGoogleResponse([]byte(responseJSON), "gemini-2.5-flash")
	if err == nil {
		t.Fatal("parseGoogleResponse() expected error")
	}
	if !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("error = %q, want to contain 'Invalid API key'", err.Error())
	}
}

func TestParseGoogleResponse_EmptyCandidates(t *testing.T) {
	responseJSON := `{"candidates": []}`

	_, err := parseGoogleResponse([]byte(responseJSON), "gemini-2.5-flash")
	if err == nil {
		t.Fatal("parseGoogleResponse() expected error for empty candidates")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("error = %q, want to contain 'empty response'", err.Error())
	}
}

func TestParseGoogleResponse_EmptyParts(t *testing.T) {
	responseJSON := `{
		"candidates": [
			{
				"content": {
					"parts": []
				}
			}
		]
	}`

	_, err := parseGoogleResponse([]byte(responseJSON), "gemini-2.5-flash")
	if err == nil {
		t.Fatal("parseGoogleResponse() expected error for empty parts")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("error = %q, want to contain 'empty response'", err.Error())
	}
}

func TestParseGoogleResponse_InvalidJSON(t *testing.T) {
	_, err := parseGoogleResponse([]byte("not valid json"), "gemini-2.5-flash")
	if err == nil {
		t.Fatal("parseGoogleResponse() expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse response") {
		t.Errorf("error = %q, want to contain 'parse response'", err.Error())
	}
}

func TestParseGoogleResponse_NoCandidatesField(t *testing.T) {
	responseJSON := `{}`

	_, err := parseGoogleResponse([]byte(responseJSON), "gemini-2.5-flash")
	if err == nil {
		t.Fatal("parseGoogleResponse() expected error for missing candidates")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("error = %q, want to contain 'empty response'", err.Error())
	}
}

func TestBuildGoogleRequest_BasicPrompt(t *testing.T) {
	client := &Client{model: "gemini-2.5-flash"}

	req := client.buildGoogleRequest(Request{
		Prompt: "Hello",
	})

	// Verify user prompt is included
	if len(req.Contents) != 1 {
		t.Fatalf("Contents length = %d, want 1", len(req.Contents))
	}
	if req.Contents[0].Role != "user" {
		t.Errorf("Role = %q, want 'user'", req.Contents[0].Role)
	}
	if len(req.Contents[0].Parts) != 1 {
		t.Fatalf("Parts length = %d, want 1", len(req.Contents[0].Parts))
	}
	if req.Contents[0].Parts[0].Text != "Hello" {
		t.Errorf("Text = %q, want 'Hello'", req.Contents[0].Parts[0].Text)
	}

	// No system instruction for basic request
	if req.SystemInstruct != nil {
		t.Error("SystemInstruct should be nil for basic request")
	}

	// No generation config for basic request
	if req.GenerationConfig != nil {
		t.Error("GenerationConfig should be nil for basic request")
	}
}

func TestBuildGoogleRequest_WithSystem(t *testing.T) {
	client := &Client{model: "gemini-2.5-flash"}

	req := client.buildGoogleRequest(Request{
		System: "You are a helpful assistant",
		Prompt: "Hello",
	})

	// Verify system instruction is included
	if req.SystemInstruct == nil {
		t.Fatal("SystemInstruct should not be nil")
	}
	if len(req.SystemInstruct.Parts) != 1 {
		t.Fatalf("SystemInstruct.Parts length = %d, want 1", len(req.SystemInstruct.Parts))
	}
	if req.SystemInstruct.Parts[0].Text != "You are a helpful assistant" {
		t.Errorf("SystemInstruct text = %q, want 'You are a helpful assistant'", req.SystemInstruct.Parts[0].Text)
	}
}

func TestBuildGoogleRequest_WithMaxTokens(t *testing.T) {
	client := &Client{model: "gemini-2.5-flash"}

	req := client.buildGoogleRequest(Request{
		Prompt:    "Hello",
		MaxTokens: 1024,
	})

	// Verify generation config is included
	if req.GenerationConfig == nil {
		t.Fatal("GenerationConfig should not be nil")
	}
	if req.GenerationConfig.MaxOutputTokens != 1024 {
		t.Errorf("MaxOutputTokens = %d, want 1024", req.GenerationConfig.MaxOutputTokens)
	}
}

func TestBuildGoogleRequest_WithTemperature(t *testing.T) {
	client := &Client{model: "gemini-2.5-flash"}

	req := client.buildGoogleRequest(Request{
		Prompt:      "Hello",
		Temperature: 0.8,
	})

	// Verify generation config is included
	if req.GenerationConfig == nil {
		t.Fatal("GenerationConfig should not be nil")
	}
	if req.GenerationConfig.Temperature != 0.8 {
		t.Errorf("Temperature = %f, want 0.8", req.GenerationConfig.Temperature)
	}
}

func TestBuildGoogleRequest_AllOptions(t *testing.T) {
	client := &Client{model: "gemini-2.5-flash"}

	req := client.buildGoogleRequest(Request{
		System:      "System prompt",
		Prompt:      "User prompt",
		MaxTokens:   2048,
		Temperature: 0.5,
	})

	// Verify all options are set
	if req.SystemInstruct == nil {
		t.Error("SystemInstruct should not be nil")
	}
	if req.GenerationConfig == nil {
		t.Fatal("GenerationConfig should not be nil")
	}
	if req.GenerationConfig.MaxOutputTokens != 2048 {
		t.Errorf("MaxOutputTokens = %d, want 2048", req.GenerationConfig.MaxOutputTokens)
	}
	if req.GenerationConfig.Temperature != 0.5 {
		t.Errorf("Temperature = %f, want 0.5", req.GenerationConfig.Temperature)
	}
}

func TestBuildGoogleRequest_ZeroValuesOmitConfig(t *testing.T) {
	client := &Client{model: "gemini-2.5-flash"}

	req := client.buildGoogleRequest(Request{
		Prompt:      "Hello",
		MaxTokens:   0, // Zero should not create config
		Temperature: 0, // Zero should not create config
	})

	// GenerationConfig should be nil when both values are zero
	if req.GenerationConfig != nil {
		t.Error("GenerationConfig should be nil when MaxTokens and Temperature are both 0")
	}
}
