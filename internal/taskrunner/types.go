package taskrunner

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gentleman-programming/gentle-ai/internal/model"
)

// ActionType represents the type of action the agent can take.
type ActionType string

const (
	ActionShell     ActionType = "shell"
	ActionWriteFile ActionType = "write_file"
	ActionReadFile  ActionType = "read_file"
	ActionEditFile  ActionType = "edit_file"
	ActionDone      ActionType = "done"
	ActionFailed    ActionType = "failed"
)

// Action represents a single action from the agent.
type Action struct {
	Type    ActionType `json:"action"`
	Command string     `json:"command,omitempty"`
	Path    string     `json:"path,omitempty"`
	Content string     `json:"content,omitempty"`
	OldText string     `json:"old_text,omitempty"`
	NewText string     `json:"new_text,omitempty"`
	Reason  string     `json:"reason"`
	Summary string     `json:"summary,omitempty"`
}

// StepRecord tracks a single iteration of the loop.
type StepRecord struct {
	Iteration int           `json:"iteration"`
	Timestamp time.Time     `json:"timestamp"`
	Prompt    string        `json:"prompt"`
	Action    Action        `json:"action"`
	Output    string        `json:"output"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
}

// Report is the final output of a task run.
type Report struct {
	Task        string        `json:"task"`
	Status      string        `json:"status"` // "success" | "failed"
	Iterations  int           `json:"iterations"`
	Duration    time.Duration `json:"duration"`
	Steps       []StepRecord  `json:"steps"`
	FinalOutput string        `json:"final_output"`
	EngineUsed  model.AgentID `json:"engine_used"`
	WorkDir     string        `json:"work_dir"`
}

// RunConfig configures a task run.
type RunConfig struct {
	Task         string
	WorkDir      string
	MaxIter      int
	Verbose      bool
	Engine       model.AgentID // empty = auto-select
	Timeout      time.Duration // per-step timeout
	SaveToEngram bool
	ProjectName  string // for engram grouping
}

// DefaultRunConfig returns sensible defaults.
func DefaultRunConfig(task string) RunConfig {
	return RunConfig{
		Task:         task,
		WorkDir:      ".",
		MaxIter:      30,
		Verbose:      false,
		Timeout:      2 * time.Minute,
		SaveToEngram: false,
	}
}

// Validate checks the configuration.
func (c RunConfig) Validate() error {
	if strings.TrimSpace(c.Task) == "" {
		return fmt.Errorf("task cannot be empty")
	}
	if c.MaxIter <= 0 {
		c.MaxIter = 30
	}
	if c.MaxIter > 100 {
		return fmt.Errorf("max iterations too high (max 100)")
	}
	if c.Timeout <= 0 {
		c.Timeout = 2 * time.Minute
	}
	return nil
}

// JSON returns the JSON representation of an action.
func (a Action) JSON() string {
	b, _ := json.MarshalIndent(a, "", "  ")
	return string(b)
}
