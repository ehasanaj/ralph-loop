# ralph-loop

A meta-orchestrator for AI coding agents that solves the context-bloat problem by running agents in a loop with persistent state.

## The Problem

AI coding agents like Claude, OpenCode, and Codex are powerful but suffer from context bloat when working on multi-step implementations. As conversations grow longer, the agents lose focus, forget earlier decisions, and produce inconsistent results.

## The Solution

ralph-loop maintains state in a `plan.md` file (the source of truth) and runs one fresh AI session per step. After each step completes, it updates the plan and starts a new session for the next step. This keeps each AI session focused and context-free.

## How It Works

```
┌─────────────────────────────────────────────────────────┐
│                      plan.md                            │
│  ┌───────────────────────────────────────────────────┐  │
│  │ # Project: My App                                 │  │
│  │ ## Context                                        │  │
│  │ Background info for the AI...                     │  │
│  │ ## Plan                                           │  │
│  │ - [x] Step 1: Set up project structure            │  │
│  │ - [ ] Step 2: Implement user authentication  <────┼──┼── Current
│  │ - [ ] Step 3: Add API endpoints                   │  │
│  │ ## Notes                                          │  │
│  │ ### Step 1                                        │  │
│  │ **Status**: completed                             │  │
│  │ **Last Run**: 2026-01-17 10:30:00                 │  │
│  │ **Notes**: Created src/, tests/, docs/ dirs       │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                    ralph-loop                           │
│  1. Parse plan.md                                       │
│  2. Find next pending/failed step                       │
│  3. Build prompt with context + step                    │
│  4. Run AI agent (claude/opencode/codex)                │
│  5. Parse output for STEP_COMPLETE or STEP_FAILED       │
│  6. Update plan.md with results                         │
│  7. Repeat until all steps complete                     │
└─────────────────────────────────────────────────────────┘
```

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/eraldohasanaj/ralph-loop.git
cd ralph-loop

# Build
make build

# Or install to GOPATH/bin
make install
```

### Requirements

- Go 1.21 or later
- One of the supported AI agents installed:
  - [Claude CLI](https://github.com/anthropics/claude-code) (`claude`)
  - [OpenCode](https://github.com/opencode-ai/opencode) (`opencode`)
  - [OpenAI Codex CLI](https://github.com/openai/codex) (`codex`)

## Quick Start

### 1. Initialize a Plan

```bash
ralph-loop init
```

This creates a `plan.md` template:

```markdown
# Project: [Your Project Name]

## Context

Add background information about your project here.
Include key technologies, constraints, or any context the AI should know.

## Plan

- [ ] Step 1: Describe your first task here
- [ ] Step 2: Describe your second task here
- [ ] Step 3: Add more steps as needed

## Notes

### Step 1
**Status**: pending
**Last Run**: N/A
**Notes**: (none)
...
```

### 2. Edit Your Plan

Fill in your project details:

```markdown
# Project: My Web API

## Context

This is a Go REST API using the Gin framework.
Database: PostgreSQL with GORM.
Authentication: JWT tokens.
Follow existing patterns in the codebase.

## Plan

- [ ] Step 1: Create user model and migration
- [ ] Step 2: Implement user registration endpoint
- [ ] Step 3: Implement login endpoint with JWT
- [ ] Step 4: Add authentication middleware
- [ ] Step 5: Write integration tests
```

### 3. Run the Loop

```bash
# Using Claude (default)
ralph-loop run

# Using OpenCode
ralph-loop run --agent opencode

# Using Codex (requires OPENAI_API_KEY)
ralph-loop run --agent codex

# Specify a different plan file
ralph-loop run --plan my-feature.md
```

### 4. Check Status

```bash
ralph-loop status

# Output:
# Project: My Web API
#
# Context:
#   This is a Go REST API using the Gin framework.
#   Database: PostgreSQL with GORM.
#   ...
#
# Steps:
#   [x] Step 1: Create user model and migration
#   [x] Step 2: Implement user registration endpoint
#   [ ] Step 3: Implement login endpoint with JWT
#   [ ] Step 4: Add authentication middleware
#   [ ] Step 5: Write integration tests
#
# Summary: 2 completed, 0 failed, 3 pending
#
# Next step: Step 3 - Implement login endpoint with JWT
```

## Plan File Format

### Sections

| Section | Required | Description |
|---------|----------|-------------|
| `# Project: Name` | Yes | Project title displayed in status and prompts |
| `## Context` | No | Background information included in every step's prompt |
| `## Plan` | Yes | List of steps with checkboxes |
| `## Notes` | Auto | Automatically maintained by ralph-loop |

