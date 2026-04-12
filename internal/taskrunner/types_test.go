package taskrunner

import (
	"testing"
	"time"
)

func TestDefaultRunConfig(t *testing.T) {
	task := "test task"
	config := DefaultRunConfig(task)

	if config.Task != task {
		t.Errorf("expected task %q, got %q", task, config.Task)
	}
	if config.WorkDir != "." {
		t.Errorf("expected workdir '.', got %q", config.WorkDir)
	}
	if config.MaxIter != 30 {
		t.Errorf("expected maxIter 30, got %d", config.MaxIter)
	}
	if config.Verbose != false {
		t.Error("expected verbose false")
	}
	if config.Timeout != 2*time.Minute {
		t.Errorf("expected timeout 2m, got %v", config.Timeout)
	}
}

func TestRunConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  RunConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultRunConfig("test task"),
			wantErr: false,
		},
		{
			name: "empty task",
			config: RunConfig{
				Task:    "",
				WorkDir: ".",
				MaxIter: 30,
				Timeout: time.Minute,
			},
			wantErr: true,
		},
		{
			name: "max iter too high",
			config: RunConfig{
				Task:    "test",
				WorkDir: ".",
				MaxIter: 200,
				Timeout: time.Minute,
			},
			wantErr: true,
		},
		{
			name: "whitespace task",
			config: RunConfig{
				Task:    "   ",
				WorkDir: ".",
				MaxIter: 30,
				Timeout: time.Minute,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestActionJSON(t *testing.T) {
	action := Action{
		Type:    ActionShell,
		Command: "echo hello",
		Reason:  "test command",
	}

	json := action.JSON()
	if json == "" {
		t.Error("JSON() returned empty string")
	}

	// Should contain action type
	if !contains(json, "shell") {
		t.Error("JSON should contain action type")
	}
}

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			input:  "hello",
			maxLen: 100,
			want:   "hello",
		},
		{
			name:   "long string",
			input:  makeString(1000),
			maxLen: 100,
			want:   "truncated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateOutput(tt.input, tt.maxLen)
			if tt.want == "truncated" {
				if len(got) >= len(tt.input) {
					t.Error("expected truncated output")
				}
				if !contains(got, "truncated") {
					t.Error("expected truncation marker")
				}
			} else if got != tt.want {
				t.Errorf("TruncateOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func makeString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(s[:len(substr)] == substr) ||
		(s[len(s)-len(substr):] == substr) ||
		findInString(s, substr))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
