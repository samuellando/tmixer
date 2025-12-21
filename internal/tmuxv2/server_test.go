package tmuxv2_test

import (
	"testing"
	
	"samuellando.com/tmixer/internal/tmuxv2"
	"samuellando.com/tmixer/internal/testutil"
)

func TestHasSession(t *testing.T) {
	f := func(tmux *tmuxv2.Server) {
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
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestHasSessionWithName(t *testing.T) {
	f := func(tmux *tmuxv2.Server) {
		name := "test_session"
		s, err := tmux.New(name)
		if err != nil {
			t.Fatal(err)
		}
		if !tmux.HasSessionWithName(name) {
			t.Fatal("Should have session")
		}
		s.Kill()
		if tmux.HasSessionWithName(name) {
			t.Fatal("Should have session anymore")
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestGetSessionWithNameFound(t *testing.T) {
	f := func(tmux *tmuxv2.Server) {
		name := "test_session"
		s, err := tmux.New(name)
		if err != nil {
			t.Fatal(err)
		}
		res, err := tmux.GetSessionWithName(name)
		if err != nil {
			t.Fatal(err)
		}
		if s.Id != res.Id {
			t.Fatal("Session id does not match")
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestGetSessionWithNameNotFound(t *testing.T) {
	f := func(tmux *tmuxv2.Server) {
		name := "test_session"
		_, err := tmux.GetSessionWithName(name)
		if err != tmuxv2.ErrSessionNotFound {
			t.Fatal(err)
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}
