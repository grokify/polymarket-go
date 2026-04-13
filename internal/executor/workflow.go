// Package executor provides workflow execution for multi-agent-spec teams.
package executor

import (
	"context"
	"fmt"
	"sort"
)

// Workflow represents a multi-agent-spec workflow.
type Workflow struct {
	Name  string
	Type  WorkflowType
	Steps []WorkflowStep
}

// WorkflowType defines the workflow execution pattern.
type WorkflowType string

const (
	WorkflowTypeChain   WorkflowType = "chain"
	WorkflowTypeScatter WorkflowType = "scatter"
	WorkflowTypeGraph   WorkflowType = "graph"
	WorkflowTypeCrew    WorkflowType = "crew"
	WorkflowTypeSwarm   WorkflowType = "swarm"
	WorkflowTypeCouncil WorkflowType = "council"
)

// IsDeterministic returns true if the workflow is schema-controlled.
func (t WorkflowType) IsDeterministic() bool {
	switch t {
	case WorkflowTypeChain, WorkflowTypeScatter, WorkflowTypeGraph:
		return true
	default:
		return false
	}
}

// IsSelfDirected returns true if the workflow is agent-controlled.
func (t WorkflowType) IsSelfDirected() bool {
	return !t.IsDeterministic()
}

// WorkflowStep represents a step in the workflow.
type WorkflowStep struct {
	Name      string
	Agent     string
	DependsOn []string
	Inputs    []Port
	Outputs   []Port
}

// Port represents an input or output of a workflow step.
type Port struct {
	Name string
	Type string
	From string // For inputs: "step_name.output_name"
}

// WorkflowExecutor runs complete workflows.
type WorkflowExecutor struct {
	executor *Executor
	agents   map[string]AgentSpec
}

// AgentSpec holds agent definition from multi-agent-spec.
type AgentSpec struct {
	Name         string
	Instructions string
	Model        string
	Tools        []string
}

// NewWorkflowExecutor creates a new WorkflowExecutor.
func NewWorkflowExecutor(executor *Executor, agents map[string]AgentSpec) *WorkflowExecutor {
	return &WorkflowExecutor{
		executor: executor,
		agents:   agents,
	}
}

// Execute runs the workflow and returns results.
func (w *WorkflowExecutor) Execute(ctx context.Context, workflow Workflow, inputs map[string]any) (map[string]any, error) {
	// Reset executor state
	w.executor.Reset()

	// Topologically sort steps for deterministic workflows
	orderedSteps, err := topoSort(workflow.Steps)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow DAG: %w", err)
	}

	results := make(map[string]any)

	for _, wfStep := range orderedSteps {
		agent, ok := w.agents[wfStep.Agent]
		if !ok {
			return nil, fmt.Errorf("agent %s not found", wfStep.Agent)
		}

		// Build step inputs from workflow port mappings
		stepInputs := make(map[string]any)
		for _, input := range wfStep.Inputs {
			if input.From != "" {
				// Get from previous step output
				value, err := w.resolvePort(input.From)
				if err != nil {
					return nil, fmt.Errorf("step %s: %w", wfStep.Name, err)
				}
				stepInputs[input.Name] = value
			} else if v, ok := inputs[input.Name]; ok {
				// Get from workflow inputs
				stepInputs[input.Name] = v
			}
		}

		// Execute step
		step := Step{
			Name:         wfStep.Name,
			AgentName:    wfStep.Agent,
			Instructions: agent.Instructions,
			Inputs:       stepInputs,
			DependsOn:    wfStep.DependsOn,
		}

		result, err := w.executor.ExecuteStep(ctx, step)
		if err != nil {
			return nil, fmt.Errorf("step %s failed: %w", wfStep.Name, err)
		}

		// Store outputs
		for _, output := range wfStep.Outputs {
			key := fmt.Sprintf("%s.%s", wfStep.Name, output.Name)
			if v, ok := result.Outputs[output.Name]; ok {
				results[key] = v
			}
		}
	}

	return results, nil
}

// resolvePort resolves a port reference like "step_name.output_name".
func (w *WorkflowExecutor) resolvePort(ref string) (any, error) {
	result, ok := w.executor.GetStepResult(ref)
	if !ok {
		return nil, fmt.Errorf("port %s not found", ref)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("port %s has error: %w", ref, result.Error)
	}
	return result.Outputs, nil
}

// topoSort performs topological sort on workflow steps.
func topoSort(steps []WorkflowStep) ([]WorkflowStep, error) {
	// Build adjacency list and in-degree count
	inDegree := make(map[string]int)
	adj := make(map[string][]string)
	stepMap := make(map[string]WorkflowStep)

	for _, step := range steps {
		stepMap[step.Name] = step
		if _, ok := inDegree[step.Name]; !ok {
			inDegree[step.Name] = 0
		}
		for _, dep := range step.DependsOn {
			adj[dep] = append(adj[dep], step.Name)
			inDegree[step.Name]++
		}
	}

	// Find all nodes with in-degree 0
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // Deterministic ordering

	var result []WorkflowStep
	for len(queue) > 0 {
		// Pop from queue
		name := queue[0]
		queue = queue[1:]

		result = append(result, stepMap[name])

		// Reduce in-degree for neighbors
		neighbors := adj[name]
		sort.Strings(neighbors) // Deterministic ordering
		for _, neighbor := range neighbors {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
		sort.Strings(queue) // Maintain deterministic order
	}

	// Check for cycles
	if len(result) != len(steps) {
		return nil, fmt.Errorf("workflow contains cycles")
	}

	return result, nil
}
