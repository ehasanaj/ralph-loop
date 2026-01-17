package plan

import "time"

// StepStatus represents the current state of a step
type StepStatus string

const (
	StatusPending   StepStatus = "pending"
	StatusCompleted StepStatus = "completed"
	StatusFailed    StepStatus = "failed"
)

// Step represents a single step in the plan
type Step struct {
	Number      int
	Description string
	Status      StepStatus
	LastRun     *time.Time
	Notes       string
}

// Plan represents the entire plan document
type Plan struct {
	ProjectName string
	Context     string // Project context/background info for the AI
	Steps       []Step
	RawContent  string // Original markdown content for preservation
}

// NextStep returns the first pending or failed step, or nil if all complete
func (p *Plan) NextStep() *Step {
	for i := range p.Steps {
		if p.Steps[i].Status == StatusPending || p.Steps[i].Status == StatusFailed {
			return &p.Steps[i]
		}
	}
	return nil
}

// IsComplete returns true if all steps are completed
func (p *Plan) IsComplete() bool {
	for _, step := range p.Steps {
		if step.Status != StatusCompleted {
			return false
		}
	}
	return len(p.Steps) > 0
}

// StepResult represents the outcome of running a step
type StepResult struct {
	Success bool
	Output  string
	Reason  string // Populated if failed
}
