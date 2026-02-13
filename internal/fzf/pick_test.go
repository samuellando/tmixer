package fzf

import (
	"testing"

	"samuellando.com/tmixer/internal/project"
)

func TestParseOutput(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		name, err := parseOutput("x dogs")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "dogs" {
			t.Fatalf("unexpected name: %q", name)
		}
	})

	t.Run("valid with newline", func(t *testing.T) {
		name, err := parseOutput("x cats\n")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "cats" {
			t.Fatalf("unexpected name: %q", name)
		}
	})

	t.Run("invalid one part", func(t *testing.T) {
		_, err := parseOutput("dogs")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid too many parts", func(t *testing.T) {
		_, err := parseOutput("a b c")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestGetSelectedProject(t *testing.T) {
	first := &project.Project{Name: "alpha"}
	second := &project.Project{Name: "beta"}
	projects := []*project.Project{first, second}

	if got := getSelectedProject("beta", projects); got != second {
		t.Fatalf("unexpected project: %#v", got)
	}
	if got := getSelectedProject("missing", projects); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}
