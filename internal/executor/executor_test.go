package executor

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	perrors "github.com/grokify/polymarket-go/internal/errors"
	"github.com/plexusone/omnillm-core/provider"
)

// mockProvider implements provider.Provider for testing.
type mockProvider struct {
	response *provider.ChatCompletionResponse
	err      error
}

func (m *mockProvider) CreateChatCompletion(ctx context.Context, req *provider.ChatCompletionRequest) (*provider.ChatCompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockProvider) CreateChatCompletionStream(ctx context.Context, req *provider.ChatCompletionRequest) (provider.ChatCompletionStream, error) {
	return nil, errors.New("streaming not implemented in mock")
}

func (m *mockProvider) Close() error {
	return nil
}

func (m *mockProvider) Name() string {
	return "mock"
}

func TestNew(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		e := New(Config{})
		if e.maxSteps != 100 {
			t.Errorf("maxSteps = %d, want 100", e.maxSteps)
		}
		if e.logger == nil {
			t.Error("logger is nil")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		logger := slog.Default()
		p := &mockProvider{}
		e := New(Config{
			LLM:      p,
			Logger:   logger,
			MaxSteps: 50,
		})
		if e.maxSteps != 50 {
			t.Errorf("maxSteps = %d, want 50", e.maxSteps)
		}
	})
}

func TestBuildPrompt(t *testing.T) {
	tests := []struct {
		name string
		step Step
		want []string // strings that should be present
	}{
		{
			name: "basic step",
			step: Step{Name: "test-step"},
			want: []string{"## Task: test-step", "## Instructions"},
		},
		{
			name: "step with inputs",
			step: Step{
				Name: "analyze",
				Inputs: map[string]any{
					"market_id": "abc123",
					"price":     0.55,
				},
			},
			want: []string{"## Inputs", "market_id", "abc123", "price"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPrompt(tt.step)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("buildPrompt() missing %q in:\n%s", want, got)
				}
			}
		})
	}
}

func TestExecuteStep_Success(t *testing.T) {
	p := &mockProvider{
		response: &provider.ChatCompletionResponse{
			Choices: []provider.ChatCompletionChoice{
				{Message: provider.Message{Content: "test response"}},
			},
		},
	}
	e := New(Config{LLM: p})

	result, err := e.ExecuteStep(context.Background(), Step{
		Name:         "test",
		AgentName:    "test-agent",
		Instructions: "Do something",
	})
	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}
	if result.StepName != "test" {
		t.Errorf("StepName = %q, want %q", result.StepName, "test")
	}
	if result.Outputs["response"] != "test response" {
		t.Errorf("Outputs[response] = %q, want %q", result.Outputs["response"], "test response")
	}
}

func TestExecuteStep_WithCustomModel(t *testing.T) {
	var capturedModel string
	p := &modelCapturingProvider{
		capturedModel: &capturedModel,
		response: &provider.ChatCompletionResponse{
			Choices: []provider.ChatCompletionChoice{
				{Message: provider.Message{Content: "response"}},
			},
		},
	}
	e := New(Config{LLM: p})

	_, err := e.ExecuteStep(context.Background(), Step{
		Name:         "test",
		Model:        "custom-model",
		Instructions: "Do something",
	})
	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}
	if capturedModel != "custom-model" {
		t.Errorf("model = %q, want %q", capturedModel, "custom-model")
	}
}

type modelCapturingProvider struct {
	capturedModel *string
	response      *provider.ChatCompletionResponse
}

func (m *modelCapturingProvider) CreateChatCompletion(ctx context.Context, req *provider.ChatCompletionRequest) (*provider.ChatCompletionResponse, error) {
	*m.capturedModel = req.Model
	return m.response, nil
}

func (m *modelCapturingProvider) CreateChatCompletionStream(ctx context.Context, req *provider.ChatCompletionRequest) (provider.ChatCompletionStream, error) {
	return nil, errors.New("not implemented")
}

func (m *modelCapturingProvider) Close() error {
	return nil
}

func (m *modelCapturingProvider) Name() string {
	return "model-capturing"
}

func TestExecuteStep_DefaultModel(t *testing.T) {
	var capturedModel string
	p := &modelCapturingProvider{
		capturedModel: &capturedModel,
		response: &provider.ChatCompletionResponse{
			Choices: []provider.ChatCompletionChoice{
				{Message: provider.Message{Content: "response"}},
			},
		},
	}
	e := New(Config{LLM: p})

	_, err := e.ExecuteStep(context.Background(), Step{
		Name:         "test",
		Instructions: "Do something",
	})
	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}
	if capturedModel != DefaultModel {
		t.Errorf("model = %q, want %q", capturedModel, DefaultModel)
	}
}

