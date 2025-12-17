package tmuxv2

import (
	"testing"
)

func TestHasSession(t *testing.T) {
	tmux := setupTestServer(t)
	defer teardownTestServer(tmux)
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

func TestHasSessionWithName(t *testing.T) {
	tmux := setupTestServer(t)
	defer teardownTestServer(tmux)
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
