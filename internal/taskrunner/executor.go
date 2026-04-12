package taskrunner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Executor runs actions on the real system.
type Executor struct {
	WorkDir string
	Timeout time.Duration
}

// NewExecutor creates a new executor.
func NewExecutor(workDir string, timeout time.Duration) *Executor {
	return &Executor{
		WorkDir: workDir,
		Timeout: timeout,
	}
}

// Execute runs an action and returns the output.
func (e *Executor) Execute(ctx context.Context, action Action) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, e.Timeout)
	defer cancel()

	switch action.Type {
	case ActionShell:
		return e.executeShell(ctx, action.Command)
	case ActionWriteFile:
		return e.executeWriteFile(action.Path, action.Content)
	case ActionReadFile:
		return e.executeReadFile(action.Path)
	case ActionEditFile:
		return e.executeEditFile(action.Path, action.OldText, action.NewText)
	case ActionDone, ActionFailed:
		return "", nil
	default:
		return "", fmt.Errorf("unknown action type: %s", action.Type)
	}
}

func (e *Executor) executeShell(ctx context.Context, command string) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = e.WorkDir
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	result := string(output)

	if ctx.Err() == context.DeadlineExceeded {
		return result, fmt.Errorf("command timed out after %v", e.Timeout)
	}

	if err != nil {
		return result, fmt.Errorf("exit error: %w", err)
	}

	return result, nil
}

func (e *Executor) executeWriteFile(path, content string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}

	// Resolve relative paths against workdir
	if !filepath.IsAbs(path) {
		path = filepath.Join(e.WorkDir, path)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return fmt.Sprintf("File written: %s (%d bytes)", path, len(content)), nil
}

func (e *Executor) executeReadFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}

	// Resolve relative paths against workdir
	if !filepath.IsAbs(path) {
		path = filepath.Join(e.WorkDir, path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	return string(content), nil
}

func (e *Executor) executeEditFile(path, oldText, newText string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	if oldText == "" {
		return "", fmt.Errorf("old_text cannot be empty for edit_file")
	}

	// Resolve relative paths against workdir
	if !filepath.IsAbs(path) {
		path = filepath.Join(e.WorkDir, path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	oldContent := string(content)
	if !strings.Contains(oldContent, oldText) {
		return "", fmt.Errorf("old_text not found in file")
	}

	newContent := strings.ReplaceAll(oldContent, oldText, newText)

	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return fmt.Sprintf("File edited: %s", path), nil
}

// FormatActionResult formats the result of an action for the history.
func FormatActionResult(action Action, output string, err error) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("=== Action: %s ===\n", action.Type))
	sb.WriteString(fmt.Sprintf("Reason: %s\n", action.Reason))

	switch action.Type {
	case ActionShell:
		sb.WriteString(fmt.Sprintf("Command: %s\n", action.Command))
	case ActionWriteFile:
		sb.WriteString(fmt.Sprintf("Path: %s\n", action.Path))
		sb.WriteString(fmt.Sprintf("Content length: %d bytes\n", len(action.Content)))
	case ActionReadFile:
		sb.WriteString(fmt.Sprintf("Path: %s\n", action.Path))
	case ActionEditFile:
		sb.WriteString(fmt.Sprintf("Path: %s\n", action.Path))
	}

	if err != nil {
		sb.WriteString(fmt.Sprintf("ERROR: %v\n", err))
	}

	if output != "" {
		sb.WriteString("Output:\n")
		sb.WriteString(output)
		sb.WriteString("\n")
	}

	sb.WriteString("=== End Action ===\n")

	return sb.String()
}

// TruncateOutput truncates long outputs for the prompt.
func TruncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}

	half := maxLen / 2
	return output[:half] + fmt.Sprintf("\n... [%d chars truncated] ...\n", len(output)-maxLen) + output[len(output)-half:]
}
