package autonomous

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gentleman-programming/gentle-ai/internal/agentbuilder"
	"github.com/gentleman-programming/gentle-ai/internal/model"
)

// RunFromArgs parses CLI args and runs SDD autonomous mode.
func RunFromArgs(args []string, stdout io.Writer) error {
	config := RunConfig{
		StartPhase:  PhaseExplore,
		EndPhase:    PhaseArchive,
		MaxIter:     30,
		Timeout:     5 * time.Minute,
		AutoApprove: true, // For now, auto-approve
	}

	var changeName string
	var verbose bool

	// Parse args
	i := 0
	for i < len(args) {
		arg := args[i]

		switch {
		case arg == "--verbose" || arg == "-v":
			verbose = true
			i++
		case strings.HasPrefix(arg, "--start="):
			config.StartPhase = PhaseType(strings.TrimPrefix(arg, "--start="))
			i++
		case arg == "--start" && i+1 < len(args):
			config.StartPhase = PhaseType(args[i+1])
			i += 2
		case strings.HasPrefix(arg, "--end="):
			config.EndPhase = PhaseType(strings.TrimPrefix(arg, "--end="))
			i++
		case arg == "--end" && i+1 < len(args):
			config.EndPhase = PhaseType(args[i+1])
			i += 2
		case strings.HasPrefix(arg, "--engine="):
			config.Engine = model.AgentID(strings.TrimPrefix(arg, "--engine="))
			i++
		case arg == "--engine" && i+1 < len(args):
			config.Engine = model.AgentID(args[i+1])
			i += 2
		case arg == "--skip-verify":
			config.SkipVerify = true
			i++
		case strings.HasPrefix(arg, "-"):
			return fmt.Errorf("unknown flag: %s", arg)
		default:
			// First non-flag is change name
			if changeName == "" {
				changeName = arg
			} else {
				changeName += " " + arg
			}
			i++
		}
	}

	if strings.TrimSpace(changeName) == "" {
		return fmt.Errorf(`usage: gentle-ai sdd-autonomous [flags] "change name"

Flags:
  --verbose          Show detailed progress
  --start PHASE      Start from phase (explore, propose, spec, design, tasks, apply, verify, archive)
  --end PHASE        End at phase
  --engine ENGINE    Force engine (claude-code, opencode, gemini, codex)
  --skip-verify      Skip verification phase`)
	}

	config.ChangeName = changeName

	// Select engine
	var engine agentbuilder.GenerationEngine
	if config.Engine != "" {
		engine = agentbuilder.NewEngine(config.Engine)
		if engine == nil {
			return fmt.Errorf("unknown engine: %s", config.Engine)
		}
		if !engine.Available() {
			return fmt.Errorf("engine %s not available on PATH", config.Engine)
		}
	} else {
		// Auto-select
		for _, agent := range []model.AgentID{
			model.AgentClaudeCode,
			model.AgentOpenCode,
			model.AgentGeminiCLI,
			model.AgentCodex,
		} {
			engine = agentbuilder.NewEngine(agent)
			if engine != nil && engine.Available() {
				config.Engine = agent
				break
			}
		}
		if engine == nil {
			return fmt.Errorf("no AI engine found on PATH")
		}
	}

	if verbose {
		fmt.Fprintf(stdout, "SDD Autonomous Mode\n")
		fmt.Fprintf(stdout, "Change: %s\n", config.ChangeName)
		fmt.Fprintf(stdout, "Engine: %s\n", config.Engine)
		fmt.Fprintf(stdout, "Phases: %s → %s\n\n", config.StartPhase, config.EndPhase)
	}

	// Run orchestrator
	orch := NewOrchestrator(engine, verbose)
	ctx := context.Background()

	result, err := orch.Run(ctx, config)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("SDD run failed")
	}

	return nil
}

// DetectComplexity analyzes a task description to determine if it should use
// SDD autonomous mode or simple taskrunner.
func DetectComplexity(task string) (useSDD bool, reason string) {
	taskLower := strings.ToLower(task)

	// Keywords that indicate complexity requiring SDD
	complexKeywords := []string{
		"redesign", "refactor", "architecture", "migrat",
		"new feature", "implement", "create system",
		"add support", "integrate", "redesign",
		"breaking change", "deprecat",
	}

	for _, keyword := range complexKeywords {
		if strings.Contains(taskLower, keyword) {
			return true, fmt.Sprintf("detected complex keyword: %s", keyword)
		}
	}

	// Simple keywords for taskrunner
	simpleKeywords := []string{
		"fix typo", "fix bug", "add test", "update doc",
		"rename", "delete", "remove file", "create script",
		"simple", "quick", "minor",
	}

	for _, keyword := range simpleKeywords {
		if strings.Contains(taskLower, keyword) {
			return false, fmt.Sprintf("detected simple keyword: %s", keyword)
		}
	}

	// Default: if task is long and has multiple sentences, likely complex
	sentences := strings.Split(task, ".")
	if len(sentences) > 2 {
		return true, "multiple sentences detected"
	}

	// Medium length tasks default to taskrunner
	return false, "defaulting to simple taskrunner"
}
