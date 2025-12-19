package tmuxv2_test

import (
	"testing"
	"samuellando.com/tmixer/internal/testutil"
)

func TestHasSession(t *testing.T) {
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

func TestHasSessionWithName(t *testing.T) {
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
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
