package projectv2

import (
	"fmt"
	"strings"
	"errors"

	"github.com/google/uuid"
	"samuellando.com/tmixer/internal/config"
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
	name string
	config *config.ProjectConfig
	server *tmuxv2.Server
}

func List(tmux *tmuxv2.Server, config *config.Config) ([]*Project, error) {
	sessionsMatched := make(map[string]bool)
	projects := make([]*Project, 0)
	for name, projectConfig := range config.Projects {
		project := &Project{name: name, config: projectConfig, server: tmux}
		projects = append(projects, project)
		sessionsMatched[project.tmuxSessionName()] = true
	}
	sessions, err := tmux.ListSessions()
	if err != nil {
		return projects, fmt.Errorf("when listing sessions for project list: %w", err)
	}
	for _, session := range sessions {
		name, err := session.Name()
		if err != nil {
			return projects, fmt.Errorf("when getting session name for project list: %w", err)
		}
		if _, ok := sessionsMatched[name]; !ok {
			project := &Project{name: name, server: tmux}
			projects = append(projects, project)
		}
	}
	return projects, nil
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
	session, err := p.server.ClientSession()
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
	if p.config != nil {
		return nil
	}
	for _, windowConfig := range p.config.StartupWindows {
		_, err := s.NewWindow(p.config.Directory, windowConfig.Name, windowConfig.Command)
		if err != nil {
			return fmt.Errorf("when creating window: %", err)
		}
	}
	windows, err := s.Windows()
	if err != nil {
		return fmt.Errorf("when listing windows: %", err)
	}
	err = windows[0].Kill()
	if err != nil {
		return fmt.Errorf("when killing default window: %", err)
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
	err = p.server.Switch(session)
	if err != nil {
		return fmt.Errorf("when swtiching to the session: %w")
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
	for _, cmd := range p.config.SwitchCommands {
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
	for _, window := range  windows {
		err := window.Link(temp)
		if err != nil {
			return fmt.Errorf("when linking window to temp session: %w", err)
		}
	}
	status, err := p.Status()
	if err != nil {
		return fmt.Errorf("when checking status for reset: %w")
	}
	if status == PROJECT_STATUS_ATTACHED {
		err = p.server.Switch(temp)
		if err != nil {
			return fmt.Errorf("when switching to temp session: %w")
		}
	}
	err = p.Stop()
	if err != nil {
		return fmt.Errorf("when stopping for reset: %w")
	}
	err = p.Start()
	if err != nil {
		return fmt.Errorf("when starting for reset: %w")
	}
	if status == PROJECT_STATUS_ATTACHED {
		err = p.Switch()
		if err != nil {
			return fmt.Errorf("when switching back after reset: %w")
		}
	}
	err = temp.Kill()
	if err != nil {
		return fmt.Errorf("when killing temp session: %w")
	}
	return nil
}

func (p *Project) createTempSession() (*tmuxv2.Session, error) {
	uuid := uuid.NewString()
	return p.server.New(uuid)
}

func (p *Project) tmuxSessionName() string {
	return strings.ReplaceAll(p.name, ".", "_")
}
