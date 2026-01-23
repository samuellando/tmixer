package tmux_test

import (
	"context"
	"strings"
	"testing"

	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

func TestCommandDoesNotLogAtInfo(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		// Test with default log level (should not log command events)
		ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_INFO)

		_, err := srv.ListSessions()
		if err != nil {
			t.Error(err)
		}

		res := testutil.GetLogEvent(ctx, logger, out)

		// Should not have tmuxCommandEvent at default log level
		if _, ok := res["tmuxCommandEvent"]; ok {
			t.Error("Should not log command events at default log level")
		}
	})
}

func TestCommandLogsEventAtDebug(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

		_, err := srv.ListSessions()
		if err != nil {
			t.Error(err)
		}

		res := testutil.GetLogEvent(ctx, logger, out)

		// Should have tmuxCommandEvent at debug level
		if _, ok := res["tmuxCommandEvent"]; !ok {
			t.Error("Should log command events at DEBUG log level")
		}
	})
}

func TestCommandEventFields(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

		_, err := srv.ListSessions()
		if err != nil {
			t.Error(err)
		}

		res := testutil.GetLogEvent(ctx, logger, out)
		event := res["tmuxCommandEvent"].(map[string]any)

		// Check required fields exist
		if _, ok := event["args"]; !ok {
			t.Error("Command event missing 'args' field")
		}
		if _, ok := event["outputLines"]; !ok {
			t.Error("Command event missing 'outputLines' field")
		}
		if _, ok := event["controlMode"]; !ok {
			t.Error("Command event missing 'controlMode' field")
		}

		// Verify args is an array
		args, ok := event["args"].([]any)
		if !ok {
			t.Error("'args' field should be an array")
		}
		if len(args) == 0 {
			t.Error("'args' should not be empty")
		}

		// Verify outputLines is an array
		if _, ok := event["outputLines"].([]any); !ok {
			t.Error("'outputLines' field should be an array")
		}
	})
}

func TestControlModeFieldInEvent(t *testing.T) {
	t.Run("ControlMode", func(t *testing.T) {
		ctx, srv := testutil.SetupTestServer(t)
		defer testutil.TeardownTestServer(srv)

		err := srv.StartControlMode()
		if err != nil {
			t.Fatal(err)
		}

		ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

		_, err = srv.ListSessions()
		if err != nil {
			t.Error(err)
		}

		err = srv.StopControlMode()
		if err != nil {
			t.Fatal(err)
		}

		res := testutil.GetLogEvent(ctx, logger, out)
		event := res["tmuxCommandEvent"].(map[string]any)
		controlMode := event["controlMode"].(bool)

		if !controlMode {
			t.Error("controlMode field should be true when running in control mode")
		}

		// Should have rawInput field in control mode
		if _, ok := event["rawInput"]; !ok {
			t.Error("Should have 'rawInput' field in control mode")
		}
	})

	t.Run("NoControlMode", func(t *testing.T) {
		ctx, srv := testutil.SetupTestServer(t)
		defer testutil.TeardownTestServer(srv)

		ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

		_, err := srv.ListSessions()
		if err != nil {
			t.Error(err)
		}

		res := testutil.GetLogEvent(ctx, logger, out)
		event := res["tmuxCommandEvent"].(map[string]any)
		controlMode := event["controlMode"].(bool)

		if controlMode {
			t.Error("controlMode field should be false when not running in control mode")
		}

		// Should not have rawInput field when not in control mode
		if _, ok := event["rawInput"]; ok {
			t.Error("Should not have 'rawInput' field when not in control mode")
		}
	})
}

func TestCommandErrorLogging(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		// Create a session BEFORE setting up logging
		_, err := srv.New("test_error_session")
		if err != nil {
			t.Fatal(err)
		}

		// NOW set up logging to capture only the error command
		ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

		// Try to create a session with the same name - this should fail and be logged
		_, err = srv.New("test_error_session")
		if err == nil {
			t.Error("Expected an error when creating duplicate session")
		}

		res := testutil.GetLogEvent(ctx, logger, out)
		event, ok := res["tmuxCommandEvent"].(map[string]any)
		if !ok {
			t.Fatalf("tmuxCommandEvent not found in log output")
		}

		// Should have errors field
		errors, ok := event["errors"]
		if !ok || errors == nil {
			t.Error("Command event should have 'errors' field when command fails")
			return
		}

		errorsList, ok := errors.([]any)
		if !ok {
			t.Errorf("errors field should be an array, got: %T", errors)
			return
		}

		if len(errorsList) == 0 {
			t.Error("errors field should not be empty when command fails")
		}
	})
}

