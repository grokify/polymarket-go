// Package loader provides loading of multi-agent-spec definitions.
package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Agent represents a loaded agent specification.
type Agent struct {
	Name         string   `yaml:"name"`
	Namespace    string   `yaml:"namespace,omitempty"`
	Description  string   `yaml:"description"`
	Model        string   `yaml:"model"`
	Tools        []string `yaml:"tools"`
	Role         string   `yaml:"role,omitempty"`
	Goal         string   `yaml:"goal,omitempty"`
	Backstory    string   `yaml:"backstory,omitempty"`
	Dependencies []string `yaml:"dependencies,omitempty"`
	Instructions string   `yaml:"-"` // Parsed from markdown body
}

// QualifiedName returns namespace/name or just name if no namespace.
func (a *Agent) QualifiedName() string {
	if a.Namespace != "" {
		return a.Namespace + "/" + a.Name
	}
	return a.Name
}

// Loader loads multi-agent-spec definitions.
type Loader struct{}

// NewLoader creates a new Loader.
func NewLoader() *Loader {
	return &Loader{}
}

// LoadAgent loads an agent from a markdown file.
func (l *Loader) LoadAgent(path string) (*Agent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading agent file: %w", err)
	}

	return l.ParseAgentMarkdown(data, path)
}

// ParseAgentMarkdown parses agent definition from markdown with YAML frontmatter.
func (l *Loader) ParseAgentMarkdown(data []byte, path string) (*Agent, error) {
	content := string(data)

	// Check for YAML frontmatter
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("agent file must start with YAML frontmatter (---)")
	}

	// Find end of frontmatter
	parts := strings.SplitN(content[3:], "---", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid YAML frontmatter format")
	}

	frontmatter := strings.TrimSpace(parts[0])
	body := strings.TrimSpace(parts[1])

	// Parse YAML frontmatter
	var agent Agent
	if err := yaml.Unmarshal([]byte(frontmatter), &agent); err != nil {
		return nil, fmt.Errorf("parsing YAML frontmatter: %w", err)
	}

	// Set instructions from markdown body
	agent.Instructions = body

	// Derive namespace from directory if not set
	if agent.Namespace == "" {
		dir := filepath.Dir(path)
		base := filepath.Base(dir)
		if base != "agents" && base != "." {
			agent.Namespace = base
		}
	}

	return &agent, nil
}

// LoadAgentsFromDir loads all agents from a directory.
func (l *Loader) LoadAgentsFromDir(dir string) ([]*Agent, error) {
	var agents []*Agent

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .md files
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		agent, err := l.LoadAgent(path)
		if err != nil {
			return fmt.Errorf("loading %s: %w", path, err)
		}

		agents = append(agents, agent)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return agents, nil
}

// AgentMap returns a map of agents keyed by qualified name.
func AgentMap(agents []*Agent) map[string]*Agent {
	m := make(map[string]*Agent)
	for _, agent := range agents {
		m[agent.QualifiedName()] = agent
		// Also add by simple name for convenience
		if agent.Namespace != "" {
			m[agent.Name] = agent
		}
	}
	return m
}
