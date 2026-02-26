package tmux_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

func TestSplit(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		sessions, _ := srv.ListSessions()
		s := sessions[0]
		windows, _ := s.Windows()
		w := windows[0]
		panes, _ := w.Panes()
		if len(panes) != 1 {
			t.Fatal("Should have one pane")
		}
		p := panes[0]
		p2, err := p.Split()
		if err != nil {
			t.Fatal(err)
		}
		panes, err = w.Panes()
		if err != nil {
			t.Error(err)
		}
		if len(panes) != 2 {
			t.Fatal("Should have two panes")
		}
		panes, err = w.Panes()
		if err != nil {
			t.Error(err)
		}
		found := false
		for _, pane := range panes {
			if pane.Id == p2.Id {
				found = true
			}
		}
		if !found {
			t.Fatal("Pane not listed!")
		}
	})
}

func TestSplitHorizontally(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		sessions, _ := srv.ListSessions()
		s := sessions[0]
		windows, _ := s.Windows()
		w := windows[0]
		panes, _ := w.Panes()
		if len(panes) != 1 {
			t.Fatal("Should have one pane")
		}
		p := panes[0]
		p2, err := p.SplitHorizontally()
		if err != nil {
			t.Fatal(err)
		}
		panes, err = w.Panes()
		if err != nil {
			t.Error(err)
		}
		if len(panes) != 2 {
			t.Fatal("Should have two panes")
		}
		panes, err = w.Panes()
		if err != nil {
			t.Error(err)
		}
		found := false
		for _, pane := range panes {
			if pane.Id == p2.Id {
				found = true
			}
		}
		if !found {
			t.Fatal("Pane not listed!")
		}
	})
}

func TestKillPane(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		sessions, _ := srv.ListSessions()
		s := sessions[0]
		windows, _ := s.Windows()
		w := windows[0]
		panes, _ := w.Panes()
		if len(panes) != 1 {
			t.Fatal("Should have one pane")
		}
		p := panes[0]
		p2, err := p.SplitHorizontally()
		if err != nil {
			t.Fatal(err)
		}
		panes, err = w.Panes()
		if err != nil {
			t.Error(err)
		}
		if len(panes) != 2 {
			t.Fatal("Should have two panes")
		}
		err = p.Kill()
		if err != nil {
			t.Fatal(err)
		}
		panes, err = w.Panes()
		if err != nil {
			t.Error(err)
		}
		if len(panes) != 1 {
			t.Fatal("Should have one pane")
		}
		found := false
		for _, pane := range panes {
			if pane.Id == p2.Id {
				found = true
			}
		}
		if !found {
			t.Fatal("Pane not listed!")
		}
	})
}

func TestSendKeysAndCapture(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		sessions, _ := srv.ListSessions()
		s := sessions[0]
		windows, _ := s.Windows()
		w := windows[0]
		panes, _ := w.Panes()
		p := panes[0]
		time.Sleep(2 * time.Second)
		if err := p.SendKeys("expr 12 \\* 88"); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Second)
		out, err := p.Capture()
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, line := range out {
			if strings.Contains(line, "1056") {
				found = true
			}
		}
		if !found {
			t.Fatal("Command did not run!")
		}
	})
}

func TestSendCommandAndCapture(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		sessions, _ := srv.ListSessions()
		s := sessions[0]
		windows, _ := s.Windows()
		w := windows[0]
		panes, _ := w.Panes()
		p := panes[0]
		time.Sleep(2 * time.Second)
		if err := p.SendCommand([]string{"bash", "-c", "\"expr 12 \\* 88\""}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Second)
		out, err := p.Capture()
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, line := range out {
			if strings.Contains(line, "1056") {
				found = true
			}
		}
		if !found {
			t.Fatal("Command did not run!")
		}
	})
}

func TestPaneOptions(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		name := "test_session_name"
		s, _ := srv.New(name)
		err := s.SetOption("@hello", "world")
		if err != nil {
			t.Fatal(err)
		}
		windows, _ := s.Windows()
		panes, _ := windows[0].Panes()
		res, err := panes[0].GetOption("@hello")
		if err != nil {
			t.Fatal(err)
		}
		if res != "world" {
			t.Fatal("values do not match")
		}
	})
}

func TestPaneOptionsNotSet(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		name := "test_session_name"
		s, _ := srv.New(name)
		windows, _ := s.Windows()
		panes, _ := windows[0].Panes()
		_, err := panes[0].GetOption("@hello")
		if err == nil {
			t.Fatal("Should return an error")
		}
	})
}

func TestWorkingDirectory(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		dir := t.TempDir()
		name := "test_sess_name"
		s, _ := srv.New(name, dir)
    if _, err := s.NewWindow(); err != nil {
			t.Fatal(err)
		}
		if _, err := s.NewWindow(); err != nil {
			t.Fatal(err)
		}
		if _, err := s.NewWindow(); err != nil {
			t.Fatal(err)
		}
		windows, _ := s.Windows()
		if err := windows[0].Kill(); err != nil {
			t.Fatal(err)
		}
		windows, err = s.Windows()
		if err != nil {
			t.Error(err)
		}
		if len(windows) != 3 {
			t.Fatal("Should have 3 windows")
		}
		for _, w := range windows {
			panes, err := w.Panes()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := panes[0].Split(); err != nil {
				t.Fatal(err)
			}
			if _, err := panes[0].SplitHorizontally(); err != nil {
				t.Fatal(err)
			}
			if _, err := panes[0].Split(); err != nil {
				t.Fatal(err)
			}
			panes, err = w.Panes()
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range panes {
				if err := p.SendKeys("pwd"); err != nil {
					t.Fatal(err)
				}
			}
		}
		time.Sleep(10 * time.Second)
		for _, w := range windows {
			panes, err := w.Panes()
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range panes {
				out, err := p.Capture()
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(strings.Join(out, ""), dir) {
					t.Fatalf("Output is missing temp dir %v", strings.Join(out, ""))
				}
			}
		}
	})
}
