package llm

import (
	"strings"
	"testing"
)

func TestParseLocalResponse_Success(t *testing.T) {
	responseJSON := `{
		"choices": [
			{
				"message": {
					"content": "Hello from local!"
				}
			}
		]
	}`

	resp, err := parseLocalResponse([]byte(responseJSON), "llama-3-8b")
	if err != nil {
		t.Fatalf("parseLocalResponse() error = %v", err)
	}

	if resp.Content != "Hello from local!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello from local!")
	}
	if resp.Model != "llama-3-8b" {
		t.Errorf("Model = %q, want %q", resp.Model, "llama-3-8b")
	}
}

func TestParseLocalResponse_ErrorResponse(t *testing.T) {
	responseJSON := `{
		"error": {
			"message": "Model not loaded"
		}
	}`

	_, err := parseLocalResponse([]byte(responseJSON), "llama-3-8b")
	if err == nil {
		t.Fatal("parseLocalResponse() expected error")
	}
	if !strings.Contains(err.Error(), "Model not loaded") {
		t.Errorf("error = %q, want to contain 'Model not loaded'", err.Error())
	}
}

func TestParseLocalResponse_EmptyChoices(t *testing.T) {
	responseJSON := `{"choices": []}`

	_, err := parseLocalResponse([]byte(responseJSON), "llama-3-8b")
	if err == nil {
		t.Fatal("parseLocalResponse() expected error for empty choices")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("error = %q, want to contain 'empty response'", err.Error())
	}
}

func TestParseLocalResponse_InvalidJSON(t *testing.T) {
	_, err := parseLocalResponse([]byte("not valid json"), "llama-3-8b")
	if err == nil {
		t.Fatal("parseLocalResponse() expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse response") {
		t.Errorf("error = %q, want to contain 'parse response'", err.Error())
	}
}

func TestParseLocalResponse_DefaultModel(t *testing.T) {
	responseJSON := `{
		"choices": [{"message": {"content": "OK"}}]
	}`

	tests := []struct {
		name      string
		model     string
		wantModel string
	}{
		{
			name:      "empty model becomes local",
			model:     "",
			wantModel: "local",
		},
		{
			name:      "default becomes local",
			model:     "default",
			wantModel: "local",
		},
		{
			name:      "specific model preserved",
			model:     "llama-3-8b",
			wantModel: "llama-3-8b",
		},
		{
			name:      "qwen model preserved",
			model:     "qwen-72b",
			wantModel: "qwen-72b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := parseLocalResponse([]byte(responseJSON), tt.model)
			if err != nil {
				t.Fatalf("parseLocalResponse() error = %v", err)
			}
			if resp.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", resp.Model, tt.wantModel)
			}
		})
	}
}

func TestBuildLocalRequest_BasicPrompt(t *testing.T) {
	client := &Client{model: "llama-3-8b"}

	req := client.buildLocalRequest(Request{
		Prompt: "Hello",
	})

	// Verify user message is included
	if len(req.Messages) != 1 {
		t.Fatalf("Messages length = %d, want 1", len(req.Messages))
	}
	if req.Messages[0].Role != "user" {
		t.Errorf("Role = %q, want 'user'", req.Messages[0].Role)
	}
	if req.Messages[0].Content != "Hello" {
		t.Errorf("Content = %q, want 'Hello'", req.Messages[0].Content)
	}

	// Model should be preserved
	if req.Model != "llama-3-8b" {
		t.Errorf("Model = %q, want 'llama-3-8b'", req.Model)
	}
}

func TestBuildLocalRequest_WithSystem(t *testing.T) {
	client := &Client{model: "llama-3-8b"}

	req := client.buildLocalRequest(Request{
		System: "You are a helpful assistant",
		Prompt: "Hello",
	})

	// Verify system message is first
	if len(req.Messages) != 2 {
		t.Fatalf("Messages length = %d, want 2", len(req.Messages))
	}
	if req.Messages[0].Role != "system" {
		t.Errorf("First message role = %q, want 'system'", req.Messages[0].Role)
	}
	if req.Messages[0].Content != "You are a helpful assistant" {
		t.Errorf("System content = %q, want 'You are a helpful assistant'", req.Messages[0].Content)
	}
	if req.Messages[1].Role != "user" {
		t.Errorf("Second message role = %q, want 'user'", req.Messages[1].Role)
	}
}

func TestBuildLocalRequest_DefaultModel(t *testing.T) {
	tests := []struct {
		name        string
		clientModel string
		wantModel   string
	}{
		{
			name:        "default becomes empty",
			clientModel: "default",
			wantModel:   "",
		},
		{
			name:        "local becomes empty",
			clientModel: "local",
			wantModel:   "",
		},
		{
			name:        "specific model preserved",
			clientModel: "llama-3-8b",
			wantModel:   "llama-3-8b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{model: tt.clientModel}
			req := client.buildLocalRequest(Request{Prompt: "test"})
			if req.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", req.Model, tt.wantModel)
			}
		})
	}
}

func TestBuildLocalRequest_WithMaxTokens(t *testing.T) {
	client := &Client{model: "llama-3-8b"}

	req := client.buildLocalRequest(Request{
		Prompt:    "Hello",
		MaxTokens: 1024,
	})

	if req.MaxTokens != 1024 {
		t.Errorf("MaxTokens = %d, want 1024", req.MaxTokens)
	}
}

func TestBuildLocalRequest_WithTemperature(t *testing.T) {
	client := &Client{model: "llama-3-8b"}

	req := client.buildLocalRequest(Request{
		Prompt:      "Hello",
		Temperature: 0.7,
	})

	if req.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want 0.7", req.Temperature)
	}
}

func TestBuildLocalRequest_ZeroValuesOmitted(t *testing.T) {
	client := &Client{model: "llama-3-8b"}

	req := client.buildLocalRequest(Request{
		Prompt:      "Hello",
		MaxTokens:   0,
		Temperature: 0,
	})

	// Zero values should remain zero (omitempty will handle serialization)
	if req.MaxTokens != 0 {
		t.Errorf("MaxTokens = %d, want 0", req.MaxTokens)
	}
	if req.Temperature != 0 {
		t.Errorf("Temperature = %f, want 0", req.Temperature)
	}
}

func TestBuildLocalRequest_AllOptions(t *testing.T) {
	client := &Client{model: "mistral-7b"}

	req := client.buildLocalRequest(Request{
		System:      "Be concise",
		Prompt:      "Explain Go",
		MaxTokens:   512,
		Temperature: 0.3,
	})

	// Verify all fields
	if len(req.Messages) != 2 {
		t.Fatalf("Messages length = %d, want 2", len(req.Messages))
	}
	if req.Messages[0].Role != "system" {
		t.Error("First message should be system")
	}
	if req.Messages[1].Role != "user" {
		t.Error("Second message should be user")
	}
	if req.Model != "mistral-7b" {
		t.Errorf("Model = %q, want 'mistral-7b'", req.Model)
	}
	if req.MaxTokens != 512 {
		t.Errorf("MaxTokens = %d, want 512", req.MaxTokens)
	}
	if req.Temperature != 0.3 {
		t.Errorf("Temperature = %f, want 0.3", req.Temperature)
	}
}
