// Package executor provides workflow execution for multi-agent-spec teams.
package executor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/plexusone/omnillm-core/provider"
)

// StepResult holds the output of a workflow step.
type StepResult struct {
	StepName string
	Outputs  map[string]any
	Error    error
}

// Executor runs multi-agent-spec workflows.
type Executor struct {
	llm       provider.Provider
	logger    *slog.Logger
	maxSteps  int
	stepCache map[string]*StepResult
	mu        sync.RWMutex
}

// Config holds executor configuration.
type Config struct {
	LLM      provider.Provider
	Logger   *slog.Logger
	MaxSteps int
}

// New creates a new Executor.
func New(cfg Config) *Executor {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.MaxSteps == 0 {
		cfg.MaxSteps = 100
	}

	return &Executor{
		llm:       cfg.LLM,
		logger:    cfg.Logger,
		maxSteps:  cfg.MaxSteps,
		stepCache: make(map[string]*StepResult),
	}
}

// Step represents a workflow step to execute.
type Step struct {
	Name         string
	AgentName    string
	Instructions string
	Inputs       map[string]any
	DependsOn    []string
}

// ExecuteStep runs a single workflow step.
func (e *Executor) ExecuteStep(ctx context.Context, step Step) (*StepResult, error) {
	e.logger.Info("executing step",
		"step", step.Name,
		"agent", step.AgentName,
	)

	// Check dependencies
	for _, dep := range step.DependsOn {
		e.mu.RLock()
		result, ok := e.stepCache[dep]
		e.mu.RUnlock()

		if !ok {
			return nil, fmt.Errorf("dependency %s not yet executed", dep)
		}
		if result.Error != nil {
			return nil, fmt.Errorf("dependency %s failed: %w", dep, result.Error)
		}
	}

	// Build prompt with inputs
	prompt := buildPrompt(step)

	// Execute via LLM
	resp, err := e.llm.CreateChatCompletion(ctx, &provider.ChatCompletionRequest{
		Model: "claude-sonnet-4-20250514", // TODO: Get from agent spec
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: step.Instructions},
			{Role: provider.RoleUser, Content: prompt},
		},
	})
	if err != nil {
		return &StepResult{
			StepName: step.Name,
			Error:    err,
		}, err
	}

	// Parse outputs
	outputs := make(map[string]any)
	if len(resp.Choices) > 0 {
		outputs["response"] = resp.Choices[0].Message.Content
	}

	result := &StepResult{
		StepName: step.Name,
		Outputs:  outputs,
	}

	// Cache result
	e.mu.Lock()
	e.stepCache[step.Name] = result
	e.mu.Unlock()

	e.logger.Info("step completed",
		"step", step.Name,
		"output_keys", len(outputs),
	)

	return result, nil
}

// GetStepResult retrieves a cached step result.
func (e *Executor) GetStepResult(stepName string) (*StepResult, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result, ok := e.stepCache[stepName]
	return result, ok
}

// Reset clears the step cache.
func (e *Executor) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stepCache = make(map[string]*StepResult)
}

func buildPrompt(step Step) string {
	prompt := fmt.Sprintf("## Task: %s\n\n", step.Name)

	if len(step.Inputs) > 0 {
		prompt += "## Inputs\n\n"
		for key, value := range step.Inputs {
			prompt += fmt.Sprintf("- **%s**: %v\n", key, value)
		}
		prompt += "\n"
	}

	prompt += "## Instructions\n\nPlease complete the task and provide your output in a structured format.\n"

	return prompt
}
