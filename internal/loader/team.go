// Package loader provides loading of multi-agent-spec definitions.
package loader

import (
	"encoding/json"
	"fmt"
	"os"
)

// Team represents a loaded team specification.
type Team struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description,omitempty"`
	Agents      []string `json:"agents"`
	Workflow    Workflow `json:"workflow"`
	Context     string   `json:"context,omitempty"`
}

// Workflow represents the team workflow configuration.
type Workflow struct {
	Type  string `json:"type"`
	Steps []Step `json:"steps"`
}

// Step represents a workflow step.
type Step struct {
	Name      string   `json:"name"`
	Agent     string   `json:"agent"`
	DependsOn []string `json:"depends_on,omitempty"`
	Inputs    []Port   `json:"inputs,omitempty"`
	Outputs   []Port   `json:"outputs,omitempty"`
}

// Port represents an input or output port.
type Port struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	From        string `json:"from,omitempty"`
	Default     any    `json:"default,omitempty"`
}

// LoadTeam loads a team specification from a JSON file.
func (l *Loader) LoadTeam(path string) (*Team, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading team file: %w", err)
	}

	var team Team
	if err := json.Unmarshal(data, &team); err != nil {
		return nil, fmt.Errorf("parsing team JSON: %w", err)
	}

	return &team, nil
}

// Deployment represents a deployment specification.
type Deployment struct {
	Platform string           `json:"platform"`
	Team     string           `json:"team"`
	Version  string           `json:"version"`
	Config   DeploymentConfig `json:"config"`
}

// DeploymentConfig holds platform-specific configuration.
type DeploymentConfig struct {
	// Claude Code specific
	AgentDir     string            `json:"agent_dir,omitempty"`
	SpecDir      string            `json:"spec_dir,omitempty"`
	ModelMapping map[string]string `json:"model_mapping,omitempty"`
	ToolMapping  map[string]string `json:"tool_mapping,omitempty"`

	// Go Server specific
	Server      *ServerConfig      `json:"server,omitempty"`
	Concurrency *ConcurrencyConfig `json:"concurrency,omitempty"`
	LLM         *LLMConfig         `json:"llm,omitempty"`
	Retrieval   *RetrievalConfig   `json:"retrieval,omitempty"`
	Polymarket  *PolymarketConfig  `json:"polymarket,omitempty"`
	Risk        *RiskConfig        `json:"risk,omitempty"`
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	Port           int    `json:"port"`
	MetricsPort    int    `json:"metrics_port,omitempty"`
	HealthEndpoint string `json:"health_endpoint,omitempty"`
}

// ConcurrencyConfig holds concurrency configuration.
type ConcurrencyConfig struct {
	Model               string `json:"model"`
	MaxConcurrentAgents int    `json:"max_concurrent_agents"`
	StepTimeoutSeconds  int    `json:"step_timeout_seconds,omitempty"`
}

// LLMConfig holds LLM provider configuration.
type LLMConfig struct {
	Provider     string            `json:"provider"`
	Config       json.RawMessage   `json:"config,omitempty"`
	ModelMapping map[string]string `json:"model_mapping,omitempty"`
}

// RetrievalConfig holds RAG retrieval configuration.
type RetrievalConfig struct {
	Provider string          `json:"provider"`
	Config   json.RawMessage `json:"config,omitempty"`
}

// PolymarketConfig holds Polymarket API configuration.
type PolymarketConfig struct {
	CLOBURL       string `json:"clob_url"`
	GammaURL      string `json:"gamma_url"`
	ChainID       int    `json:"chain_id"`
	PrivateKeyEnv string `json:"private_key_env"`
}

// RiskConfig holds risk management configuration.
type RiskConfig struct {
	MaxPositionPercent float64 `json:"max_position_percent"`
	MaxExposurePercent float64 `json:"max_exposure_percent"`
	MinEdgePercent     float64 `json:"min_edge_percent"`
	KellyMultiplier    float64 `json:"kelly_multiplier"`
}

// LoadDeployment loads a deployment specification from a JSON file.
func (l *Loader) LoadDeployment(path string) (*Deployment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading deployment file: %w", err)
	}

	var deployment Deployment
	if err := json.Unmarshal(data, &deployment); err != nil {
		return nil, fmt.Errorf("parsing deployment JSON: %w", err)
	}

	return &deployment, nil
}
