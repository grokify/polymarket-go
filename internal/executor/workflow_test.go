package executor

import (
	"testing"
)

func TestWorkflowTypeDeterministic(t *testing.T) {
	tests := []struct {
		wfType WorkflowType
		want   bool
	}{
		{WorkflowTypeChain, true},
		{WorkflowTypeScatter, true},
		{WorkflowTypeGraph, true},
		{WorkflowTypeCrew, false},
		{WorkflowTypeSwarm, false},
		{WorkflowTypeCouncil, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.wfType), func(t *testing.T) {
			if got := tt.wfType.IsDeterministic(); got != tt.want {
				t.Errorf("IsDeterministic() = %v, want %v", got, tt.want)
			}
			if got := tt.wfType.IsSelfDirected(); got != !tt.want {
				t.Errorf("IsSelfDirected() = %v, want %v", got, !tt.want)
			}
		})
	}
}

func TestTopoSort(t *testing.T) {
	tests := []struct {
		name    string
		steps   []WorkflowStep
		want    []string // expected order of step names
		wantErr bool
	}{
		{
			name: "simple chain",
			steps: []WorkflowStep{
				{Name: "step1"},
				{Name: "step2", DependsOn: []string{"step1"}},
				{Name: "step3", DependsOn: []string{"step2"}},
			},
			want: []string{"step1", "step2", "step3"},
		},
		{
			name: "parallel then join",
			steps: []WorkflowStep{
				{Name: "start"},
				{Name: "branch_a", DependsOn: []string{"start"}},
				{Name: "branch_b", DependsOn: []string{"start"}},
				{Name: "join", DependsOn: []string{"branch_a", "branch_b"}},
			},
			want: []string{"start", "branch_a", "branch_b", "join"},
		},
		{
			name: "cycle detection",
			steps: []WorkflowStep{
				{Name: "step1", DependsOn: []string{"step3"}},
				{Name: "step2", DependsOn: []string{"step1"}},
				{Name: "step3", DependsOn: []string{"step2"}},
			},
			wantErr: true,
		},
		{
			name:  "empty workflow",
			steps: []WorkflowStep{},
			want:  []string{},
		},
		{
			name: "single step",
			steps: []WorkflowStep{
				{Name: "only"},
			},
			want: []string{"only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := topoSort(tt.steps)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != len(tt.want) {
				t.Fatalf("result length = %d, want %d", len(result), len(tt.want))
			}

			for i, step := range result {
				if step.Name != tt.want[i] {
					t.Errorf("result[%d].Name = %q, want %q", i, step.Name, tt.want[i])
				}
			}
		})
	}
}
