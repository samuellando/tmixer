package tmux

import (
	"fmt"
)

type paneId string

func parsePaneId(s string) (paneId, error) {
	if len(s) == 0 || s[0] != '%' {
		return "", fmt.Errorf("invalid pane id: %q", s)
	}
	return paneId(s), nil
}

type Pane struct {
	Id     paneId
	server *Server
}

func (p *Pane) GetOption(key string) (string, error) {
	lines, err := p.server.command("show-options").withTargetPane(p).withArgument(key).withFlag("-v").withFlag("-q").run()
	if len(lines) == 0 {
		return "", fmt.Errorf("Got no output: %w", err)
	}
	return lines[0], err
}

func (p *Pane) Split() (*Pane, error) {
	c := p.server.command("split-window").withTargetPane(p).withFlag("-h").print().withFormat("#{pane_id}")
	if dir, err := p.GetOption("@working_dir"); err == nil {
		c = c.withWorkingDirectory(dir)
	}
	lines, err := c.run()
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
	c := p.server.command("split-window").withTargetPane(p).withFlag("-v").print().withFormat("#{pane_id}")
	if dir, err := p.GetOption("@working_dir"); err == nil {
		c = c.withWorkingDirectory(dir)
	}
	lines, err := c.run()
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

func (p *Pane) Kill() error {
	_, err := p.server.command("kill-pane").withTargetPane(p).run()
	return err
}

func (p *Pane) SendKeys(keys string) error {
	_, err := p.server.command("send-keys").withTargetPane(p).withArgument(keys).withArgument("enter").run()
	return err
}

func (p *Pane) Capture() ([]string, error) {
	return p.server.command("capture-pane").withFlag("-p").withFlag("-e").withTargetPane(p).run()
}
