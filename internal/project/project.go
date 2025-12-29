package project

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/tmux"
)

type ProjectState int

const (
	PROJECT_STATUS_INACTIVE ProjectState = iota
	PROJECT_STATUS_ACTIVE
	PROJECT_STATUS_ATTACHED
)

var ErrSessionNotFound = errors.New("session not found")

type Project struct {
	Name       string
	Config     *config.ProjectConfig
	server     *tmux.Server
	fullConfig *config.Config
}

func (p *Project) TmuxSessionName() string {
	return strings.ReplaceAll(p.Name, ".", "_")
}

func (p *Project) Session() (*tmux.Session, error) {
	s, err := p.server.GetSessionWithName(p.TmuxSessionName())
	if err == tmux.ErrSessionNotFound {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("when getting project session: %w", err)
	}
	return s, nil
}

func (p *Project) LastActivity() (*time.Time, error) {
	session, err := p.Session()
	if err != nil {
		return nil, err
	}
	return session.LastActivity()
}

func (p *Project) Status() (ProjectState, error) {
	var activeSessionName string
	status := PROJECT_STATUS_INACTIVE
	client, err := p.server.ActiveClient()
	if err != nil && err != tmux.ErrNoActiveClient {
		return status, fmt.Errorf("when geting active client for status: %w", err)
	} else if err == nil {
		session, err := client.Session()
		if err != nil {
			return status, fmt.Errorf("when geting active session for status: %w", err)
		}
		activeSessionName, err = session.Name()
		if err != nil {
			return status, fmt.Errorf("when geting active session name status: %w", err)
		}
	}
	if activeSessionName == p.TmuxSessionName() {
		status = PROJECT_STATUS_ATTACHED
	} else if p.server.HasSessionWithName(p.TmuxSessionName()) {
		status = PROJECT_STATUS_ACTIVE
	}
	return status, nil
}

func (p *Project) Start() (*tmux.Session, error) {
	status, err := p.Status()
	if err != nil {
		return nil, fmt.Errorf("when getting project status: %w", err)
	}
	if status >= PROJECT_STATUS_ACTIVE {
		return p.Session()
	}
	s, err := p.server.New(p.TmuxSessionName(), p.Config.Directory)
	if err != nil {
		return nil, fmt.Errorf("when starting project: %w", err)
	}
	err = p.createWindows(s)
	if err != nil {
		return nil, fmt.Errorf("when starting project: %w", err)
	}
	return s, nil
}

func (p *Project) createWindows(s *tmux.Session) error {
	if p.Config == nil {
		return nil
	}
	if len(p.Config.Windows) > 0 {
		for _, windowConfig := range p.Config.Windows {
			w, err := s.NewWindow(windowConfig.Name)
			if err != nil {
				return fmt.Errorf("when creating window: %w", err)
			}
			err = setupWindow(w, windowConfig)
			if err != nil {
				return err
			}
		}
		windows, err := s.Windows()
		if err != nil {
			return fmt.Errorf("when listing windows: %w", err)
		}
		err = windows[0].Kill()
		if err != nil {
			return fmt.Errorf("when killing default window: %w", err)
		}
		err = windows[1].Select()
		if err != nil {
			return fmt.Errorf("when switching to the first window: %w", err)
		}
	}
	return nil
}

func setupWindow(w *tmux.Window, config config.WindowConfig) error {
	if config.Command == nil && len(config.Panes) == 0 {
		return nil
	}
	panes, err := w.Panes()
	if err != nil {
		return fmt.Errorf("when creating window, getting window pane: %w", err)
	}
	firstPane := panes[0]
	keepalive := false
	if config.Command != nil {
		keepalive = true
		err = firstPane.SendKeys(*config.Command)
		if err != nil {
			return fmt.Errorf("when creating window, sending command: %w", err)
		}
	}
	lastPane := firstPane
	for _, pane := range config.Panes {
		var p *tmux.Pane
		if pane.Split == nil || strings.ToLower((*pane.Split)[:1]) == "h" {
			p, err = lastPane.SplitHorizontally()
			if err != nil {
				return fmt.Errorf("when creating pane: %w", err)
			}
		} else {
			p, err = lastPane.Split()
			if err != nil {
				return fmt.Errorf("when creating pane: %w", err)
			}
		}
		if pane.Command != nil {
			err = p.SendKeys(*pane.Command)
			if err != nil {
				return fmt.Errorf("when creating pane, sending command: %w", err)
			}
		}
		lastPane = p
	}
	if !keepalive {
		err = firstPane.Kill()
		if err != nil {
			return fmt.Errorf("when creating window, killing the initial pane: %w", err)
		}
	}
	return nil
}

