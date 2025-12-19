package tmuxv2

import "fmt"

type windowId string

func parseWindowId(s string) (windowId,error) {
	if len(s) == 0 || s[0] != '@' {
		return "", fmt.Errorf("invalid window id: %q", s)
	}
	return windowId(s), nil
}

type Window struct {
	Id windowId

	server *Server
}

func (w *Window) Kill() error {
	_, err := w.server.command("kill-window").withTargetWindow(w).run()
	return err
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
