package autonomous

import (
	"context"
	"fmt"
	"time"

	"github.com/gentleman-programming/gentle-ai/internal/agentbuilder"
	"github.com/gentleman-programming/gentle-ai/internal/model"
	"github.com/gentleman-programming/gentle-ai/internal/taskrunner"
)

// PhaseType represents an SDD phase that can run autonomously.
type PhaseType string

const (
	PhaseExplore  PhaseType = "explore"
	PhasePropose  PhaseType = "propose"
	PhaseSpec     PhaseType = "spec"
	PhaseDesign   PhaseType = "design"
	PhaseTasks    PhaseType = "tasks"
	PhaseApply    PhaseType = "apply"
	PhaseVerify   PhaseType = "verify"
	PhaseArchive  PhaseType = "archive"
)

// PhaseConfig configures a single SDD phase run.
type PhaseConfig struct {
	Phase       PhaseType
	ChangeName  string
	Context     string           // Previous phase outputs
	MaxIter     int
	Timeout     time.Duration
	Engine      string           // Force specific engine
	Verbose     bool
}

// PhaseResult is the output of running a phase.
type PhaseResult struct {
	Phase       PhaseType
	Content     string
	Artifacts   map[string]string
	Duration    time.Duration
	Iterations  int
	Success     bool
	Error       error
}

// PhaseRunner runs a single SDD phase autonomously.
type PhaseRunner struct {
	Engine agentbuilder.GenerationEngine
}

// NewPhaseRunner creates a new phase runner.
func NewPhaseRunner(engine agentbuilder.GenerationEngine) *PhaseRunner {
	return &PhaseRunner{Engine: engine}
}

// Run executes a single phase autonomously.
func (r *PhaseRunner) Run(ctx context.Context, config PhaseConfig) (*PhaseResult, error) {
	if r.Engine == nil {
		return nil, fmt.Errorf("no generation engine available")
	}

	// Build phase-specific task
	task := buildPhaseTask(config)

	// Create run config for taskrunner
	runConfig := taskrunner.RunConfig{
		Task:    task,
		MaxIter: config.MaxIter,
		Timeout: config.Timeout,
		Verbose: config.Verbose,
	}

	if config.Engine != "" {
		runConfig.Engine = model.AgentID(config.Engine)
	}

	// Run the phase using taskrunner loop
	loop := taskrunner.NewLoop(runConfig, r.Engine)
	report, err := loop.Run(ctx)
	if err != nil {
		return &PhaseResult{
			Phase:    config.Phase,
			Success:  false,
			Error:    err,
			Duration: report.Duration,
		}, nil
	}

	result := &PhaseResult{
		Phase:      config.Phase,
		Content:    report.FinalOutput,
		Duration:   report.Duration,
		Iterations: report.Iterations,
		Success:    report.Status == "success",
		Artifacts:  extractArtifacts(report),
	}

	if !result.Success {
		result.Error = fmt.Errorf("phase %s failed: %s", config.Phase, report.FinalOutput)
	}

	return result, nil
}