func TestExecuteStep_LLMError(t *testing.T) {
	p := &mockProvider{
		err: errors.New("LLM unavailable"),
	}
	e := New(Config{LLM: p})

	_, err := e.ExecuteStep(context.Background(), Step{
		Name:         "test",
		Instructions: "Do something",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var llmErr *perrors.LLMError
	if !errors.As(err, &llmErr) {
		t.Fatalf("expected LLMError, got %T: %v", err, err)
	}
	if llmErr.Operation != "ExecuteStep:test" {
		t.Errorf("Operation = %q, want %q", llmErr.Operation, "ExecuteStep:test")
	}
}

func TestExecuteStep_DependencyNotExecuted(t *testing.T) {
	e := New(Config{LLM: &mockProvider{}})

	_, err := e.ExecuteStep(context.Background(), Step{
		Name:      "test",
		DependsOn: []string{"missing-step"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var depErr *perrors.DependencyError
	if !errors.As(err, &depErr) {
		t.Fatalf("expected DependencyError, got %T: %v", err, err)
	}
	if depErr.Dependency != "missing-step" {
		t.Errorf("Dependency = %q, want %q", depErr.Dependency, "missing-step")
	}
	if depErr.Reason != "not yet executed" {
		t.Errorf("Reason = %q, want %q", depErr.Reason, "not yet executed")
	}
}

func TestExecuteStep_DependencyFailed(t *testing.T) {
	p := &mockProvider{
		response: &provider.ChatCompletionResponse{
			Choices: []provider.ChatCompletionChoice{{Message: provider.Message{Content: "ok"}}},
		},
	}
	e := New(Config{LLM: p})

	// Manually cache a failed step
	e.mu.Lock()
	e.stepCache["failed-step"] = &StepResult{
		StepName: "failed-step",
		Error:    errors.New("previous failure"),
	}
	e.mu.Unlock()

	_, err := e.ExecuteStep(context.Background(), Step{
		Name:      "test",
		DependsOn: []string{"failed-step"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var depErr *perrors.DependencyError
	if !errors.As(err, &depErr) {
		t.Fatalf("expected DependencyError, got %T: %v", err, err)
	}
	if depErr.Reason != "failed" {
		t.Errorf("Reason = %q, want %q", depErr.Reason, "failed")
	}
}

func TestGetStepResult(t *testing.T) {
	e := New(Config{})

	// Not found
	_, ok := e.GetStepResult("nonexistent")
	if ok {
		t.Error("expected not found")
	}

	// Found
	e.mu.Lock()
	e.stepCache["test"] = &StepResult{StepName: "test", Outputs: map[string]any{"key": "value"}}
	e.mu.Unlock()

	result, ok := e.GetStepResult("test")
	if !ok {
		t.Error("expected found")
	}
	if result.StepName != "test" {
		t.Errorf("StepName = %q, want %q", result.StepName, "test")
	}
}

func TestReset(t *testing.T) {
	e := New(Config{})

	// Add some cached results
	e.mu.Lock()
	e.stepCache["step1"] = &StepResult{StepName: "step1"}
	e.stepCache["step2"] = &StepResult{StepName: "step2"}
	e.mu.Unlock()

	e.Reset()

	e.mu.RLock()
	count := len(e.stepCache)
	e.mu.RUnlock()

	if count != 0 {
		t.Errorf("stepCache length = %d, want 0", count)
	}
}

func TestExecuteStep_WithDependency(t *testing.T) {
	p := &mockProvider{
		response: &provider.ChatCompletionResponse{
			Choices: []provider.ChatCompletionChoice{{Message: provider.Message{Content: "ok"}}},
		},
	}
	e := New(Config{LLM: p})

	// Execute first step
	_, err := e.ExecuteStep(context.Background(), Step{
		Name:         "step1",
		Instructions: "First step",
	})
	if err != nil {
		t.Fatalf("Step 1 failed: %v", err)
	}

	// Execute second step depending on first
	result, err := e.ExecuteStep(context.Background(), Step{
		Name:         "step2",
		DependsOn:    []string{"step1"},
		Instructions: "Second step",
	})
	if err != nil {
		t.Fatalf("Step 2 failed: %v", err)
	}
	if result.StepName != "step2" {
		t.Errorf("StepName = %q, want %q", result.StepName, "step2")
	}
}

func TestExecuteStep_EmptyChoices(t *testing.T) {
	p := &mockProvider{
		response: &provider.ChatCompletionResponse{
			Choices: []provider.ChatCompletionChoice{},
		},
	}
	e := New(Config{LLM: p})

	result, err := e.ExecuteStep(context.Background(), Step{
		Name:         "test",
		Instructions: "Do something",
	})
	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}

	// With empty choices, outputs should not have "response" key
	if _, ok := result.Outputs["response"]; ok {
		t.Error("expected no response key for empty choices")
	}
}
