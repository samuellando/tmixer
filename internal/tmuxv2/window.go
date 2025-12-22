package tmuxv2

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
	return err
}
