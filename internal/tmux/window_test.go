package tmux_test

import (
	"testing"

	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

func TestKill(t *testing.T) {
	f := func(tmux *tmux.Server) {
		s, err := tmux.New("test_session")
		if err != nil {
			t.Fatal(err)
		}
		w, err := s.NewWindow("test-window")
		if err != nil {
			t.Fatal(err)
		}
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
		err = w.Kill()
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
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestWindowName(t *testing.T) {
	f := func(tmux *tmux.Server) {
		s, _ := tmux.New("test_ses")
		w, _ := s.NewWindow("test_window")
		res, err := w.Name()
		if err != nil {
			t.Fatal(err)
		}
		if res != "test_window" {
			t.Fatal("Names dont match")
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestPanes(t *testing.T) {
	f := func(tmux *tmux.Server) {
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
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestLink(t *testing.T) {
	f := func(tmux *tmux.Server) {
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
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestWindowOptions(t *testing.T) {
	f := func(tmux *tmux.Server) {
		name := "test_sess_name"
		s, _ := tmux.New(name)
		err := s.SetOption("@hello", "world")
		if err != nil {
			t.Fatal(err)
		}
		windows, _ := s.Windows()
		res, err := windows[0].GetOption("@hello")
		if err != nil {
			t.Fatal(err)
		}
		if res != "world" {
			t.Fatal("values dont match")
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestWindowOptionsNotSet(t *testing.T) {
	f := func(tmux *tmux.Server) {
		name := "test_sess_name"
		s, _ := tmux.New(name)
		windows, _ := s.Windows()
		_, err := windows[0].GetOption("@hello")
		if err == nil {
			t.Fatal("Should return an error")
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}
