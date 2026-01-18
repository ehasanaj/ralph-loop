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

// Options configures agent behavior
type Options struct {
	// Model specifies the model to use in format "provider/model"
	// e.g., "openai/gpt-5.2", "anthropic/claude-sonnet-4-20250514"
	Model string
}

// New creates a new agent of the specified type with options
func New(agentType AgentType, opts Options) (Agent, error) {
	switch agentType {
	case AgentTypeOpencode:
		return NewOpencodeAgent(opts), nil
	case AgentTypeClaude:
		return NewClaudeAgent(opts), nil
	case AgentTypeCodex:
		return NewCodexAgent(opts), nil
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
