package tmux_test

import (
	"context"
	"testing"

	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

func TestHasSession(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		s, err := srv.New("test_session")
		if err != nil {
			t.Fatal(err)
		}
		if !srv.HasSession(s) {
			t.Fatal("Should have session")
		}
		s.Kill()
		if srv.HasSession(s) {
			t.Fatal("Should have session anymore")
		}
	})
}

func TestHasSessionWithName(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		name := "test_session"
		s, err := srv.New(name)
		if err != nil {
			t.Fatal(err)
		}
		if !srv.HasSessionWithName(name) {
			t.Fatal("Should have session")
		}
		s.Kill()
		if srv.HasSessionWithName(name) {
			t.Fatal("Should have session anymore")
		}
	})
}

func TestGetSessionWithNameFound(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		name := "test_session"
		s, err := srv.New(name)
		if err != nil {
			t.Fatal(err)
		}
		res, err := srv.GetSessionWithName(name)
		if err != nil {
			t.Fatal(err)
		}
		if s.Id != res.Id {
			t.Fatal("Session id does not match")
		}
	})
}

func TestGetSessionWithNameNotFound(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		name := "test_session"
		_, err := srv.GetSessionWithName(name)
		if err != tmux.ErrSessionNotFound {
			t.Fatal(err)
		}
	})
}
