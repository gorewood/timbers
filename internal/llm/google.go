package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Google Gemini API types.
type googleRequest struct {
	Contents         []googleContent      `json:"contents"`
	SystemInstruct   *googleContent       `json:"systemInstruction,omitempty"`
	GenerationConfig *googleGenerationCfg `json:"generationConfig,omitempty"`
}

type googleContent struct {
	Parts []googlePart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type googlePart struct {
	Text string `json:"text"`
}

type googleGenerationCfg struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
}

type googleResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) completeGoogle(ctx context.Context, req Request) (*Response, error) {
	body := c.buildGoogleRequest(req)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", c.model)
	headers := map[string]string{"x-goog-api-key": c.apiKey}

	respBody, err := c.doRequest(ctx, url, body, headers)
	if err != nil {
		return nil, err
	}

	return parseGoogleResponse(respBody, c.model)
}

func (c *Client) buildGoogleRequest(req Request) googleRequest {
	body := googleRequest{
		Contents: []googleContent{{
			Parts: []googlePart{{Text: req.Prompt}},
			Role:  "user",
		}},
	}

	if req.System != "" {
		body.SystemInstruct = &googleContent{
			Parts: []googlePart{{Text: req.System}},
		}
	}

	if req.MaxTokens > 0 || req.Temperature > 0 {
		body.GenerationConfig = &googleGenerationCfg{
			MaxOutputTokens: req.MaxTokens,
			Temperature:     req.Temperature,
		}
	}

	return body
}

func parseGoogleResponse(respBody []byte, model string) (*Response, error) {
	var result googleResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("empty response from API")
	}

	var content strings.Builder
	for _, part := range result.Candidates[0].Content.Parts {
		content.WriteString(part.Text)
	}

	return &Response{Content: content.String(), Model: model}, nil
}
