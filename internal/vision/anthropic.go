package vision

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

type AnthropicProvider struct {
	apiKey string
}

func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{apiKey: apiKey}
}

var (
	anthropicModelCache string
	anthropicModelMutex sync.Mutex
)

func (p *AnthropicProvider) getBestModel(ctx context.Context) (string, error) {
	anthropicModelMutex.Lock()
	if anthropicModelCache != "" {
		anthropicModelMutex.Unlock()
		return anthropicModelCache, nil
	}
	anthropicModelMutex.Unlock()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Just fallback if the endpoint doesn't exist or errors (for older API keys)
		return "claude-3-5-sonnet-latest", nil
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "claude-3-5-sonnet-latest", nil
	}

	bestModel := ""
	priority := -1

	for _, m := range result.Data {
		name := m.ID
		currPriority := -1

		if strings.Contains(name, "fable-5") {
			currPriority = 5
		} else if strings.Contains(name, "claude-5") {
			currPriority = 4
		} else if strings.Contains(name, "sonnet-2024") || strings.Contains(name, "3-5-sonnet") {
			currPriority = 3
		} else if strings.Contains(name, "opus-2024") || strings.Contains(name, "3-opus") {
			currPriority = 2
		}

		if currPriority > priority {
			priority = currPriority
			bestModel = name
		}
	}

	if bestModel == "" {
		bestModel = "claude-3-5-sonnet-latest" // ultimate fallback
	}

	anthropicModelMutex.Lock()
	anthropicModelCache = bestModel
	anthropicModelMutex.Unlock()

	return bestModel, nil
}

func (p *AnthropicProvider) AnalyzeImage(ctx context.Context, base64Image string, mimeType string, projectContext string) (string, error) {
	modelName, err := p.getBestModel(ctx)
	if err != nil {
		return "", err
	}

	url := "https://api.anthropic.com/v1/messages"

	payload := map[string]interface{}{
		"model": modelName,
		"max_tokens": 2048,
		"system": SystemPrompt + projectContext,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image",
						"source": map[string]string{
							"type": "base64",
							"media_type": mimeType,
							"data": base64Image,
						},
					},
					{
						"type": "text",
						"text": "Analyze this UI and generate a structural blueprint.",
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Anthropic API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	return result.Content[0].Text, nil
}

func (p *AnthropicProvider) GenerateText(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	modelName, err := p.getBestModel(ctx)
	if err != nil {
		return "", err
	}

	url := "https://api.anthropic.com/v1/messages"

	payload := map[string]interface{}{
		"model":      modelName,
		"max_tokens": 4096,
		"system":     systemPrompt,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": userPrompt,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Anthropic API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	return result.Content[0].Text, nil
}
