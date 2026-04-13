// Package executor provides workflow execution for multi-agent-spec teams.
package executor

import (
	"context"

	"github.com/plexusone/omnillm-core/provider"
	omnillm "github.com/plexusone/omnillm-langchaingo"
	"github.com/tmc/langchaingo/llms"
)

// LangChainExecutor wraps omnillm with LangChainGo for advanced agent features.
type LangChainExecutor struct {
	model  llms.Model
	prov   provider.Provider
	config LangChainConfig
}

// LangChainConfig holds configuration for LangChainGo integration.
type LangChainConfig struct {
	Model       string
	Temperature float64
	MaxTokens   int
}

// NewLangChainExecutor creates a new LangChainExecutor using omnillm-langchaingo.
func NewLangChainExecutor(prov provider.Provider, cfg LangChainConfig) *LangChainExecutor {
	// Create LangChainGo model adapter from omnillm provider
	model := omnillm.New(prov, cfg.Model)

	return &LangChainExecutor{
		model:  model,
		prov:   prov,
		config: cfg,
	}
}

// Model returns the underlying LangChainGo model for use with chains/agents.
func (e *LangChainExecutor) Model() llms.Model {
	return e.model
}

// GenerateContent generates content using the LangChainGo model interface.
func (e *LangChainExecutor) GenerateContent(ctx context.Context, prompt string) (string, error) {
	resp, err := llms.GenerateFromSinglePrompt(ctx, e.model, prompt,
		llms.WithTemperature(e.config.Temperature),
		llms.WithMaxTokens(e.config.MaxTokens),
	)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// Call implements a simple call interface for agent steps.
func (e *LangChainExecutor) Call(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error) {
	return e.model.GenerateContent(ctx, messages, opts...)
}
