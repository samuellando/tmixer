package tmux_test

import (
	"testing"

	"samuellando.com/tmixer/internal/testutil"
)

func TestStats(t *testing.T) {
	_, srv := testutil.SetupTestServer(t)
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

	// Check that command runs are tracked
	commandRuns := srv.Stats().CommandRuns
	if commandRuns-1 != numCommands {
		t.Errorf("Expected %d command runs, got %d", numCommands, commandRuns)
	}

	// Check that average duration is set
	duration := srv.Stats().TotalTime
	if duration <= 0 {
		t.Error("TotalDuration should be greater than 0")
	}
}

func TestSessionStats(t *testing.T) {
	_, srv := testutil.SetupTestServer(t)
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
	session := srv.Session()
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

	// Check that command runs are tracked
	commandRuns := srv.Stats().CommandRuns
	if commandRuns-1 != numCommands*2 {
		t.Errorf("Expected %d command runs, got %d", numCommands, commandRuns)
	}

	// Check that average duration is set
	duration := srv.Stats().TotalTime
	if duration <= 0 {
		t.Error("TotalDuration should be greater than 0")
	}

	// Check that command runs are tracked
	commandRuns = session.Stats().CommandRuns
	if commandRuns != numCommands {
		t.Errorf("Expected %d command runs, got %d", numCommands, commandRuns)
	}

	// Check that average duration is set
	duration = session.Stats().TotalTime
	if duration <= 0 && duration < srv.Session().Stats().TotalTime {
		t.Error("TotalDuration should be greater than 0 and less than server total time")
	}
}
