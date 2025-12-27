package tmux_test

import (
	"fmt"
	"testing"

	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

func TestCreateAndKill(t *testing.T) {
	f := func(tmux *tmux.Server) {
		s, err := tmux.New("test_session")
		if err != nil {
			t.Fatal(err)
		}
		if !tmux.HasSession(s) {
			t.Fatal("Should have session")
		}
		err = s.Kill()
		if err != nil {
			t.Fatal(err)
		}
		if tmux.HasSession(s) {
			t.Fatal("Should have session anymore")
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestListSessions(t *testing.T) {
	f := func(s *tmux.Server) {
		n := 10
		res, _ := s.ListSessions()
		initialCount := len(res)
		sessions := make([]*tmux.Session, n)
		var err error
		for i := range n {
			sessions[i], err = s.New(fmt.Sprintf("test_session_%d", i))
			if err != nil {
				t.Fatal(err)
			}
		}
		res, err = s.ListSessions()
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
		if len(res)-initialCount != n {
			t.Fatalf("Expected %d sessions got %d", n, len(res)-initialCount)
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestWindows(t *testing.T) {
	f := func(srv *tmux.Server) {
		n := 10
		sessions, _ := srv.ListSessions()
		s := sessions[0]
		windows := make([]*tmux.Window, n)
		var err error
		for i := range n {
			windows[i], err = s.NewWindow("test")
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
		if len(res) != n+1 {
			t.Fatalf("Expected %d windows got %d", n+1, len(res))
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestSessionName(t *testing.T) {
	f := func(tmux *tmux.Server) {
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
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestSessionLastActivity(t *testing.T) {
	f := func(tmux *tmux.Server) {
		s1, _ := tmux.New("s1")
		s2, _ := tmux.New("s2")
		t1, err := s1.LastActivity()
		if err != nil {
			t.Fatal(err)
		}
		t2, err := s2.LastActivity()
		if err != nil {
			t.Fatal(err)
		}
		if t1.After(*t2) {
			t.Fatal("Shoul be before")
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestOptions(t *testing.T) {
	f := func(tmux *tmux.Server) {
		name := "test_sess_name"
		s, _ := tmux.New(name)
		err := s.SetOption("@hello", "world")
		if err != nil {
			t.Fatal(err)
		}
		res, err := s.GetOption("@hello")
		if err != nil {
			t.Fatal(err)
		}
		if res != "world" {
			t.Fatal("values dont match")
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestOptionsNotSet(t *testing.T) {
	f := func(tmux *tmux.Server) {
		name := "test_sess_name"
		s, _ := tmux.New(name)
		_, err := s.GetOption("@hello")
		if err == nil {
			t.Fatal("Should return an error")
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}