### Step Status Markers

| Marker | Status | Description |
|--------|--------|-------------|
| `[ ]` | Pending | Not yet started |
| `[x]` | Completed | Successfully finished |
| `[!]` | Failed | Failed, will be retried |

### Example with All Features

```markdown
# Project: E-commerce Platform

## Context

Tech stack:
- Frontend: React 18 with TypeScript
- Backend: Node.js with Express
- Database: MongoDB with Mongoose
- Testing: Jest + React Testing Library

Constraints:
- Must support mobile viewports
- All API responses under 200ms
- 80% test coverage minimum

## Plan

- [x] Step 1: Set up project monorepo structure
- [x] Step 2: Create product listing component
- [!] Step 3: Implement shopping cart logic
- [ ] Step 4: Add checkout flow
- [ ] Step 5: Integrate payment gateway

## Notes

### Step 1
**Status**: completed
**Last Run**: 2026-01-17 09:15:00
**Notes**: Created packages/frontend and packages/backend

### Step 2
**Status**: completed
**Last Run**: 2026-01-17 09:45:00
**Notes**: ProductList and ProductCard components created

### Step 3
**Status**: failed
**Last Run**: 2026-01-17 10:30:00
**Notes**: Failed: Redux store configuration error
```

## Commands

### `ralph-loop init`

Create a new plan template.

```bash
ralph-loop init                    # Creates plan.md
ralph-loop init -o feature.md      # Creates feature.md
```

### `ralph-loop run`

Execute the plan loop.

```bash
ralph-loop run                     # Run with claude (default)
ralph-loop run -a opencode         # Run with opencode
ralph-loop run -a codex            # Run with codex
ralph-loop run -p feature.md       # Use different plan file
```

**Flags:**
- `-a, --agent`: AI agent to use (`opencode`, `claude`, or `codex`). Default: `claude`
- `-p, --plan`: Path to plan file. Default: `plan.md`

### `ralph-loop status`

Display current plan status.

```bash
ralph-loop status                  # Show plan.md status
ralph-loop status -p feature.md   # Show feature.md status
```

## Supported Agents

### Claude (`claude`)

Uses the [Claude CLI](https://github.com/anthropics/claude-code) from Anthropic.

```bash
# Install Claude CLI first
npm install -g @anthropic-ai/claude-code

# Run with Claude
ralph-loop run --agent claude
```

### OpenCode (`opencode`)

Uses [OpenCode](https://github.com/opencode-ai/opencode).

```bash
# Install OpenCode first
go install github.com/opencode-ai/opencode@latest

# Run with OpenCode
ralph-loop run --agent opencode
```

### Codex (`codex`)

Uses [OpenAI Codex CLI](https://github.com/openai/codex).

```bash
# Install Codex CLI first
npm install -g @openai/codex

# Set API key
export OPENAI_API_KEY=your-key-here

# Run with Codex
ralph-loop run --agent codex
```

## How Agents Communicate Completion

ralph-loop expects agents to output specific markers when they finish:

### Success
```
STEP_COMPLETE
```

### Failure
```
STEP_FAILED: Brief description of what went wrong
```

The prompt sent to agents includes instructions to output these markers. If no marker is found, the step is treated as failed.

## Graceful Shutdown

Press `Ctrl+C` to stop the loop gracefully. ralph-loop will:

1. Cancel the current agent execution
2. Mark the current step as failed with "Interrupted by user"
3. Save the plan state
4. Exit cleanly

You can resume by running `ralph-loop run` again.

## Project Structure

```
ralph-loop/
├── cmd/
│   └── ralph-loop/
│       └── main.go           # CLI entry point
├── internal/
│   ├── agent/
│   │   ├── agent.go          # Agent interface and factory
│   │   ├── claude.go         # Claude CLI agent
│   │   ├── codex.go          # OpenAI Codex agent
│   │   └── opencode.go       # OpenCode agent
│   ├── loop/
│   │   └── runner.go         # Main orchestration loop
│   ├── plan/
│   │   ├── parser.go         # Plan file parser
│   │   ├── template.go       # Plan template generation
│   │   ├── types.go          # Plan/Step types
│   │   └── writer.go         # Plan file writer
│   └── prompt/
│       └── builder.go        # Prompt construction
├── Makefile
├── go.mod
└── README.md
```

## Development

```bash
# Build
make build

# Install locally
make install

# Run tests
make test

# Format code
make fmt

# Run go vet
make vet

# Run all checks
make check

# Build for all platforms
make build-all
```

## License

MIT License - see LICENSE file for details.
