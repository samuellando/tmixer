package tmux_test

import (
	"context"
	"testing"

	"samuellando.com/tmixer/internal/log/v2"
	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

func TestControlModeSessionTracking(t *testing.T) {
	ctx, srv := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(srv)

	err := srv.StartControlMode()
	if err != nil {
		t.Fatal(err)
	}

	// Run multiple commands
	numCommands := 5
	for range numCommands {
		_, err := srv.ListSessions()
		if err != nil {
			t.Error(err)
		}
	}

	err = srv.StopControlMode()
	if err != nil {
		t.Fatal(err)
	}

	res := testutil.GetLogEvent(t, ctx)
	session := res["controlModeSession"].(map[string]any)

	// Check that command runs are tracked
	commandRuns := int(session["commandRuns"].(float64))
	if commandRuns != numCommands {
		t.Errorf("Expected %d command runs, got %d", numCommands, commandRuns)
	}

	// Check that average duration is set
	if _, ok := session["averageDuration"]; !ok {
		t.Error("controlModeSession should have 'AverageDuration' field")
	}

	avgDuration := int64(session["averageDuration"].(float64))
	if avgDuration <= 0 {
		t.Error("AverageDuration should be greater than 0")
	}
}

func TestControlModeSessionErrors(t *testing.T) {
	// Create a server that will fail to start control mode
	// by using an invalid socket path
	ctx := log.ContextLogger(context.Background())
	srv := tmux.Tmux(ctx, "/nonexistent/path/socket")

	err := srv.StartControlMode()
	if err != nil {
		t.Error(err)
	}
	err = srv.StopControlMode()
	if err != nil {
		t.Error(err)
	}

	res := testutil.GetLogEvent(t, ctx)
	session := res["controlModeSession"].(map[string]any)

	// Should have errors
	errors, ok := session["errors"]
	if !ok {
		t.Error("controlModeSession should have 'errors' field when failing to start")
	}

	errorsList := errors.([]any)
	if len(errorsList) == 0 {
		t.Error("errors field should not be empty when control mode fails to start")
	}
}
