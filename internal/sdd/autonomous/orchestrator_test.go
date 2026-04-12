package autonomous

import (
	"testing"
)

func TestDetectComplexity(t *testing.T) {
	tests := []struct {
		name       string
		task       string
		wantSDD    bool
		wantReason string
	}{
		{
			name:    "simple typo fix",
			task:    "fix typo in readme",
			wantSDD: false,
		},
		{
			name:    "simple test add",
			task:    "add test for auth function",
			wantSDD: false,
		},
		{
			name:    "complex redesign",
			task:    "redesign the authentication system",
			wantSDD: true,
		},
		{
			name:    "complex architecture",
			task:    "implement new architecture for payments",
			wantSDD: true,
		},
		{
			name:    "simple script",
			task:    "create a simple python script",
			wantSDD: false,
		},
		{
			name:    "complex migration",
			task:    "migrate the database to new schema",
			wantSDD: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSDD, _ := DetectComplexity(tt.task)
			if gotSDD != tt.wantSDD {
				t.Errorf("DetectComplexity(%q) sdd = %v, want %v", tt.task, gotSDD, tt.wantSDD)
			}
		})
	}
}

func TestDeterminePhaseOrder(t *testing.T) {
	o := &Orchestrator{}

	tests := []struct {
		name       string
		start      PhaseType
		end        PhaseType
		skipVerify bool
		want       []PhaseType
	}{
		{
			name: "full cycle",
			start: PhaseExplore,
			end:   PhaseArchive,
			want:  []PhaseType{PhaseExplore, PhasePropose, PhaseSpec, PhaseDesign, PhaseTasks, PhaseApply, PhaseVerify, PhaseArchive},
		},
		{
			name: "explore to propose only",
			start: PhaseExplore,
			end:   PhasePropose,
			want:  []PhaseType{PhaseExplore, PhasePropose},
		},
		{
			name:       "apply to archive skip verify",
			start:      PhaseApply,
			end:        PhaseArchive,
			skipVerify: true,
			want:       []PhaseType{PhaseApply, PhaseArchive},
		},
		{
			name: "design to tasks",
			start: PhaseDesign,
			end:   PhaseTasks,
			want:  []PhaseType{PhaseDesign, PhaseTasks},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := o.determinePhaseOrder(tt.start, tt.end, tt.skipVerify)
			if len(got) != len(tt.want) {
				t.Errorf("determinePhaseOrder() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("determinePhaseOrder()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
