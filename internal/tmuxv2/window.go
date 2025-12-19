package tmuxv2

type Window struct {
	Id string

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
		panes = append(panes, &Pane{Id: line, server: w.server})
	}
	return panes, err
}

func (w *Window) Link(s *Session) error {
	_, err := w.server.command("link-window").withWindow(w).withTargetSession(s).run()
	return err
}
