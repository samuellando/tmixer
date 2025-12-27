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
	Name   string
	Config *config.ProjectConfig
	server *tmux.Server
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
	err = p.createStartupWindows(s)
	if err != nil {
		return nil, fmt.Errorf("when starting project: %w", err)
	}
	return s, nil
}

func (p *Project) createStartupWindows(s *tmux.Session) error {
	if p.Config == nil {
		return nil
	}
	if len(p.Config.StartupWindows) > 0 {
		for _, windowConfig := range p.Config.StartupWindows {
			w, err := s.NewWindow(windowConfig.Name)
			if err != nil {
				return fmt.Errorf("when creating window: %w", err)
			}
			if windowConfig.Command != "" {
				panes, err := w.Panes()
				if err != nil {
					return fmt.Errorf("when creating window, getting window pane: %w", err)
				}
				err = panes[0].SendKeys(windowConfig.Command)
				if err != nil {
					return fmt.Errorf("when creating window, sending command: %w", err)
				}
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

func (p *Project) Stop() error {
	session, err := p.Session()
	if err != nil {
		return fmt.Errorf("when stoping the session: %w", err)
	}
	err = session.Kill()
	if err != nil {
		return fmt.Errorf("when stoping the session: %w", err)
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
	err = p.Stop()
	if err != nil {
		return nil, fmt.Errorf("when stopping for reset: %w", err)
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
