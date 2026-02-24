package tmux_test

import (
	"context"
	"fmt"
	"testing"

	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

func TestCreateAndKill(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		s, err := srv.New("test_session")
		if err != nil {
			t.Fatal(err)
		}
		if !srv.HasSession(s) {
			t.Fatal("Should have session")
		}
		err = s.Kill()
		if err != nil {
			t.Fatal(err)
		}
		if srv.HasSession(s) {
			t.Fatal("Should have session anymore")
		}
	})
}

func TestListSessions(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		n := 10
		res, _ := srv.ListSessions()
		initialCount := len(res)
		sessions := make([]*tmux.Session, n)
		var err error
		for i := range n {
			sessions[i], err = srv.New(fmt.Sprintf("test_session_%d", i))
			if err != nil {
				t.Fatal(err)
			}
		}
		res, err = srv.ListSessions()
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
	})
}

func TestWindows(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
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
	})
}

func TestSessionName(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		name := "test_session_name"
		s, _ := srv.New(name)
		res, err := s.Name()
		if err != nil {
			t.Fatal(err)
		}
		if res != name {
			t.Fatal("Names do not match")
		}
	})
}

func TestSessionReName(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		name := "test_session_name"
		rename := "new_test_session_name"
		s, _ := srv.New(name)
		res, err := s.Name()
		if err != nil {
			t.Fatal(err)
		}
		if res != name {
			t.Fatal("Names do not match")
		}
		err = s.Rename(rename)
		if err != nil {
			t.Fatal(err)
		}
		res, err = s.Name()
		if err != nil {
			t.Fatal(err)
		}
		if res != rename {
			t.Fatal("Renames do not match")
		}
	})
}

func TestSessionLastActivity(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		s1, _ := srv.New("s1")
		s2, _ := srv.New("s2")
		t1, err := s1.LastActivity()
		if err != nil {
			t.Fatal(err)
		}
		t2, err := s2.LastActivity()
		if err != nil {
			t.Fatal(err)
		}
		if t1.After(*t2) {
			t.Fatal("Should be before")
		}
	})
}

func TestOptions(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		name := "test_session_name"
		s, _ := srv.New(name)
		err := s.SetOption("@hello", "world")
		if err != nil {
			t.Fatal(err)
		}
		res, err := s.GetOption("@hello")
		if err != nil {
			t.Fatal(err)
		}
		if res != "world" {
			t.Fatal("values do not match")
		}
	})
}

func TestOptionsNotSet(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		name := "test_session_name"
		s, _ := srv.New(name)
		_, err := s.GetOption("@hello")
		if err == nil {
			t.Fatal("Should return an error")
		}
	})
}
