package plan

import "time"

// StepStatus represents the current state of a step
type StepStatus string

const (
	StatusPending   StepStatus = "pending"
	StatusCompleted StepStatus = "completed"
	StatusFailed    StepStatus = "failed"
	StatusSkipped   StepStatus = "skipped" // For steps that exceeded max retries
)

// Step represents a single step in the plan
type Step struct {
	Number      int
	Description string
	Status      StepStatus
	LastRun     *time.Time
	Notes       string
	RetryCount  int // Track retry attempts
}

// Plan represents the entire plan document
type Plan struct {
	ProjectName string
	Context     string // Project context/background info for the AI
	Steps       []Step
	RawContent  string // Original markdown content for preservation
}

// NextStep returns the first pending or failed step, or nil if all complete/skipped
func (p *Plan) NextStep() *Step {
	for i := range p.Steps {
		s := p.Steps[i].Status
		if s == StatusPending || s == StatusFailed {
			return &p.Steps[i]
		}
		// Skip StatusCompleted and StatusSkipped
	}
	return nil
}

// IsComplete returns true if all steps are completed or skipped
func (p *Plan) IsComplete() bool {
	for _, step := range p.Steps {
		if step.Status != StatusCompleted && step.Status != StatusSkipped {
			return false
		}
	}
	return len(p.Steps) > 0
}

// StepResult represents the outcome of running a step
type StepResult struct {
	Success    bool
	Output     string
	Reason     string     // Populated if failed
	Status     StepStatus // Optional explicit status (use for skipped)
	RetryCount int        // Current retry count for the step
}
