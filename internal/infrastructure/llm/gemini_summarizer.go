package llm

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/genai"

	"misskeyRSSbot/internal/domain/repository"
)

type geminiSummarizer struct {
	client       *genai.Client
	model        string
	maxTokens    *int32
	systemPrompt string
	timeout      time.Duration
}

func newGeminiSummarizer(ctx context.Context, cfg Config) (repository.SummarizerRepository, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}

	if cfg.Model == "" {
		return nil, fmt.Errorf("gemini model name is required")
	}

	var maxTokens *int32
	if cfg.MaxTokens > 0 {
		tokens := int32(cfg.MaxTokens)
		maxTokens = &tokens
	}

	systemInstruction := cfg.SystemInstruction
	if systemInstruction == "" {
		systemInstruction = DefaultSystemInstruction
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: cfg.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	return &geminiSummarizer{
		client:       client,
		model:        cfg.Model,
		maxTokens:    maxTokens,
		systemPrompt: systemInstruction,
		timeout:      timeout,
	}, nil
}

func (s *geminiSummarizer) Summarize(ctx context.Context, url, title string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	userPrompt := fmt.Sprintf("以下のURLの記事を要約してください。\n\n記事タイトル: %s\n記事URL: %s", title, url)

	systemInstruction := genai.NewContentFromText(s.systemPrompt, genai.RoleUser)
	userContent := genai.NewContentFromText(userPrompt, genai.RoleUser)

	config := &genai.GenerateContentConfig{
		SystemInstruction: systemInstruction,
		Tools: []*genai.Tool{
			{
				URLContext: &genai.URLContext{},
			},
		},
	}
	if s.maxTokens != nil {
		config.MaxOutputTokens = *s.maxTokens
	}

	resp, err := s.client.Models.GenerateContent(ctx, s.model, []*genai.Content{userContent}, config)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates returned from gemini API")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("no content in candidate response")
	}

	summary := candidate.Content.Parts[0].Text

	return summary, nil
}

func (s *geminiSummarizer) IsEnabled() bool {
	return true
}
