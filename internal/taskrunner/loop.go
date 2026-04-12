package taskrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gentleman-programming/gentle-ai/internal/agentbuilder"
)

// Loop runs the agentic loop until completion.
type Loop struct {
	Config       RunConfig
	Executor     *Executor
	Engine       agentbuilder.GenerationEngine
	History      []StepRecord
	LessonStore  *LessonStore
}

// NewLoop creates a new agent loop.
func NewLoop(config RunConfig, engine agentbuilder.GenerationEngine) *Loop {
	return &Loop{
		Config:      config,
		Executor:    NewExecutor(config.WorkDir, config.Timeout),
		Engine:      engine,
		History:     make([]StepRecord, 0),
		LessonStore: DefaultLessonStore(),
	}
}

// Run executes the loop until done, failed, or max iterations.
func (l *Loop) Run(ctx context.Context) (*Report, error) {
	startTime := time.Now()

	if err := l.Config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if l.Engine == nil {
		return nil, fmt.Errorf("no generation engine available")
	}

	// Load relevant lessons from previous runs
	var lessons []ErrorLesson
	if l.LessonStore != nil {
		var err error
		lessons, err = l.LessonStore.FindRelevantLessons(l.Config.Task, l.Config.WorkDir)
		if err != nil && l.Config.Verbose {
			fmt.Printf("[Warning] Failed to load lessons: %v\n", err)
		}
		if len(lessons) > 0 && l.Config.Verbose {
			fmt.Printf("[Info] Loaded %d lessons from previous runs\n", len(lessons))
		}
	}

	for i := 0; i < l.Config.MaxIter; i++ {
		stepStart := time.Now()

		// Build prompt with lessons
		prompt := BuildTurnPrompt(l.Config.Task, l.History, l.Config.WorkDir, lessons)

		if l.Config.Verbose {
			fmt.Printf("\n[Iteration %d] Sending prompt to %s...\n", i+1, l.Engine.Agent())
		}

		// Generate action
		rawResponse, err := l.Engine.Generate(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("generation failed: %w", err)
		}

		// Parse action
		action, err := ParseAction(rawResponse)
		if err != nil {
			// Record parse error as a failed step
			step := StepRecord{
				Iteration: i + 1,
				Timestamp: time.Now(),
				Prompt:    prompt,
				Action:    Action{Type: ActionFailed, Reason: "parse error"},
				Output:    "",
				Error:     fmt.Sprintf("parse action: %v\nRaw response:\n%s", err, rawResponse),
				Duration:  time.Since(stepStart),
			}
			l.History = append(l.History, step)

			// Try to recover by continuing
			continue
		}

		if l.Config.Verbose {
			fmt.Printf("[Iteration %d] Action: %s\n", i+1, action.Type)
			if action.Reason != "" {
				fmt.Printf("  Reason: %s\n", action.Reason)
			}
		}

		// Execute action
		var output string
		var execErr error

		if action.Type == ActionDone || action.Type == ActionFailed {
			// Terminal actions - save lessons before returning
			step := StepRecord{
				Iteration: i + 1,
				Timestamp: time.Now(),
				Prompt:    prompt,
				Action:    action,
				Output:    action.Summary,
				Duration:  time.Since(stepStart),
			}
			l.History = append(l.History, step)

			status := "success"
			if action.Type == ActionFailed {
				status = "failed"
			}

			report := &Report{
				Task:        l.Config.Task,
				Status:      status,
				Iterations:  i + 1,
				Duration:    time.Since(startTime),
				Steps:       l.History,
				FinalOutput: action.Summary,
				EngineUsed:  l.Engine.Agent(),
				WorkDir:     l.Config.WorkDir,
			}

			// Extract and save lessons from this run
			if l.LessonStore != nil {
				newLessons := ExtractLessons(report)
				if len(newLessons) > 0 {
					if err := l.LessonStore.SaveLessons(newLessons); err != nil && l.Config.Verbose {
						fmt.Printf("[Warning] Failed to save lessons: %v\n", err)
					} else if l.Config.Verbose && len(newLessons) > 0 {
						fmt.Printf("[Info] Saved %d new lessons\n", len(newLessons))
					}
				}
			}

			return report, nil
		}

		// Execute non-terminal action
		output, execErr = l.Executor.Execute(ctx, action)

		step := StepRecord{
			Iteration: i + 1,
			Timestamp: time.Now(),
			Prompt:    prompt,
			Action:    action,
			Output:    output,
			Duration:  time.Since(stepStart),
		}

		if execErr != nil {
			step.Error = execErr.Error()
			if l.Config.Verbose {
				fmt.Printf("[Iteration %d] Error: %v\n", i+1, execErr)
			}
		} else if l.Config.Verbose {
			truncated := TruncateOutput(output, 500)
			if truncated != "" {
				fmt.Printf("[Iteration %d] Output:\n%s\n", i+1, truncated)
			}
		}

		l.History = append(l.History, step)
	}

	// Max iterations reached
	report := &Report{
		Task:        l.Config.Task,
		Status:      "failed",
		Iterations:  l.Config.MaxIter,
		Duration:    time.Since(startTime),
		Steps:       l.History,
		FinalOutput: fmt.Sprintf("Max iterations (%d) reached without completion", l.Config.MaxIter),
		EngineUsed:  l.Engine.Agent(),
		WorkDir:     l.Config.WorkDir,
	}

	// Extract and save lessons even on max iterations
	if l.LessonStore != nil {
		newLessons := ExtractLessons(report)
		if len(newLessons) > 0 {
			if err := l.LessonStore.SaveLessons(newLessons); err != nil && l.Config.Verbose {
				fmt.Printf("[Warning] Failed to save lessons: %v\n", err)
			}
		}
	}

	return report, nil
}

// ParseAction extracts a JSON action from the raw response.
func ParseAction(raw string) (Action, error) {
	// Try to extract JSON from markdown fences
	jsonStr := extractJSON(raw)
	if jsonStr == "" {
		jsonStr = raw
	}

	// Clean up common issues
	jsonStr = strings.TrimSpace(jsonStr)

	var action Action
	if err := json.Unmarshal([]byte(jsonStr), &action); err != nil {
		return Action{}, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate action type
	switch action.Type {
	case ActionShell, ActionWriteFile, ActionReadFile, ActionEditFile, ActionDone, ActionFailed:
		// Valid
	default:
		return Action{}, fmt.Errorf("invalid action type: %s", action.Type)
	}

	return action, nil
}

// extractJSON tries to extract JSON from markdown fences or raw text.
func extractJSON(raw string) string {
	// Try markdown code fences
	fencePattern := regexp.MustCompile("```(?:json)?\\s*\\n?({[\\s\\S]*?})\\s*```")
	matches := fencePattern.FindStringSubmatch(raw)
	if len(matches) > 1 {
		return matches[1]
	}

	// Try to find JSON object boundaries
	startIdx := strings.Index(raw, "{")
	endIdx := strings.LastIndex(raw, "}")

	if startIdx >= 0 && endIdx > startIdx {
		return raw[startIdx : endIdx+1]
	}

	return ""
}
