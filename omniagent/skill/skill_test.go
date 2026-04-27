package skill

import (
	"testing"

	"github.com/plexusone/omniagent/skills/compiled"
	"github.com/plexusone/omniskill/skill"
)

func TestSkillImplementsInterface(t *testing.T) {
	var _ compiled.Skill = (*Skill)(nil)
}

func TestSkillName(t *testing.T) {
	s := New(Config{})
	if s.Name() != "predictions" {
		t.Errorf("expected name 'predictions', got '%s'", s.Name())
	}
}

func TestSkillDescription(t *testing.T) {
	s := New(Config{})
	if s.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestSkillTools(t *testing.T) {
	s := New(Config{})
	tools := s.Tools()

	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}

	// Check tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name()] = true
	}

	if !toolNames["get_markets"] {
		t.Error("expected get_markets tool")
	}
	if !toolNames["get_orderbook"] {
		t.Error("expected get_orderbook tool")
	}
}

func TestGetMarketsParameters(t *testing.T) {
	s := New(Config{})
	tools := s.Tools()

	var marketsTool skill.Tool
	for _, tool := range tools {
		if tool.Name() == "get_markets" {
			marketsTool = tool
			break
		}
	}

	if marketsTool == nil {
		t.Fatal("get_markets tool not found")
	}

	params := marketsTool.Parameters()
	if params == nil {
		t.Fatal("expected non-nil parameters")
	}

	// Check expected parameters exist
	expectedParams := []string{"min_liquidity", "category", "limit", "text_query"}
	for _, param := range expectedParams {
		if _, ok := params[param]; !ok {
			t.Errorf("expected parameter '%s' not found", param)
		}
	}
}

func TestGetOrderBookParameters(t *testing.T) {
	s := New(Config{})
	tools := s.Tools()

	var orderbookTool skill.Tool
	for _, tool := range tools {
		if tool.Name() == "get_orderbook" {
			orderbookTool = tool
			break
		}
	}

	if orderbookTool == nil {
		t.Fatal("get_orderbook tool not found")
	}

	params := orderbookTool.Parameters()
	if params == nil {
		t.Fatal("expected non-nil parameters")
	}

	// Check token_id parameter exists
	if _, ok := params["token_id"]; !ok {
		t.Error("expected token_id parameter")
	}
}