// buildPhaseTask creates the task description for a specific phase.
func buildPhaseTask(config PhaseConfig) string {
	baseTask := fmt.Sprintf("Complete the SDD %s phase for change: %s\n\n", config.Phase, config.ChangeName)

	switch config.Phase {
	case PhaseExplore:
		baseTask += `Your task is to EXPLORE and understand the codebase related to this change.

You should:
1. Read relevant files to understand current implementation
2. Identify affected components and dependencies
3. Research possible approaches and their trade-offs
4. Document your findings

Output format: Write an exploration report that includes:
- Current state analysis
- Identified constraints and requirements
- Possible approaches with pros/cons
- Recommended approach with justification

Files to create:
- openspec/changes/{change-name}/explore.md (or save to engram with topic_key "sdd/{change-name}/explore")`

	case PhasePropose:
		baseTask += fmt.Sprintf(`Your task is to CREATE A PROPOSAL for this change.

Previous exploration context:
%s

You should:
1. Define the intent and scope of the change
2. Choose an approach based on the exploration
3. Document the high-level solution
4. Identify risks and mitigation strategies

Output format: Write a proposal that includes:
- Intent: What problem are we solving?
- Scope: What's in and out of scope?
- Approach: High-level solution description
- Risks: What could go wrong?

Files to create:
- openspec/changes/{change-name}/proposal.md (or save to engram with topic_key "sdd/{change-name}/proposal")`, config.Context)

	case PhaseSpec:
		baseTask += fmt.Sprintf(`Your task is to WRITE SPECIFICATIONS for this change.

Previous proposal context:
%s

You should:
1. Define detailed requirements
2. Write acceptance criteria (Given/When/Then)
3. Document scenarios (happy path, edge cases, error cases)
4. Specify interfaces and contracts

Output format: Write specifications that include:
- Requirements (functional and non-functional)
- Scenarios with acceptance criteria
- Interface definitions
- Validation rules

Files to create:
- openspec/changes/{change-name}/specs/*/spec.md (or save to engram with topic_key "sdd/{change-name}/spec")`, config.Context)

	case PhaseDesign:
		baseTask += fmt.Sprintf(`Your task is to CREATE TECHNICAL DESIGN for this change.

Previous proposal and spec context:
%s

You should:
1. Design the architecture and components
2. Define data models and interfaces
3. Document algorithms and logic
4. Plan the implementation approach

Output format: Write a design document that includes:
- Architecture decisions
- Component diagrams (as text)
- Data models
- API contracts
- Implementation plan

Files to create:
- openspec/changes/{change-name}/design.md (or save to engram with topic_key "sdd/{change-name}/design")`, config.Context)

	case PhaseTasks:
		baseTask += fmt.Sprintf(`Your task is to BREAK DOWN IMPLEMENTATION TASKS for this change.

Previous spec and design context:
%s

You should:
1. Create a checklist of implementation tasks
2. Define task dependencies
3. Estimate complexity
4. Assign logical order

Output format: Write a tasks document that includes:
- Task checklist (numbered)
- Dependencies between tasks
- File changes needed
- Testing requirements

Files to create:
- openspec/changes/{change-name}/tasks.md (or save to engram with topic_key "sdd/{change-name}/tasks")`, config.Context)

	case PhaseApply:
		baseTask += fmt.Sprintf(`Your task is to IMPLEMENT the change.

Previous tasks context:
%s

You should:
1. Implement each task from the checklist
2. Write code following the design
3. Add tests as specified
4. Mark tasks as complete

Output format:
- Implement all code changes
- Create/modify files as needed
- Run tests to verify
- Update task progress

This is the implementation phase - write actual code!`, config.Context)

	case PhaseVerify:
		baseTask += fmt.Sprintf(`Your task is to VERIFY the implementation.

Previous spec and implementation context:
%s

You should:
1. Check implementation against specifications
2. Verify acceptance criteria are met
3. Run tests and validate
4. Document any discrepancies

Output format: Write a verification report that includes:
- CRITICAL issues (must fix)
- WARNING issues (should fix)
- SUGGESTION issues (nice to have)
- Overall pass/fail status

Files to create:
- openspec/changes/{change-name}/verify-report.md (or save to engram with topic_key "sdd/{change-name}/verify-report")`, config.Context)

	case PhaseArchive:
		baseTask += fmt.Sprintf(`Your task is to ARCHIVE the completed change.

Previous context:
%s

You should:
1. Sync delta specs to main specs
2. Archive the change documentation
3. Update any indexes
4. Clean up temporary files

Output format: Write an archive report that includes:
- What was archived
- Where to find the final artifacts
- Any follow-up actions needed

Files to create:
- Archive report in openspec/changes/archive/`, config.Context)
	}

	baseTask += `

IMPORTANT RULES:
- Work autonomously - figure things out without asking
- If you encounter errors, analyze and retry with a different approach
- Create all necessary files
- Be thorough but concise
- Focus on quality over speed`

	return baseTask
}

// extractArtifacts extracts any file artifacts from the report.
func extractArtifacts(report *taskrunner.Report) map[string]string {
	// This would parse the report output to find created files
	// For now, return empty - can be enhanced later
	return make(map[string]string)
}
