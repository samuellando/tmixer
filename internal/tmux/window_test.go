package tmux_test

import (
	"context"
	"strings"
	"testing"

	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

func TestKill(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		s, err := srv.New("test_session")
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
	})
}

func TestWindowName(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		s, _ := srv.New("test_ses")
		w, _ := s.NewWindow("test_window")
		res, err := w.Name()
		if err != nil {
			t.Fatal(err)
		}
		if res != "test_window" {
			t.Fatal("Names dont match")
		}
	})
}

func TestWindowNameless(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		s, _ := srv.New("test_ses")
		w, _ := s.NewWindow()
		res, err := w.Name()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(res, "sh") {
			t.Fatal("Name does not contain 'sh'")
		}
	})
}

func TestPanes(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		s, _ := srv.New("test_session")
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
	})
}

func TestUnlink(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		s1, _ := srv.New("test_session")
		s2, _ := srv.New("test_session2")
		s1.NewWindow("extra window")
		windows, _ := s1.Windows()
		w := windows[0]
		err := w.Link(s2)
		if err != nil {
			t.Fatal(err)
		}
		w.Unlink(s1)
		windows, _ = s1.Windows()
		if len(windows) != 1 {
			t.Fatal("Should have 1 windows")
		}
	})
}

func TestLink(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		s1, _ := srv.New("test_session")
		s2, _ := srv.New("test_session2")
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
	})
}

func TestWindowOptions(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		name := "test_sess_name"
		s, _ := srv.New(name)
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
	})
}

func TestWindowOptionsNotSet(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		name := "test_sess_name"
		s, _ := srv.New(name)
		windows, _ := s.Windows()
		_, err := windows[0].GetOption("@hello")
		if err == nil {
			t.Fatal("Should return an error")
		}
	})
}
