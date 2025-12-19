package tmuxv2

import (
	"fmt"
)

type paneId string

func parsePaneId(s string) (paneId, error)  {
	if len(s) == 0 || s[0] != '%' {
		return "", fmt.Errorf("invalid pane id: %q", s)
	}
	return paneId(s), nil
}

type Pane struct {
	Id paneId
	server *Server
}

func (p *Pane) Split() (*Pane, error) {
	lines, err := p.server.command("split-window").withTargetPane(p).withFlag("-v").print().withFormat("#{pane_id}").run()
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("Got no pane")
	}
	id, err := parsePaneId(lines[0])
	if err != nil {
		return nil, err
	}
	return &Pane{Id: id, server: p.server}, err
}

func (p *Pane) SplitHorizontally() (*Pane, error) {
	lines, err := p.server.command("split-window").withTargetPane(p).withFlag("-h").print().withFormat("#{pane_id}").run()
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("Got no pane")
	}
	id, err := parsePaneId(lines[0])
	if err != nil {
		return nil, err
	}
	return &Pane{Id: id, server: p.server}, err
}

func (p *Pane) SendKeys(keys string) error {
	_, err := p.server.command("send-keys").withTargetPane(p).withArgument(keys).withArgument("enter").run()
	return err
}

func (p *Pane) Capture() ([]string, error) {
	return p.server.command("capture-pane").withFlag("-p").withFlag("-e").withTargetPane(p).run()
}
