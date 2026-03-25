package fzf_test

import (
	"context"
	"testing"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/fzf"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/testutil"
)

func TestPickEventFields(t *testing.T) {
	ctx := context.Background()
	ctx = testutil.SetupLoggingV2(ctx, log.LEVEL_DEBUG)

	// Create a config with a project and use --filter for non-interactive selection
	cfg := &config.Config{
		FzfFlags: []string{"--filter=test-project"},
	}

	options := []string{"abcd", "123", "test-project", "hello"}

	// Call PickProject - fzf will run non-interactively with --filter
	_, err := fzf.Pick(ctx, cfg, options)
	if err != nil {
		t.Fatal(err)
	}

	res := testutil.GetLogEventV2(t, ctx)

	// Check pickProjectEvent exists
	event, ok := res["fzf"].(map[string]any)
	if !ok {
		t.Fatal("pickEvent not found in log output")
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

	// Check options field exists and contains fzf command
	optionsOut, ok := event["options"].([]any)
	if !ok {
		t.Fatal("pickProjectEvent missing 'options' field")
	}
	if len(optionsOut) == 0 {
		t.Error("'options' should not be empty")
	}
	if optionsOut[0] != "abcd" {
		t.Errorf("First option should be 'abcd', got '%s'", optionsOut[0])
	}

	// Check output field exists
	output, ok := event["output"].(string)
	if !ok {
		t.Fatal("pickProjectEvent missing 'output' field")
	}
	if output != "test-project" {
		t.Error("'output' should be 'test-project'")
	}

	// errors field should be omitted when there are no errors
	if _, ok := event["errors"]; ok {
		t.Error("'errors' field should be omitted when there are no errors")
	}
}

func TestPickProjectEventWithError(t *testing.T) {
	ctx := context.Background()
	ctx = testutil.SetupLoggingV2(ctx, log.LEVEL_DEBUG)

	// Create a config with --filter for a non-existent project
	cfg := &config.Config{
		FzfFlags: []string{"--filter=nonexistent-project"},
	}

	options := []string{"abcd", "123", "test-project", "hello"}

	// Call PickProject - should fail because filter matches nothing
	_, err := fzf.Pick(ctx, cfg, options)
	if err == nil {
		t.Fatal("Expected an error when filtering for non-existent project")
	}

	res := testutil.GetLogEventV2(t, ctx)

	// Check pickProjectEvent exists
	event, ok := res["fzf"].(map[string]any)
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
