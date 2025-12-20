package projectv2

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"samuellando.com/tmixer/internal/configv2"
	"samuellando.com/tmixer/internal/tmuxv2"
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
	server *tmuxv2.Server
}

func (p *Project) Session() (*tmuxv2.Session, error) {
	sessions, err := p.server.ListSessions()
	if err != nil {
		return nil, fmt.Errorf("when listing sessions: %w", err)
	}
	for _, session := range sessions {
		name, err := session.Name()
		if err != nil {
			return nil, fmt.Errorf("when getting session name: %w", err)
		}
		if name == p.tmuxSessionName() {
			return session, nil
		}
	}
	return nil, ErrSessionNotFound
}

func (p *Project) Status() (ProjectState, error) {
	status := PROJECT_STATUS_INACTIVE
	client, err := p.server.ActiveClient()
	if err != nil {
		return status, fmt.Errorf("when geting active client for status: %w", err)
	}
	session, err := client.Session()
	if err != nil {
		return status, fmt.Errorf("when geting active session for status: %w", err)
	}
	name, err := session.Name()
	if err != nil {
		return status, fmt.Errorf("when geting active session name status: %w", err)
	}
	if name == p.tmuxSessionName() {
		status = PROJECT_STATUS_ATTACHED
	} else if p.server.HasSessionWithName(p.tmuxSessionName()) {
		status = PROJECT_STATUS_ACTIVE
	}
	return status, nil
}

func (p *Project) Start() error {
	status, err := p.Status()
	if err != nil {
		return fmt.Errorf("when getting project status: %w", err)
	}
	if status == PROJECT_STATUS_INACTIVE {
		return nil
	}
	s, err := p.server.New(p.tmuxSessionName())
	if err != nil {
		return fmt.Errorf("when starting project: %w", err)
	}
	err = p.createStartupWindows(s)
	if err != nil {
		return fmt.Errorf("when starting project: %w", err)
	}
	return nil
}

func (p *Project) createStartupWindows(s *tmuxv2.Session) error {
	if p.Config != nil {
		return nil
	}
	for _, windowConfig := range p.Config.StartupWindows {
		_, err := s.NewWindow(p.Config.Directory, windowConfig.Name, windowConfig.Command)
		if err != nil {
			return fmt.Errorf("when creating window: %w", err)
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
	err := p.Start()
	if err == ErrSessionNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("when starting the project for switching: %w", err)
	}
	session, err := p.Session()
	if err != nil {
		return fmt.Errorf("when getting the session for switching: %w", err)
	}
	client, err := p.server.ActiveClient()
	if err != nil {
		return fmt.Errorf("when geting active client for switch: %w", err)
	}
	err = client.Switch(session)
	if err != nil {
		return fmt.Errorf("when swtiching to the session: %w", err)
	}
	return p.runSwitchCommands(session)
}

func (p *Project) runSwitchCommands(session *tmuxv2.Session) error {
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
		err = cmdPane.SendKeys(cmd)
		if err != nil {
			return fmt.Errorf("when sending command to pane: %w", err)
		}
	}
	return nil
}

func (p *Project) Reset() error {
	temp, err := p.createTempSession()
	if err != nil {
		return fmt.Errorf("when creating temp session: %w", err)
	}
	session, err := p.Session()
	if err != nil {
		return fmt.Errorf("when getting session to reset: %w", err)
	}
	windows, err := session.Windows()
	for _, window := range windows {
		err := window.Link(temp)
		if err != nil {
			return fmt.Errorf("when linking window to temp session: %w", err)
		}
	}
	status, err := p.Status()
	if err != nil {
		return fmt.Errorf("when checking status for reset: %w", err)
	}
	if status == PROJECT_STATUS_ATTACHED {
		client, err := p.server.ActiveClient()
		if err != nil {
			return fmt.Errorf("when geting active client for reset: %w", err)
		}
		err = client.Switch(temp)
		if err != nil {
			return fmt.Errorf("when switching to temp session: %w", err)
		}
	}
	err = p.Stop()
	if err != nil {
		return fmt.Errorf("when stopping for reset: %w", err)
	}
	err = p.Start()
	if err != nil {
		return fmt.Errorf("when starting for reset: %w", err)
	}
	if status == PROJECT_STATUS_ATTACHED {
		err = p.Switch()
		if err != nil {
			return fmt.Errorf("when switching back after reset: %w", err)
		}
	}
	err = temp.Kill()
	if err != nil {
		return fmt.Errorf("when killing temp session: %w", err)
	}
	return nil
}

func (p *Project) createTempSession() (*tmuxv2.Session, error) {
	uuid := uuid.NewString()
	return p.server.New(uuid)
}

func (p *Project) tmuxSessionName() string {
	return strings.ReplaceAll(p.Name, ".", "_")
}
