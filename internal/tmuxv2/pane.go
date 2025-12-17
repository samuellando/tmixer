package tmuxv2

import (
	"fmt"
)

type Pane struct {
	Id string
	server *Server
}

func (p *Pane) Split() (*Pane, error) {
	lines, err := p.server.command("split-window").withTargetPane(p).withFlag("-v").withFormat("#{pane_id}").run()
	if len(lines) == 0 {
		return nil, fmt.Errorf("Got no window id %w", err)
	}
	return &Pane{Id: lines[0], server: p.server}, err
}

func (p *Pane) SplitHorizontally() (*Pane, error) {
	lines, err := p.server.command("split-window").withTargetPane(p).withFlag("-h").withFormat("#{pane_id}").run()
	if len(lines) == 0 {
		return nil, fmt.Errorf("Got no window id %w", err)
	}
	return &Pane{Id: lines[0], server: p.server}, err
}

func (p *Pane) SendKeys(keys string) error {
	_, err := p.server.command("send-keys").withTargetPane(p).withArgument(keys).run()
	return err
}

func (p *Pane) Capture() ([]string, error) {
	return p.server.command("capture-pane").withFlag("-p").withFlag("-e").withTargetPane(p).run()
}
