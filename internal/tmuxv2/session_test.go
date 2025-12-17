package tmuxv2

import (
	"fmt"
	"testing"
)

func TestCreateAndKill(t *testing.T) {
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

func TestListSessions(t *testing.T) {
	tmux := setupTestServer(t)
	defer teardownTestServer(tmux)
	n := 10
	sessions := make([]*Session, n)
	var err error
	for i := range n {
		sessions[i], err = tmux.New(fmt.Sprintf("test_session_%d", i))
		if err != nil {
			t.Fatal(err)
		}
	}
	//TODO
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range sessions {
		s.Kill()
	}
}
