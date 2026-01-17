package agent

import (
	"context"
	"fmt"
	"io"
)

// Agent represents an AI coding agent that can execute prompts
type Agent interface {
	// Name returns the agent's name
	Name() string

	// Run executes the agent with the given prompt
	// It streams output to the writer while collecting for parsing
	// Returns the full output when complete
	Run(ctx context.Context, prompt string, output io.Writer) (string, error)
}

// AgentType represents the type of agent to use
type AgentType string

const (
	AgentTypeOpencode AgentType = "opencode"
	AgentTypeClaude   AgentType = "claude"
	AgentTypeCodex    AgentType = "codex"
)

// New creates a new agent of the specified type
func New(agentType AgentType) (Agent, error) {
	switch agentType {
	case AgentTypeOpencode:
		return NewOpencodeAgent(), nil
	case AgentTypeClaude:
		return NewClaudeAgent(), nil
	case AgentTypeCodex:
		return NewCodexAgent(), nil
	default:
		return nil, fmt.Errorf("unknown agent type: %s", agentType)
	}
}

// ParseAgentType parses a string into an AgentType
func ParseAgentType(s string) (AgentType, error) {
	switch s {
	case "opencode":
		return AgentTypeOpencode, nil
	case "claude":
		return AgentTypeClaude, nil
	case "codex":
		return AgentTypeCodex, nil
	default:
		return "", fmt.Errorf("unknown agent type: %s (valid: opencode, claude, codex)", s)
	}
}
