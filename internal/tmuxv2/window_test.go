package tmuxv2_test

import  (
	"testing"
	"samuellando.com/tmixer/internal/testutil"
)

func TestKill(t *testing.T) {
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
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
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
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

func TestLink(t *testing.T) {
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
	s1, _ := tmux.New("test_session")
	s2, _ := tmux.New("test_session2")
	windows, _ := s1.Windows()
	w := windows[0]
	err := w.Link(s2)
	if err != nil {
		t.Fatal(err)
	}
	windows, _ = s2.Windows()
	if len(windows) != 2 {
		t.Fatal("Should have 2 windows")
	}
}
