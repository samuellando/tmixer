package tmux

import (
	"errors"
	"fmt"
)

type windowId string

func parseWindowId(s string) (windowId, error) {
	if len(s) == 0 || s[0] != '@' {
		return "", fmt.Errorf("invalid window id: %q", s)
	}
	return windowId(s), nil
}

type Window struct {
	Id     windowId
	server *Server
}

func (w *Window) GetOption(key string) (string, error) {
	lines, err := w.server.command("show-options").withTargetWindow(w).withArgument(key).withFlag("-v").withFlag("-q").run()
	if len(lines) == 0 {
		return "", fmt.Errorf("Got no output: %w", err)
	}
	return lines[0], err
}

func (w *Window) Kill() error {
	_, err := w.server.command("kill-window").withTargetWindow(w).run()
	return err
}

func (w *Window) Select() error {
	_, err := w.server.command("select-window").withTargetWindow(w).run()
	return err
}

func (w *Window) Name() (string, error) {
	lines, err := w.server.command("display").withFlag("-p").withTargetWindow(w).withFormat("#{window_name}").run()
	if len(lines) == 0 {
		return "", errors.Join(fmt.Errorf("Got no name"), err)
	}
	return lines[0], err
}

func (w *Window) Panes() ([]*Pane, error) {
	lines, err := w.server.command("list-panes").withTargetWindow(w).withFormat("#{pane_id}").run()
	panes := make([]*Pane, 0)
	for _, line := range lines {
		id, err := parsePaneId(line)
		if err != nil {
			return nil, err
		}
		panes = append(panes, &Pane{Id: id, server: w.server})
	}
	return panes, err
}

func (w *Window) Link(s *Session) error {
	_, err := w.server.command("link-window").withWindow(w).withTargetSession(s).run()
	if err != nil {
		return err
	}
	// Wait for the window to show up
	return w.waitForLinkState(s, true)
}

func (w *Window) Unlink(s *Session) error {
	_, err := w.server.command("unlink-window").withTargetSessionWindow(s, w).run()
	if err != nil {
		return err
	}
	// Wait for the window to be gone
	return w.waitForLinkState(s, false)
}

func (w *Window) waitForLinkState(s *Session, expected bool) error {
	return timeout(func() (bool, error) {
		windows, err := s.Windows()
		if err != nil {
			return false, err
		}
		found := false
		for _, aw := range windows {
			if aw.Id == w.Id {
				found = true
				break
			}
		}
		return found == expected, nil
	})
}
