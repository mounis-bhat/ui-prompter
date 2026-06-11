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

type GeminiProvider struct {
	apiKey string
}

func NewGeminiProvider(apiKey string) *GeminiProvider {
	return &GeminiProvider{apiKey: apiKey}
}

var (
	geminiModelCache string
	geminiModelMutex sync.Mutex
)

func (p *GeminiProvider) getBestModel(ctx context.Context) (string, error) {
	geminiModelMutex.Lock()
	if geminiModelCache != "" {
		geminiModelMutex.Unlock()
		return geminiModelCache, nil
	}
	geminiModelMutex.Unlock()

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s", p.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini API error when fetching models: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Models []struct {
			Name                       string   `json:"name"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	bestModel := ""
	priority := -1

	for _, m := range result.Models {
		supportsGenerate := false
		for _, method := range m.SupportedGenerationMethods {
			if method == "generateContent" {
				supportsGenerate = true
				break
			}
		}
		if !supportsGenerate {
			continue
		}

		name := strings.TrimPrefix(m.Name, "models/")
		currPriority := -1

		if strings.HasPrefix(name, "gemini-3") && strings.Contains(name, "pro") {
			currPriority = 4
		} else if strings.HasPrefix(name, "gemini-2") && strings.Contains(name, "pro") {
			currPriority = 3
		} else if strings.HasPrefix(name, "gemini-1.5-pro") {
			currPriority = 2
		} else if strings.HasPrefix(name, "gemini-1.5-flash") {
			currPriority = 1
		} else if name == "gemini-pro-vision" {
			currPriority = 0
		}

		if currPriority > priority {
			priority = currPriority
			bestModel = name
		}
	}

	if bestModel == "" {
		bestModel = "gemini-1.5-pro-latest" // ultimate fallback
	}

	geminiModelMutex.Lock()
	geminiModelCache = bestModel
	geminiModelMutex.Unlock()

	return bestModel, nil
}

func (p *GeminiProvider) AnalyzeImage(ctx context.Context, base64Image string, mimeType string, projectContext string) (string, error) {
	modelName, err := p.getBestModel(ctx)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, p.apiKey)

	payload := map[string]interface{}{
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": SystemPrompt + projectContext},
			},
		},
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"inlineData": map[string]string{
							"mimeType": mimeType,
							"data": base64Image,
						},
					},
					{
						"text": "Analyze this UI and generate a structural blueprint.",
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": 2048,
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

func (p *GeminiProvider) GenerateText(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	modelName, err := p.getBestModel(ctx)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, p.apiKey)

	payload := map[string]interface{}{
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": systemPrompt},
			},
		},
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": userPrompt},
				},
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}
