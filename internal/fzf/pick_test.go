package fzf_test

import (
	"context"
	"testing"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/fzf"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/testutil"
)

func TestPick(t *testing.T) {
	ctx := context.Background()
	ctx = testutil.SetupLoggingV2(ctx, log.LEVEL_DEBUG)

	// Create a config with a project and use --filter for non-interactive selection
	cfg := &config.Config{
		FzfFlags: []string{"--filter=test-project"},
	}

	options := []string{"abcd", "123", "test-project", "hello"}

	// Call PickProject - fzf will run non-interactively with --filter
	selectedProject, err := fzf.Pick(ctx, cfg, options)
	if err != nil {
		t.Fatal(err)
	}

	if *selectedProject != "test-project" {
		t.Errorf("Expected 'test-project', got '%s'", *selectedProject)
	}
}