func (p *Project) Kill() error {
	session, err := p.Session()
	if err != nil {
		return fmt.Errorf("when killing the session: %w", err)
	}
	status, err := p.Status()
	if err != nil {
		return fmt.Errorf("when killing the session: %w", err)
	}
	if status == PROJECT_STATUS_ATTACHED {
		all, err := List(p.server, p.fullConfig)
		if err != nil {
			return fmt.Errorf("While listing projects before killing: %w", err)
		}
		err = sortProjects(p.fullConfig, all)
		if err != nil {
			return fmt.Errorf("While sorting projects before killing: %w", err)
		}
		for _, o := range all {
			if o.Name != p.Name {
				err = o.Switch()
				if err != nil {
					return fmt.Errorf("While switching project before killing: %w", err)
				}
				break
			}
		}
	}
	err = session.Kill()
	if err != nil {
		return fmt.Errorf("when killing the session: %w", err)
	}
	return nil
}

func (p *Project) Switch() error {
	session, err := p.Start()
	if err != nil {
		return fmt.Errorf("when starting the project for switching: %w", err)
	}
	client, err := p.server.ActiveClient()
	if err != nil {
		return fmt.Errorf("when geting active client for switch: %w", err)
	}
	err = client.Switch(session)
	if err != nil {
		return fmt.Errorf("when swtiching to the session: %w", err)
	}
	return p.RunSwitchCommands()
}

func (p *Project) RunSwitchCommands() error {
	if p.Config == nil {
		return nil
	}
	session, err := p.Session()
	if err != nil {
		return err
	}
	windows, err := session.Windows()
	if err != nil {
		return fmt.Errorf("when getting windows for switch commands: %w", err)
	}
	panes, err := windows[0].Panes()
	if err != nil {
		return fmt.Errorf("when getting panes for switch commands: %w", err)
	}
	pane := panes[0]
	for _, cmd := range p.Config.SwitchCommands {
		cmdPane, err := pane.SplitHorizontally()
		if err != nil {
			return fmt.Errorf("when splitting command pane: %w", err)
		}
		err = cmdPane.SendKeys(cmd + " && exit")
		if err != nil {
			return fmt.Errorf("when sending command to pane: %w", err)
		}
	}
	return nil
}

func (p *Project) Reset() (*tmux.Session, error) {
	temp, err := p.createTempSession()
	if err != nil {
		return nil, fmt.Errorf("when creating temp session: %w", err)
	}
	defer temp.Kill()
	session, err := p.Session()
	if err != nil {
		return nil, fmt.Errorf("when getting session to reset: %w", err)
	}
	windows, err := session.Windows()
	for _, window := range windows {
		err := window.Link(temp)
		if err != nil {
			return nil, fmt.Errorf("when linking window to temp session: %w", err)
		}
	}
	status, err := p.Status()
	if err != nil {
		return nil, fmt.Errorf("when checking status for reset: %w", err)
	}
	if status == PROJECT_STATUS_ATTACHED {
		client, err := p.server.ActiveClient()
		if err != nil {
			return nil, fmt.Errorf("when geting active client for reset: %w", err)
		}
		err = client.Switch(temp)
		if err != nil {
			return nil, fmt.Errorf("when switching to temp session: %w", err)
		}
	}
	err = p.Kill()
	if err != nil {
		return nil, fmt.Errorf("when killing for reset: %w", err)
	}
	s, err := p.Start()
	if err != nil {
		return nil, fmt.Errorf("when starting for reset: %w", err)
	}
	if status == PROJECT_STATUS_ATTACHED {
		err = p.Switch()
		if err != nil {
			return nil, fmt.Errorf("when switching back after reset: %w", err)
		}
	}
	return s, nil
}

func (p *Project) createTempSession() (*tmux.Session, error) {
	uuid := uuid.NewString()
	return p.server.New(uuid)
}