func TestControlModeSessionTracking(t *testing.T) {
	ctx, srv := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(srv)

	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

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

	res := testutil.GetLogEvent(ctx, logger, out)
	session := res["controlModeSession"].(map[string]any)

	// Check that command runs are tracked
	commandRuns := int(session["commandRuns"].(float64))
	if commandRuns != numCommands {
		t.Errorf("Expected %d command runs, got %d", numCommands, commandRuns)
	}

	// Check that average duration is set
	if _, ok := session["AverageDuration"]; !ok {
		t.Error("controlModeSession should have 'AverageDuration' field")
	}

	avgDuration := int64(session["AverageDuration"].(float64))
	if avgDuration <= 0 {
		t.Error("AverageDuration should be greater than 0")
	}
}

func TestControlModeSessionErrors(t *testing.T) {
	// Create a server that will fail to start control mode
	// by using an invalid socket path
	ctx, _ := log.New(context.Background(), nil)
	srv := tmux.Tmux(ctx, "/nonexistent/path/socket")

	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

	err := srv.StartControlMode()
	err = srv.StopControlMode()
	if err == nil {
		t.Error("Expected error when starting control mode with invalid socket")
	}

	res := testutil.GetLogEvent(ctx, logger, out)
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

func TestMultipleCommandsLoggedSeparately(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

		// Run multiple different commands
		_, err := srv.ListSessions()
		if err != nil {
			t.Error(err)
		}

		session, err := srv.New("test_session_logging")
		if err != nil {
			t.Error(err)
		}

		name, err := session.Name()
		if err != nil {
			t.Error(err)
		}

		_, err = srv.GetSessionWithName(name)
		if err != nil {
			t.Error(err)
		}

		// Count how many command events were logged
		res := testutil.GetLogEvent(ctx, logger, out)
		events := res["tmuxCommandEvents"].([]any)

		// Should have 4 command events (list-sessions, new-session, name, get session)
		if len(events) != 4 {
			t.Errorf("Expected at least 3 command events, found %d", len(events))
		}
	})
}

func TestRawInputOnlyInControlMode(t *testing.T) {
	t.Run("ControlMode_HasRawInput", func(t *testing.T) {
		ctx, srv := testutil.SetupTestServer(t)
		defer testutil.TeardownTestServer(srv)

		err := srv.StartControlMode()
		if err != nil {
			t.Fatal(err)
		}

		ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

		_, err = srv.ListSessions()
		if err != nil {
			t.Error(err)
		}

		err = srv.StopControlMode()
		if err != nil {
			t.Fatal(err)
		}

		res := testutil.GetLogEvent(ctx, logger, out)
		event := res["tmuxCommandEvent"].(map[string]any)
		rawInput, ok := event["rawInput"].(string)

		if !ok {
			t.Error("Control mode should have 'rawInput' field")
		}

		if rawInput == "" {
			t.Error("rawInput should not be empty in control mode")
		}

		// Verify rawInput contains the command
		if !strings.Contains(rawInput, "list-sessions") {
			t.Errorf("rawInput should contain command, got: %s", rawInput)
		}
	})

	t.Run("NoControlMode_NoRawInput", func(t *testing.T) {
		ctx, srv := testutil.SetupTestServer(t)
		defer testutil.TeardownTestServer(srv)

		ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

		_, err := srv.ListSessions()
		if err != nil {
			t.Error(err)
		}

		res := testutil.GetLogEvent(ctx, logger, out)
		event := res["tmuxCommandEvent"].(map[string]any)

		// Should not have rawInput in non-control mode
		if rawInput, ok := event["rawInput"]; ok && rawInput != "" {
			t.Error("Non-control mode should not have 'rawInput' field")
		}
	})
}
