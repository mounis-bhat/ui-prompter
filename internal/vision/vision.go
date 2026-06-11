package vision

import (
	"context"
	"errors"
)

var ErrUnsupportedModel = errors.New("unsupported model")

type Provider interface {
	AnalyzeImage(ctx context.Context, base64Image string, mimeType string, projectContext string) (string, error)
	GenerateText(ctx context.Context, systemPrompt string, userPrompt string) (string, error)
}

func NewProvider(model string, apiKey string) (Provider, error) {
	switch model {
	case "openai":
		return NewOpenAIProvider(apiKey), nil
	case "anthropic":
		return NewAnthropicProvider(apiKey), nil
	case "gemini":
		return NewGeminiProvider(apiKey), nil
	default:
		return nil, ErrUnsupportedModel
	}
}
