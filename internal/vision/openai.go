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

type OpenAIProvider struct {
	apiKey string
}

func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{apiKey: apiKey}
}

var (
	openaiModelCache string
	openaiModelMutex sync.Mutex
)

func (p *OpenAIProvider) getBestModel(ctx context.Context) (string, error) {
	openaiModelMutex.Lock()
	if openaiModelCache != "" {
		openaiModelMutex.Unlock()
		return openaiModelCache, nil
	}
	openaiModelMutex.Unlock()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error when fetching models: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	bestModel := ""
	priority := -1

	for _, m := range result.Data {
		name := m.ID
		currPriority := -1

		if strings.HasPrefix(name, "gpt-5.5") {
			currPriority = 5
		} else if strings.HasPrefix(name, "gpt-5") {
			currPriority = 4
		} else if strings.HasPrefix(name, "gpt-4o") {
			currPriority = 3
		} else if strings.HasPrefix(name, "gpt-4-turbo") {
			currPriority = 2
		} else if strings.HasPrefix(name, "gpt-4-vision") {
			currPriority = 1
		}

		if currPriority > priority {
			priority = currPriority
			bestModel = name
		}
	}

	if bestModel == "" {
		bestModel = "gpt-4o" // ultimate fallback
	}

	openaiModelMutex.Lock()
	openaiModelCache = bestModel
	openaiModelMutex.Unlock()

	return bestModel, nil
}

func (p *OpenAIProvider) AnalyzeImage(ctx context.Context, base64Image string, mimeType string, projectContext string) (string, error) {
	modelName, err := p.getBestModel(ctx)
	if err != nil {
		return "", err
	}

	url := "https://api.openai.com/v1/chat/completions"

	payload := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]interface{}{
			{
				"role": "system",
				"content": SystemPrompt + projectContext,
			},
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "Analyze this UI and generate a structural blueprint.",
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image),
						},
					},
				},
			},
		},
		"max_tokens": 2048,
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
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return result.Choices[0].Message.Content, nil
}

func (p *OpenAIProvider) GenerateText(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	modelName, err := p.getBestModel(ctx)
	if err != nil {
		return "", err
	}

	url := "https://api.openai.com/v1/chat/completions"

	payload := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
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
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	return result.Choices[0].Message.Content, nil
}
