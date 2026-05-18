// Package llm provides utilities for interacting with LLM providers.
package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/plexusone/omnillm-core/provider"
)

// AskConfig holds configuration for the Ask function.
type AskConfig struct {
	// Model is the LLM model to use.
	Model string
	// SystemPrompt is an optional system prompt.
	SystemPrompt string
	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens int
}

// AskResult contains the response from the LLM.
type AskResult struct {
	// Content is the response text.
	Content string
	// Model is the model that was used.
	Model string
	// Usage contains token usage information if available.
	Usage *Usage
}

// Usage contains token usage information.
type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// DefaultMaxTokens is the default maximum tokens if not specified.
const DefaultMaxTokens = 4096

// Ask sends a prompt to an LLM provider and returns the response.
// This function is the core logic that can be easily unit tested with a mock provider.
func Ask(ctx context.Context, p provider.Provider, cfg AskConfig, prompt string) (*AskResult, error) {
	if p == nil {
		return nil, errors.New("provider is nil")
	}
	if prompt == "" {
		return nil, errors.New("prompt cannot be empty")
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = DefaultMaxTokens
	}

	var messages []provider.Message
	if cfg.SystemPrompt != "" {
		messages = append(messages, provider.Message{
			Role:    provider.RoleSystem,
			Content: cfg.SystemPrompt,
		})
	}
	messages = append(messages, provider.Message{
		Role:    provider.RoleUser,
		Content: prompt,
	})

	req := &provider.ChatCompletionRequest{
		Model:     cfg.Model,
		Messages:  messages,
		MaxTokens: &maxTokens,
	}

	resp, err := p.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("no response choices returned")
	}

	result := &AskResult{
		Content: resp.Choices[0].Message.Content,
		Model:   resp.Model,
		Usage: &Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}

	return result, nil
}

// AskStream sends a prompt to an LLM provider and streams the response.
// The stream function is called for each chunk of the response.
func AskStream(ctx context.Context, p provider.Provider, cfg AskConfig, prompt string, streamFn func(chunk string) error) error {
	if p == nil {
		return errors.New("provider is nil")
	}
	if prompt == "" {
		return errors.New("prompt cannot be empty")
	}
	if streamFn == nil {
		return errors.New("stream function is nil")
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = DefaultMaxTokens
	}

	streamTrue := true
	var messages []provider.Message
	if cfg.SystemPrompt != "" {
		messages = append(messages, provider.Message{
			Role:    provider.RoleSystem,
			Content: cfg.SystemPrompt,
		})
	}
	messages = append(messages, provider.Message{
		Role:    provider.RoleUser,
		Content: prompt,
	})

	req := &provider.ChatCompletionRequest{
		Model:     cfg.Model,
		Messages:  messages,
		MaxTokens: &maxTokens,
		Stream:    &streamTrue,
	}

	stream, err := p.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return fmt.Errorf("stream creation failed: %w", err)
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("stream recv failed: %w", err)
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			delta := chunk.Choices[0].Delta.Content
			if delta != "" {
				if err := streamFn(delta); err != nil {
					return fmt.Errorf("stream function failed: %w", err)
				}
			}
		}
	}

	return nil
}

// BuildPromptFromArgs builds a prompt string from command arguments.
// If args is empty, it returns an error indicating stdin should be used.
func BuildPromptFromArgs(args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("no prompt provided: pass as arguments or pipe to stdin")
	}
	return strings.Join(args, " "), nil
}
