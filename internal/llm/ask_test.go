package llm

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/plexusone/omnillm-core/provider"
)

// mockProvider implements provider.Provider for testing.
type mockProvider struct {
	response  string
	model     string
	err       error
	streamErr error
	chunks    []string
}

func (m *mockProvider) CreateChatCompletion(ctx context.Context, req *provider.ChatCompletionRequest) (*provider.ChatCompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &provider.ChatCompletionResponse{
		Model: m.model,
		Choices: []provider.ChatCompletionChoice{
			{
				Message: provider.Message{
					Role:    provider.RoleAssistant,
					Content: m.response,
				},
			},
		},
		Usage: provider.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

func (m *mockProvider) CreateChatCompletionStream(ctx context.Context, req *provider.ChatCompletionRequest) (provider.ChatCompletionStream, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	return &mockStream{chunks: m.chunks, index: 0}, nil
}

func (m *mockProvider) Close() error {
	return nil
}

func (m *mockProvider) Name() string {
	return "mock"
}

// mockStream implements provider.ChatCompletionStream for testing.
type mockStream struct {
	chunks []string
	index  int
}

func (s *mockStream) Recv() (*provider.ChatCompletionChunk, error) {
	if s.index >= len(s.chunks) {
		return nil, io.EOF
	}
	chunk := s.chunks[s.index]
	s.index++
	return &provider.ChatCompletionChunk{
		Choices: []provider.ChatCompletionChoice{
			{
				Delta: &provider.Message{
					Content: chunk,
				},
			},
		},
	}, nil
}

func (s *mockStream) Close() error {
	return nil
}

func TestAskNilProvider(t *testing.T) {
	_, err := Ask(context.Background(), nil, AskConfig{}, "test prompt")
	if err == nil {
		t.Error("expected error for nil provider")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error should mention nil: %v", err)
	}
}

func TestAskEmptyPrompt(t *testing.T) {
	p := &mockProvider{response: "test"}
	_, err := Ask(context.Background(), p, AskConfig{}, "")
	if err == nil {
		t.Error("expected error for empty prompt")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention empty: %v", err)
	}
}

func TestAskSuccess(t *testing.T) {
	p := &mockProvider{
		response: "Hello, world!",
		model:    "test-model",
	}
	result, err := Ask(context.Background(), p, AskConfig{Model: "test-model"}, "Say hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "Hello, world!" {
		t.Errorf("content = %q, want %q", result.Content, "Hello, world!")
	}
	if result.Model != "test-model" {
		t.Errorf("model = %q, want %q", result.Model, "test-model")
	}
	if result.Usage == nil {
		t.Fatal("usage should not be nil")
	}
	if result.Usage.TotalTokens != 30 {
		t.Errorf("total tokens = %d, want 30", result.Usage.TotalTokens)
	}
}

func TestAskWithSystemPrompt(t *testing.T) {
	p := &mockProvider{response: "test"}

	_, err := Ask(context.Background(), p, AskConfig{
		SystemPrompt: "You are helpful",
		Model:        "test",
	}, "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAskProviderError(t *testing.T) {
	p := &mockProvider{
		err: errors.New("provider failed"),
	}
	_, err := Ask(context.Background(), p, AskConfig{}, "test")
	if err == nil {
		t.Error("expected error from provider")
	}
	if !strings.Contains(err.Error(), "chat completion failed") {
		t.Errorf("error should wrap provider error: %v", err)
	}
}

func TestAskDefaultMaxTokens(t *testing.T) {
	p := &mockProvider{response: "test"}
	_, err := Ask(context.Background(), p, AskConfig{}, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify default is used - implementation passes DefaultMaxTokens
}

func TestAskStreamNilProvider(t *testing.T) {
	err := AskStream(context.Background(), nil, AskConfig{}, "test", func(s string) error { return nil })
	if err == nil {
		t.Error("expected error for nil provider")
	}
}

func TestAskStreamEmptyPrompt(t *testing.T) {
	p := &mockProvider{}
	err := AskStream(context.Background(), p, AskConfig{}, "", func(s string) error { return nil })
	if err == nil {
		t.Error("expected error for empty prompt")
	}
}

func TestAskStreamNilFunction(t *testing.T) {
	p := &mockProvider{}
	err := AskStream(context.Background(), p, AskConfig{}, "test", nil)
	if err == nil {
		t.Error("expected error for nil stream function")
	}
}

func TestAskStreamSuccess(t *testing.T) {
	p := &mockProvider{
		chunks: []string{"Hello", ", ", "world", "!"},
	}
	var collected strings.Builder
	err := AskStream(context.Background(), p, AskConfig{}, "test", func(chunk string) error {
		collected.WriteString(chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if collected.String() != "Hello, world!" {
		t.Errorf("collected = %q, want %q", collected.String(), "Hello, world!")
	}
}

func TestAskStreamError(t *testing.T) {
	p := &mockProvider{
		streamErr: errors.New("stream failed"),
	}
	err := AskStream(context.Background(), p, AskConfig{}, "test", func(s string) error { return nil })
	if err == nil {
		t.Error("expected error from stream creation")
	}
	if !strings.Contains(err.Error(), "stream creation failed") {
		t.Errorf("error should wrap stream error: %v", err)
	}
}

func TestAskStreamFunctionError(t *testing.T) {
	p := &mockProvider{
		chunks: []string{"Hello"},
	}
	err := AskStream(context.Background(), p, AskConfig{}, "test", func(s string) error {
		return errors.New("handler error")
	})
	if err == nil {
		t.Error("expected error from stream function")
	}
	if !strings.Contains(err.Error(), "stream function failed") {
		t.Errorf("error should wrap handler error: %v", err)
	}
}

func TestBuildPromptFromArgsEmpty(t *testing.T) {
	_, err := BuildPromptFromArgs(nil)
	if err == nil {
		t.Error("expected error for empty args")
	}
	if !strings.Contains(err.Error(), "no prompt") {
		t.Errorf("error should mention no prompt: %v", err)
	}
}

func TestBuildPromptFromArgsSingle(t *testing.T) {
	prompt, err := BuildPromptFromArgs([]string{"hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prompt != "hello" {
		t.Errorf("prompt = %q, want %q", prompt, "hello")
	}
}

func TestBuildPromptFromArgsMultiple(t *testing.T) {
	prompt, err := BuildPromptFromArgs([]string{"hello", "world", "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prompt != "hello world test" {
		t.Errorf("prompt = %q, want %q", prompt, "hello world test")
	}
}

func TestDefaultMaxTokensValue(t *testing.T) {
	if DefaultMaxTokens != 4096 {
		t.Errorf("DefaultMaxTokens = %d, want 4096", DefaultMaxTokens)
	}
}

func TestAskConfigFields(t *testing.T) {
	cfg := AskConfig{
		Model:        "test-model",
		SystemPrompt: "You are helpful",
		MaxTokens:    1000,
	}
	if cfg.Model != "test-model" {
		t.Errorf("Model = %q, want test-model", cfg.Model)
	}
	if cfg.SystemPrompt != "You are helpful" {
		t.Errorf("SystemPrompt = %q, want 'You are helpful'", cfg.SystemPrompt)
	}
	if cfg.MaxTokens != 1000 {
		t.Errorf("MaxTokens = %d, want 1000", cfg.MaxTokens)
	}
}

func TestAskResultFields(t *testing.T) {
	result := AskResult{
		Content: "response",
		Model:   "model",
		Usage: &Usage{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
	}
	if result.Content != "response" {
		t.Errorf("Content = %q, want response", result.Content)
	}
	if result.Model != "model" {
		t.Errorf("Model = %q, want model", result.Model)
	}
	if result.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d, want 10", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 20 {
		t.Errorf("OutputTokens = %d, want 20", result.Usage.OutputTokens)
	}
}

func TestAskNoChoices(t *testing.T) {
	// Create a mock that returns empty choices
	p := &mockProviderNoChoices{}
	_, err := Ask(context.Background(), p, AskConfig{}, "test")
	if err == nil {
		t.Error("expected error for no choices")
	}
	if !strings.Contains(err.Error(), "no response choices") {
		t.Errorf("error should mention no choices: %v", err)
	}
}

// mockProviderNoChoices returns empty choices.
type mockProviderNoChoices struct{}

func (m *mockProviderNoChoices) CreateChatCompletion(ctx context.Context, req *provider.ChatCompletionRequest) (*provider.ChatCompletionResponse, error) {
	return &provider.ChatCompletionResponse{
		Model:   "test",
		Choices: []provider.ChatCompletionChoice{}, // Empty
		Usage:   provider.Usage{},
	}, nil
}

func (m *mockProviderNoChoices) CreateChatCompletionStream(ctx context.Context, req *provider.ChatCompletionRequest) (provider.ChatCompletionStream, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProviderNoChoices) Close() error {
	return nil
}

func (m *mockProviderNoChoices) Name() string {
	return "mock-no-choices"
}
