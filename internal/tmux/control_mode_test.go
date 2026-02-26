package tmux_test

import (
	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
	"testing"
	"time"
)

func TestStartStopControlMode(t *testing.T) {
	_, s := testutil.SetupTestServer(t)
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

func TestStartStopControlModeParallel(t *testing.T) {
	ctx, s := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(s)
	n := 10
	servers := make([]*tmux.Server, n)
	var err error
	for i := range n {
		servers[i] = tmux.Tmux(ctx, s.SocketPath)
		err = servers[i].StartControlMode()
		if err != nil {
			t.Fatal(err)
		}
		sessions, err := servers[i].ListSessions()
		if err != nil {
			t.Fatal(err)
		}
		if len(sessions) == 0 {
			t.Fatal("expected some sessions")
		}
	}
	for i := range n {
		sessions, err := servers[i].ListSessions()
		if err != nil {
			t.Fatal(err)
		}
		if len(sessions) == 0 {
			t.Fatal("expected some sessions")
		}
		err = servers[i].StopControlMode()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunCommand(t *testing.T) {
	_, s := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(s)
	if err := s.StartControlMode(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.StopControlMode(); err != nil {
			t.Fatal(err)
		}
	}()
	_, err := s.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if !s.HasSessionWithName(tmux.CONTROL_SESSION_NAME) {
		t.Fatal("CONTROL_SESSION_NAME not detected")
	}
}

func TestUsesControlMode(t *testing.T) {
	_, s := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(s)
	// Since control mode is so much faster (~7000x) this is a reliable test
	n := 100
	start := time.Now()
	for range n {
		_, err := s.ListSessions()
		if err != nil {
			t.Fatal(err)
		}
	}
	psTime := time.Since(start)
	if err := s.StartControlMode(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.StopControlMode(); err != nil {
			t.Fatal(err)
		}
	}()
	start = time.Now()
	for range n {
		_, err := s.ListSessions()
		if err != nil {
			t.Fatal(err)
		}
	}
	cmTime := time.Since(start)
	if 20*cmTime > psTime {
		t.Fatal("Should be slower")
	}
}

func BenchmarkRun(b *testing.B) {
	_, s := testutil.SetupTestServer(b)
	defer testutil.TeardownTestServer(s)
	for b.Loop() {
		_, err := s.ListSessions()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkControlModeRun(b *testing.B) {
	_, s := testutil.SetupTestServer(b)
	defer testutil.TeardownTestServer(s)
	if err := s.StartControlMode(); err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := s.StopControlMode(); err != nil {
			b.Fatal(err)
		}
	}()
	for b.Loop() {
		_, err := s.ListSessions()
		if err != nil {
			b.Fatal(err)
		}
	}
}
