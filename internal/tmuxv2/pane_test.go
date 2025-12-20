package tmuxv2_test

import (
	"strings"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmuxv2"
)

func TestSplit(t *testing.T) {
	f := func(tmux *tmuxv2.Server) {
		sessions, _ := tmux.ListSessions()
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
		panes, _ = w.Panes()
		if len(panes) != 2 {
			t.Fatal("Should have two panes")
		}
		panes, _ = w.Panes()
		found := false
		for _, pane := range panes {
			if pane.Id == p2.Id {
				found = true
			}
		}
		if !found {
			t.Fatal("Pane not listed!")
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestSplitHorizontally(t *testing.T) {
	f := func(tmux *tmuxv2.Server) {
		sessions, _ := tmux.ListSessions()
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
		panes, _ = w.Panes()
		if len(panes) != 2 {
			t.Fatal("Should have two panes")
		}
		panes, _ = w.Panes()
		found := false
		for _, pane := range panes {
			if pane.Id == p2.Id {
				found = true
			}
		}
		if !found {
			t.Fatal("Pane not listed!")
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestSendKeysAndCapture(t *testing.T) {
	f := func(tmux *tmuxv2.Server) {
		sessions, _ := tmux.ListSessions()
		s := sessions[0]
		windows, _ := s.Windows()
		w := windows[0]
		panes, _ := w.Panes()
		p := panes[0]
		time.Sleep(2 * time.Second)
		p.SendKeys("expr 12 \\* 88")
		time.Sleep(2 * time.Second)
		out, _ := p.Capture()
		found := false
		for _, line := range out {
			if strings.Contains(line, "1056") {
				found = true
			}
		}
		if !found {
			t.Fatal("Command did not run!")
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}
