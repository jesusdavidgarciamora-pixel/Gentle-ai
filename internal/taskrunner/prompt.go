package taskrunner

import (
	"fmt"
	"strings"
	"time"
)

// SystemPrompt is the base prompt for the agent loop.
const SystemPrompt = `You are an autonomous task execution agent. Your goal is to complete the given task WITHOUT asking the user for input.

RULES:
1. NEVER ask the user for clarification — figure it out autonomously
2. If a command fails, analyze the error and try a different approach
3. You can read files to understand the codebase, write files to create/modify code, and run shell commands
4. When you believe the task is complete, respond with {"action":"done",...}
5. If you cannot complete the task after reasonable effort, respond with {"action":"failed",...}
6. Always provide a clear reason for each action
7. Keep responses concise — focus on action, not explanation

AVAILABLE ACTIONS:
- {"action":"shell","command":"...","reason":"..."} — Execute a shell command
- {"action":"write_file","path":"...","content":"...","reason":"..."} — Create or overwrite a file
- {"action":"read_file","path":"...","reason":"..."} — Read a file's contents
- {"action":"edit_file","path":"...","old_text":"...","new_text":"...","reason":"..."} — Replace text in a file
- {"action":"done","summary":"...","reason":"..."} — Task completed successfully
- {"action":"failed","summary":"...","reason":"..."} — Task failed, explain why

RESPONSE FORMAT:
Respond with ONLY a JSON object. No markdown fences, no explanations outside the JSON. The JSON must be valid and parseable.

WORKING DIRECTORY: %s
CURRENT TIME: %s
`

// BuildTurnPrompt builds the complete prompt for a turn.
func BuildTurnPrompt(task string, history []StepRecord, workDir string, lessons []ErrorLesson) string {
	var sb strings.Builder

	// System prompt
	sb.WriteString(fmt.Sprintf(SystemPrompt, workDir, time.Now().Format(time.RFC3339)))
	sb.WriteString("\n")

	// Add lessons learned if available
	if len(lessons) > 0 {
		sb.WriteString(FormatLessonsForPrompt(lessons))
		sb.WriteString("\n")
	}

	// Task
	sb.WriteString("=== TASK ===\n")
	sb.WriteString(task)
	sb.WriteString("\n\n")

	// History
	if len(history) > 0 {
		sb.WriteString("=== HISTORY ===\n")
		sb.WriteString(fmt.Sprintf("You have taken %d steps so far.\n\n", len(history)))

		// Show last 10 steps to keep context manageable
		start := 0
		if len(history) > 10 {
			start = len(history) - 10
			sb.WriteString(fmt.Sprintf("(Showing last 10 of %d steps)\n\n", len(history)))
		}

		for i := start; i < len(history); i++ {
			step := history[i]
			sb.WriteString(fmt.Sprintf("--- Step %d ---\n", step.Iteration))
			sb.WriteString(fmt.Sprintf("Action: %s\n", step.Action.Type))

			if step.Action.Type == ActionShell {
				sb.WriteString(fmt.Sprintf("Command: %s\n", step.Action.Command))
			} else if step.Action.Type == ActionWriteFile {
				sb.WriteString(fmt.Sprintf("Wrote: %s (%d bytes)\n", step.Action.Path, len(step.Action.Content)))
			} else if step.Action.Type == ActionReadFile {
				sb.WriteString(fmt.Sprintf("Read: %s\n", step.Action.Path))
			} else if step.Action.Type == ActionEditFile {
				sb.WriteString(fmt.Sprintf("Edited: %s\n", step.Action.Path))
			}

			if step.Error != "" {
				sb.WriteString(fmt.Sprintf("Error: %s\n", step.Error))
			}

			// Truncate output for prompt
			output := TruncateOutput(step.Output, 2000)
			if output != "" {
				sb.WriteString(fmt.Sprintf("Output:\n%s\n", output))
			}

			sb.WriteString("\n")
		}
		sb.WriteString("=== END HISTORY ===\n\n")
	}

	// Next action request
	sb.WriteString("=== YOUR TURN ===\n")
	sb.WriteString("Based on the task and history, what is your next action?\n")
	sb.WriteString("Respond with ONLY a JSON object.\n")

	return sb.String()
}
