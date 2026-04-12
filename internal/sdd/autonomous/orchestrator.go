package autonomous

import (
	"context"
	"fmt"
	"time"

	"github.com/gentleman-programming/gentle-ai/internal/agentbuilder"
	"github.com/gentleman-programming/gentle-ai/internal/model"
)

// Orchestrator runs the complete SDD workflow with autonomous phases.
type Orchestrator struct {
	phaseRunner *PhaseRunner
	verbose     bool
}

// NewOrchestrator creates a new autonomous SDD orchestrator.
func NewOrchestrator(engine agentbuilder.GenerationEngine, verbose bool) *Orchestrator {
	return &Orchestrator{
		phaseRunner: NewPhaseRunner(engine),
		verbose:     verbose,
	}
}

// RunConfig configures a complete SDD run.
type RunConfig struct {
	ChangeName  string
	StartPhase  PhaseType      // Where to start (default: explore)
	EndPhase    PhaseType      // Where to end (default: archive)
	SkipVerify  bool           // Skip verification phase
	AutoApprove bool           // Auto-approve each phase without user confirmation
	MaxIter     int            // Max iterations per phase
	Timeout     time.Duration  // Timeout per phase
	Engine      model.AgentID  // Force specific engine
}

// RunResult contains the results of all phases.
type RunResult struct {
	ChangeName string
	Phases     []*PhaseResult
	StartTime  time.Time
	EndTime    time.Time
	Success    bool
}

// Run executes the complete SDD workflow autonomously.
func (o *Orchestrator) Run(ctx context.Context, config RunConfig) (*RunResult, error) {
	startTime := time.Now()
	result := &RunResult{
		ChangeName: config.ChangeName,
		Phases:     make([]*PhaseResult, 0),
		StartTime:  startTime,
	}

	// Determine phase order
	phases := o.determinePhaseOrder(config.StartPhase, config.EndPhase, config.SkipVerify)

	// Context accumulates outputs from previous phases
	accumulatedContext := fmt.Sprintf("Change: %s\n", config.ChangeName)

	for _, phase := range phases {
		if o.verbose {
			fmt.Printf("\n=== Starting Phase: %s ===\n", phase)
		}

		phaseConfig := PhaseConfig{
			Phase:      phase,
			ChangeName: config.ChangeName,
			Context:    accumulatedContext,
			MaxIter:    config.MaxIter,
			Timeout:    config.Timeout,
			Engine:     string(config.Engine),
			Verbose:    o.verbose,
		}

		phaseResult, err := o.phaseRunner.Run(ctx, phaseConfig)
		if err != nil {
			result.EndTime = time.Now()
			result.Success = false
			return result, fmt.Errorf("phase %s failed: %w", phase, err)
		}

		result.Phases = append(result.Phases, phaseResult)

		if !phaseResult.Success {
			result.EndTime = time.Now()
			result.Success = false
			if o.verbose {
				fmt.Printf("\n=== Phase %s FAILED ===\n", phase)
			}
			return result, fmt.Errorf("phase %s did not complete successfully", phase)
		}

		if o.verbose {
			fmt.Printf("\n=== Phase %s COMPLETED (%d iterations, %s) ===\n",
				phase, phaseResult.Iterations, phaseResult.Duration.Round(time.Second))
		}

		// Accumulate context for next phase
		accumulatedContext += fmt.Sprintf("\n\n=== %s Phase Output ===\n%s",
			phase, phaseResult.Content)

		// If not auto-approve, we would ask user here
		// For now, continue automatically
		if !config.AutoApprove && !o.autoApprove(phase, phaseResult) {
			result.EndTime = time.Now()
			result.Success = false
			return result, fmt.Errorf("phase %s was not approved by user", phase)
		}
	}

	result.EndTime = time.Now()
	result.Success = true

	if o.verbose {
		o.printSummary(result)
	}

	return result, nil
}

// determinePhaseOrder returns the phases to run based on start/end.
func (o *Orchestrator) determinePhaseOrder(start, end PhaseType, skipVerify bool) []PhaseType {
	allPhases := []PhaseType{
		PhaseExplore,
		PhasePropose,
		PhaseSpec,
		PhaseDesign,
		PhaseTasks,
		PhaseApply,
		PhaseVerify,
		PhaseArchive,
	}

	// Find start index
	startIdx := 0
	for i, p := range allPhases {
		if p == start {
			startIdx = i
			break
		}
	}

	// Find end index
	endIdx := len(allPhases) - 1
	for i, p := range allPhases {
		if p == end {
			endIdx = i
			break
		}
	}

	// Filter phases
	phases := make([]PhaseType, 0)
	for i := startIdx; i <= endIdx && i < len(allPhases); i++ {
		if allPhases[i] == PhaseVerify && skipVerify {
			continue
		}
		phases = append(phases, allPhases[i])
	}

	return phases
}

// autoApprove determines if a phase result should be auto-approved.
// This is a placeholder - in a real implementation, this would ask the user.
func (o *Orchestrator) autoApprove(phase PhaseType, result *PhaseResult) bool {
	// For now, auto-approve everything
	// Future: could have criteria like "if iterations > 10, ask user"
	return true
}

// printSummary prints a summary of the run.
func (o *Orchestrator) printSummary(result *RunResult) {
	fmt.Println("\n============================================================")
	fmt.Println("SDD AUTONOMOUS RUN COMPLETE")
	fmt.Println("============================================================")
	fmt.Printf("Change: %s\n", result.ChangeName)
	fmt.Printf("Duration: %s\n", result.EndTime.Sub(result.StartTime).Round(time.Second))
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Println("\nPhases:")
	for _, p := range result.Phases {
		status := "✓"
		if !p.Success {
			status = "✗"
		}
		fmt.Printf("  %s %s (%d iter, %s)\n",
			status, p.Phase, p.Iterations, p.Duration.Round(time.Second))
	}
	fmt.Println("============================================================")
}

// RunSinglePhase runs just one phase (useful for resuming).
func (o *Orchestrator) RunSinglePhase(ctx context.Context, config PhaseConfig) (*PhaseResult, error) {
	return o.phaseRunner.Run(ctx, config)
}
