package tmux_test

import (
	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
	"testing"
	"time"
)

func TestStartStopControlMode(t *testing.T) {
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
	for range 10 {
		err := tmux.StartControlMode()
		if err != nil {
			t.Fatal(err)
		}
		err = tmux.StopControlMode()
		if err != nil {
			t.Fatal(err)
		}
	}
	if tmux.HasSessionWithName(tmux.CONTROL_SESSION_NAME) {
		t.Fatal("CONTROL_SESSION_NAME detected")
	}
}

func TestStartStopControlModeSimul(t *testing.T) {
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
	n := 10
	servers := make([]*tmux.Server, n)
	var err error
	for i := range n {
		servers[i] = tmux.Tmux(tmux.SocketPath)
		servers[i].StartControlMode()
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := range n {
		err = servers[i].StopControlMode()
		if err != nil {
			t.Fatal(err)
		}
	}
	if tmux.HasSessionWithName(tmux.CONTROL_SESSION_NAME) {
		t.Fatal("CONTROL_SESSION_NAME detected")
	}
}

func TestRunCommand(t *testing.T) {
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
	tmux.StartControlMode()
	defer tmux.StopControlMode()
	_, err := tmux.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if !tmux.HasSessionWithName(tmux.CONTROL_SESSION_NAME) {
		t.Fatal("CONTROL_SESSION_NAME not detected")
	}
}

func TestUsesControlMode(t *testing.T) {
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
	// Since control mode is so much fater (~7000x) this is a reliable test
	n := 100
	start := time.Now()
	for range n {
		_, err := tmux.ListSessions()
		if err != nil {
			t.Fatal(err)
		}
	}
	psTime := time.Since(start)
	tmux.StartControlMode()
	defer tmux.StopControlMode()
	start = time.Now()
	for range n {
		_, err := tmux.ListSessions()
		if err != nil {
			t.Fatal(err)
		}
	}
	cmTime := time.Since(start)
	if !(20*cmTime <= psTime) {
		t.Fatal("Should be slower")
	}
}

func BenchmarkRun(b *testing.B) {
	tmux := testutil.SetupTestServer(b)
	defer testutil.TeardownTestServer(tmux)
	for b.Loop() {
		_, err := tmux.ListSessions()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkControlModeRun(b *testing.B) {
	tmux := testutil.SetupTestServer(b)
	defer testutil.TeardownTestServer(tmux)
	tmux.StartControlMode()
	defer tmux.StopControlMode()
	for b.Loop() {
		_, err := tmux.ListSessions()
		if err != nil {
			b.Fatal(err)
		}
	}
}
