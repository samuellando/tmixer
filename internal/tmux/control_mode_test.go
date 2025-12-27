package tmux_test

import (
	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
	"testing"
	"time"
)

func TestStartStopControlMode(t *testing.T) {
	s := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(s)
	for range 10 {
		err := s.StartControlMode()
		if err != nil {
			t.Fatal(err)
		}
		err = s.StopControlMode()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestStartStopControlModeSimul(t *testing.T) {
	s := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(s)
	n := 10
	servers := make([]*tmux.Server, n)
	var err error
	for i := range n {
		servers[i] = tmux.Tmux(s.SocketPath)
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
}

func TestRunCommand(t *testing.T) {
	s := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(s)
	s.StartControlMode()
	defer s.StopControlMode()
	_, err := s.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if !s.HasSessionWithName(tmux.CONTROL_SESSION_NAME) {
		t.Fatal("CONTROL_SESSION_NAME not detected")
	}
}

func TestUsesControlMode(t *testing.T) {
	s := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(s)
	// Since control mode is so much fater (~7000x) this is a reliable test
	n := 100
	start := time.Now()
	for range n {
		_, err := s.ListSessions()
		if err != nil {
			t.Fatal(err)
		}
	}
	psTime := time.Since(start)
	s.StartControlMode()
	defer s.StopControlMode()
	start = time.Now()
	for range n {
		_, err := s.ListSessions()
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
	s := testutil.SetupTestServer(b)
	defer testutil.TeardownTestServer(s)
	for b.Loop() {
		_, err := s.ListSessions()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkControlModeRun(b *testing.B) {
	s := testutil.SetupTestServer(b)
	defer testutil.TeardownTestServer(s)
	s.StartControlMode()
	defer s.StopControlMode()
	for b.Loop() {
		_, err := s.ListSessions()
		if err != nil {
			b.Fatal(err)
		}
	}
}
