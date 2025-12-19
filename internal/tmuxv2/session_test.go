package tmuxv2_test

import (
	"fmt"
	"testing"

	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmuxv2"
)

func TestCreateAndKill(t *testing.T) {
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
	s, err := tmux.New("test_session")
	if err != nil {
		t.Fatal(err)
	}
	if !tmux.HasSession(s) {
		t.Fatal("Should have session")
	}
	s.Kill()
	if tmux.HasSession(s) {
		t.Fatal("Should have session anymore")
	}
}

func TestListSessions(t *testing.T) {
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
	n := 10
	res, _ := tmux.ListSessions()
	initialCount := len(res)
	sessions := make([]*tmuxv2.Session, n)
	var err error
	for i := range n {
		sessions[i], err = tmux.New(fmt.Sprintf("test_session_%d", i))
		if err != nil {
			t.Fatal(err)
		}
	}
	res, err = tmux.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	for _, session := range sessions {
		found := false
		for _, res_session := range res {
			if res_session.Id == session.Id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Session %s not found", session.Id)
		}
	}
	if len(res) - initialCount != n {
		t.Fatalf("Expected %d sessions got %d", n, len(res) - initialCount)
	}
}

func TestKillSessionWithName(t *testing.T) {
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
	name := "abcd123"
	s, _ := tmux.New(name)
	list, _ := tmux.ListSessions()
	found := false
	for _, session := range list {
		if s.Id == session.Id {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Session not found")
	}
	err := tmux.KillSessionWithName(name)
	if err != nil {
		t.Fatal(err)
	}
	list, _ = tmux.ListSessions()
	found = false
	for _, session := range list {
		if s.Id == session.Id {
			found = true
			break
		}
	}
	if found {
		t.Fatal("Session should be killed")
	}
}

func TestWindows(t *testing.T) {
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
	n := 10
	sessions, _ := tmux.ListSessions()
	s := sessions[0]
	windows := make([]*tmuxv2.Window, n)
	var err error
	for i := range n {
		windows[i], err = s.NewWindow("~", "test", "sh")
		if err != nil {
			t.Fatal(err)
		}
	}
	res, err := s.Windows()
	if err != nil {
		t.Fatal(err)
	}
	for _, window := range windows {
		found := false
		for _, res_window := range res {
			if res_window.Id == window.Id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Window %s not found", window.Id)
		}
	}
	if len(res) != n + 1 {
		t.Fatalf("Expected %d windows got %d", n + 1, len(res))
	}
}

func TestName(t *testing.T) {
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
	name := "test_sess_name"
	s, _ := tmux.New(name)
	res, err := s.Name()
	if err != nil {
		t.Fatal(err)
	}
	if res != name {
		t.Fatal("Names dont match")
	}
}
