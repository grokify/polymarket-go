package loader

import (
	"testing"
)

func TestParseAgentMarkdown(t *testing.T) {
	l := NewLoader()

	tests := []struct {
		name       string
		content    string
		wantName   string
		wantModel  string
		wantTools  []string
		wantInstr  string
		wantErr    bool
	}{
		{
			name: "valid agent",
			content: `---
name: test-agent
model: sonnet
tools: [Read, Write]
---
You are a test agent.`,
			wantName:  "test-agent",
			wantModel: "sonnet",
			wantTools: []string{"Read", "Write"},
			wantInstr: "You are a test agent.",
		},
		{
			name: "agent with all fields",
			content: `---
name: full-agent
namespace: polymarket
description: A full agent
model: opus
tools: [WebSearch, WebFetch, Read, Write]
role: Researcher
goal: Find information
backstory: Expert researcher
---
# Instructions

Do research.`,
			wantName:  "full-agent",
			wantModel: "opus",
			wantTools: []string{"WebSearch", "WebFetch", "Read", "Write"},
			wantInstr: "# Instructions\n\nDo research.",
		},
		{
			name:    "no frontmatter",
			content: "Just some text",
			wantErr: true,
		},
		{
			name:    "incomplete frontmatter",
			content: "---\nname: test\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := l.ParseAgentMarkdown([]byte(tt.content), "test.md")
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if agent.Name != tt.wantName {
				t.Errorf("name = %q, want %q", agent.Name, tt.wantName)
			}
			if agent.Model != tt.wantModel {
				t.Errorf("model = %q, want %q", agent.Model, tt.wantModel)
			}
			if len(agent.Tools) != len(tt.wantTools) {
				t.Errorf("tools = %v, want %v", agent.Tools, tt.wantTools)
			}
			if agent.Instructions != tt.wantInstr {
				t.Errorf("instructions = %q, want %q", agent.Instructions, tt.wantInstr)
			}
		})
	}
}

func TestAgentQualifiedName(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		want      string
	}{
		{name: "agent1", namespace: "", want: "agent1"},
		{name: "agent2", namespace: "polymarket", want: "polymarket/agent2"},
	}

	for _, tt := range tests {
		agent := &Agent{Name: tt.name, Namespace: tt.namespace}
		if got := agent.QualifiedName(); got != tt.want {
			t.Errorf("QualifiedName() = %q, want %q", got, tt.want)
		}
	}
}

func TestAgentMap(t *testing.T) {
	agents := []*Agent{
		{Name: "agent1"},
		{Name: "agent2", Namespace: "ns"},
	}

	m := AgentMap(agents)

	// Should have both by simple name
	if _, ok := m["agent1"]; !ok {
		t.Error("missing agent1")
	}
	if _, ok := m["agent2"]; !ok {
		t.Error("missing agent2 by simple name")
	}

	// Should have namespaced by qualified name
	if _, ok := m["ns/agent2"]; !ok {
		t.Error("missing ns/agent2")
	}
}

func TestLoadAgentsFromDir(t *testing.T) {
	l := NewLoader()

	// Test loading from the actual specs directory
	agents, err := l.LoadAgentsFromDir("../../agents/specs/agents")
	if err != nil {
		t.Fatalf("LoadAgentsFromDir failed: %v", err)
	}

	if len(agents) != 3 {
		t.Errorf("expected 3 agents, got %d", len(agents))
	}

	// Verify expected agents exist
	names := make(map[string]bool)
	for _, a := range agents {
		names[a.Name] = true
	}

	for _, expected := range []string{"market-analyst", "superforecaster", "trader"} {
		if !names[expected] {
			t.Errorf("missing expected agent: %s", expected)
		}
	}
}

func TestLoadTeam(t *testing.T) {
	l := NewLoader()

	team, err := l.LoadTeam("../../agents/specs/team.json")
	if err != nil {
		t.Fatalf("LoadTeam failed: %v", err)
	}

	if team.Name != "polymarket-trading-team" {
		t.Errorf("name = %q, want polymarket-trading-team", team.Name)
	}
	if team.Workflow.Type != "graph" {
		t.Errorf("workflow.type = %q, want graph", team.Workflow.Type)
	}
	if len(team.Workflow.Steps) != 3 {
		t.Errorf("workflow.steps = %d, want 3", len(team.Workflow.Steps))
	}
}

func TestLoadDeployment(t *testing.T) {
	l := NewLoader()

	deployment, err := l.LoadDeployment("../../agents/specs/deployment-go-server.json")
	if err != nil {
		t.Fatalf("LoadDeployment failed: %v", err)
	}

	if deployment.Platform != "go-server" {
		t.Errorf("platform = %q, want go-server", deployment.Platform)
	}
	if deployment.Team != "polymarket-trading-team" {
		t.Errorf("team = %q, want polymarket-trading-team", deployment.Team)
	}
}
