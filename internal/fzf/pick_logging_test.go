package fzf_test

import (
	"testing"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/fzf"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/testutil"
)

func TestPickProjectEventFields(t *testing.T) {
	ctx, tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)

	// Set up logging
	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

	// Create a config with a project and use --filter for non-interactive selection
	cfg := &config.Config{
		FzfFlags: []string{"--filter=test-project"},
		Projects: []*config.ProjectConfig{
			{
				Name:      "test-project",
				Directory: "/tmp/test",
			},
		},
	}

	// Get the projects list
	projects, err := project.List(ctx, tmux, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Call PickProject - fzf will run non-interactively with --filter
	selectedProject, err := fzf.Pick(ctx, cfg, projects)
	if err != nil {
		t.Fatal(err)
	}

	if selectedProject.Name != "test-project" {
		t.Errorf("Expected 'test-project', got '%s'", selectedProject.Name)
	}

	res := testutil.GetLogEvent(ctx, logger, out)

	// Check pickProjectEvent exists
	event, ok := res["pickProjectEvent"].(map[string]any)
	if !ok {
		t.Fatal("pickProjectEvent not found in log output")
	}

	// Check args field exists and contains fzf command
	args, ok := event["args"].([]any)
	if !ok {
		t.Fatal("pickProjectEvent missing 'args' field")
	}
	if len(args) == 0 {
		t.Error("'args' should not be empty")
	}
	if args[0] != "fzf" {
		t.Errorf("First arg should be 'fzf', got '%s'", args[0])
	}

	// Check output field exists
	output, ok := event["output"].(string)
	if !ok {
		t.Fatal("pickProjectEvent missing 'output' field")
	}
	if output == "" {
		t.Error("'output' should not be empty")
	}

	// Check parsedOutput field exists and matches selected project
	parsedOutput, ok := event["parsedOutput"].(string)
	if !ok {
		t.Fatal("pickProjectEvent missing 'parsedOutput' field")
	}
	if parsedOutput != "test-project" {
		t.Errorf("'parsedOutput' should be 'test-project', got '%s'", parsedOutput)
	}

	// errors field should be omitted when there are no errors
	if _, ok := event["errors"]; ok {
		t.Error("'errors' field should be omitted when there are no errors")
	}
}

func TestPickProjectEventWithError(t *testing.T) {
	ctx, tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)

	// Set up logging
	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

	// Create a config with --filter for a non-existent project
	cfg := &config.Config{
		FzfFlags: []string{"--filter=nonexistent-project"},
		Projects: []*config.ProjectConfig{
			{
				Name:      "test-project",
				Directory: "/tmp/test",
			},
		},
	}

	// Get the projects list
	projects, err := project.List(ctx, tmux, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Call PickProject - should fail because filter matches nothing
	_, err = fzf.Pick(ctx, cfg, projects)
	if err == nil {
		t.Fatal("Expected an error when filtering for non-existent project")
	}

	res := testutil.GetLogEvent(ctx, logger, out)

	// Check pickProjectEvent exists
	event, ok := res["pickProjectEvent"].(map[string]any)
	if !ok {
		t.Fatal("pickProjectEvent not found in log output")
	}

	// errors field should be present
	errorsField, ok := event["errors"]
	if !ok {
		t.Fatal("'errors' field should be present when errors occur")
	}

	// Verify errors is an array with at least 1 error
	errorsList, ok := errorsField.([]any)
	if !ok {
		t.Fatalf("'errors' field should be an array, got: %T", errorsField)
	}
	if len(errorsList) == 0 {
		t.Error("'errors' should not be empty when an error occurs")
	}
}
