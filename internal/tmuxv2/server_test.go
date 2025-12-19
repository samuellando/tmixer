package tmuxv2_test

import (
	"samuellando.com/tmixer/internal/tmuxv2"
	"testing"
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
	runWithAndWithoutControlMode(f, t)
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
	runWithAndWithoutControlMode(f, t)
}
