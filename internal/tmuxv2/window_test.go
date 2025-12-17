package tmuxv2

import "testing"

func TestKill(t *testing.T) {
	tmux := setupTestServer(t)
	defer teardownTestServer(tmux)
	s, _ := tmux.New("test_session")
	w, _ := s.NewWindow("~",  "test-window", "sh")
	windows, _ := s.Windows()
	found := false
	for _, window := range windows {
		if window.Id == w.Id {
			found = true
		}
	}
	if !found {
		t.Fatal("Window was not found")
	}
	err := w.Kill()
	if err != nil {
		t.Fatal(err)
	}
	windows, _ = s.Windows()
	found = false
	for _, window := range windows {
		if window.Id == w.Id {
			found = true
		}
	}
	if found {
		t.Fatal("Window was not killed")
	}
}

func TestPanes(t *testing.T) {
	tmux := setupTestServer(t)
	defer teardownTestServer(tmux)
	s, _ := tmux.New("test_session")
	windows, _ := s.Windows()
	w := windows[0]
	panes, _ := w.Panes()
	if len(panes) != 1 {
		t.Fatal("Window should have one pane")
	}
	panes[0].Split()
	panes, _ = w.Panes()
	if len(panes) != 2 {
		t.Fatal("Window should have two panes")
	}
}
