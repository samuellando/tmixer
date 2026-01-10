package project

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/tmux"
)

type ProjectStatus int

const (
	PROJECT_STATUS_INACTIVE ProjectStatus = iota
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

// Returns one of PROJECT_STATUS_INACTIVE, PROJECT_STATUS_ACTIVE, and PROJECT_STATUS_ATTACHED
// A project is inactive if it has no running session, active if it does and attached
// if the currently running client is attached to it's session.
func (p *Project) Status() (ProjectStatus, error) {
	var activeSessionName string
	status := PROJECT_STATUS_INACTIVE
	client, err := p.server.ActiveClient()
	if err != nil && err != tmux.ErrNoActiveClient {
		return status, fmt.Errorf("when getting active client for status: %w", err)
	} else if err == nil {
		session, err := client.Session()
		if err != nil {
			return status, fmt.Errorf("when getting active session for status: %w", err)
		}
		activeSessionName, err = session.Name()
		if err != nil {
			return status, fmt.Errorf("when getting active session name status: %w", err)
		}
	}
	if activeSessionName == p.TmuxSessionName() {
		status = PROJECT_STATUS_ATTACHED
	} else if p.server.HasSessionWithName(p.TmuxSessionName()) {
		status = PROJECT_STATUS_ACTIVE
	}
	return status, nil
}

// Return the project's tmux session.
// If there is no session returns ErrSessionNotFound
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

// Return the tmux session Name for the project. Since not all project names are
// compatible with tmux, this will return a compatible name.
func (p *Project) TmuxSessionName() string {
	s := p.Name
	b := strings.Builder{}
	for i := 0; i < len(s); i++ {
		if s[i] == '#' {
			if i < len(s)-1 {
				if s[i+1] == '#' {
					// ##
					b.WriteByte('#')
					i++
					continue
				}
				if s[i+1] == '}' {
					// #}
					b.WriteByte('}')
					i++
					continue
				}
				if s[i+1] == '}' {
					// #,
					b.WriteByte(',')
					i++
					continue
				}
			}
		}
		b.WriteByte(s[i])
	}
	s = b.String()
	out := ""
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch r {
		case '.':
			out += `_`
		case ':':
			out += `_`
		case '\a':
			out += `\a`
		case '\b':
			out += `\b`
		case '\f':
			out += `\f`
		case '\n':
			out += `\n`
		case '\r':
			out += `\r`
		case '\t':
			out += `\t`
		case '\v':
			out += `\v`
		case '\\':
			out += `\\`
		case '"':
			out += `"`
		default:
			if r >= 0x20 && r <= 0x7e { // printable ASCII
				out += string(r)
			} else if (r >= 0x1 && r <= 0x7f) || r == utf8.RuneError {
				// Control characters
				out += fmt.Sprintf(`\%03o`, s[i])
			} else {
				out += string(r)
			}
		}
		i += size
	}
	return out
}

// Return the last activity time of the projects session.
// If no session exists will return ErrSessionNotFound
func (p *Project) LastActivity() (*time.Time, error) {
	session, err := p.Session()
	if err != nil {
		return nil, err
	}
	return session.LastActivity()
}

// Starts a tmux session for the project, and creates all it's configured windows and panes.
// If a session for the project already exists, it is returned and nothing else is done.
func (p *Project) Start(ctx context.Context) (*tmux.Session, error) {
	type projectStartEvent struct {
		Name          string         `json:"name"`
		InitialStatus ProjectStatus  `json:"initialStatus"`
		SessionId     tmux.SessionId `json:"sessionId"`
		Errors        []string       `json:"errors,omitempty"`
	}
	event := &projectStartEvent{Name: p.Name}
	finish := log.Track(ctx, "projectStartEvent", event)
	defer finish()

	status, err := p.Status()
	if err != nil {
		err := fmt.Errorf("when getting project status: %w", err)
		event.Errors = append(event.Errors, err.Error())
		return nil, err
	}

	event.InitialStatus = status
	if status >= PROJECT_STATUS_ACTIVE {
		session, err := p.Session()
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			return nil, err
		}
		event.SessionId = session.Id
		return session, nil
	}

	s, err := p.server.New(p.TmuxSessionName(), p.Config.Directory)
	if err != nil {
		err := fmt.Errorf("when starting project: %w", err)
		event.Errors = append(event.Errors, err.Error())
		return nil, err
	}
	err = p.createWindows(s)
	if err != nil {
		err := fmt.Errorf("when starting project: %w", err)
		event.Errors = append(event.Errors, err.Error())
		return nil, err
	}
	event.SessionId = s.Id
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
			err = createPanes(w, windowConfig)
			if err != nil {
				return err
			}
		}
		// Kill the default window.
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

func createPanes(w *tmux.Window, config config.WindowConfig) error {
	if config.Command == nil && len(config.Panes) == 0 {
		return nil
	}
	panes, err := w.Panes()
	if err != nil {
		return fmt.Errorf("when creating window, getting window pane: %w", err)
	}
	firstPane := panes[0]
	// If the window has a command, we need to run that command, and keep the pane
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

// Kills the projects tmux session.
// If no project session exists returns ErrSessionNotFound
// If the session being killed is attached, will  switch to another project based on:
// 1. If there is other active sessions, it will switch to the one with the most recent activity.
// 2. Otherwise, it will try to start and switch to the default project
// 3. If no default project is set, it will switch to a random project.
// 4. It will exit tmux if there are no other configured projects.
func (p *Project) Kill(ctx context.Context) (func() error, error) {
	type projectKillEvent struct {
		Name            string         `json:"name"`
		InitialStatus   ProjectStatus  `json:"initialStatus"`
		SessionId       tmux.SessionId `json:"sessionId,omitempty"`
		TempSessionName string         `json:"tempSessionName,omitempty"`
		Errors          []string       `json:"errors,omitempty"`
	}
	event := &projectKillEvent{Name: p.Name}
	finish := log.Track(ctx, "projectKillEvent", event)
	defer finish()

	cleanup := func() error { return nil }

	status, err := p.Status()
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return cleanup, fmt.Errorf("when killing the session: %w", err)
	}
	event.InitialStatus = status
	session, err := p.Session()
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return cleanup, fmt.Errorf("when killing the session: %w", err)
	}
	event.SessionId = session.Id
	if status == PROJECT_STATUS_ATTACHED {
		err = switchToBestProject(ctx, p)
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			return cleanup, fmt.Errorf("when killing the session: %w", err)
		}
		name, err := randomlyRename(session)
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			return cleanup, fmt.Errorf("when randomly renaming attached session: %w", err)
		}
		event.TempSessionName = name
		return session.Kill, nil
	} else {
		err = session.Kill()
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			return cleanup, fmt.Errorf("when killing the session: %w", err)
		}
		return cleanup, nil
	}
}

func switchToBestProject(ctx context.Context, p *Project) error {
	type switchToBestProjectEvent struct {
		Name       string   `json:"name"`
		SortResult []string `json:"sortResult"`
		Selected   string   `json:"selected"`
		Errors     []string `json:"errors,omitempty"`
	}
	event := &switchToBestProjectEvent{Name: p.Name}
	finish := log.Track(ctx, "switchToBestProject", event)
	defer finish()

	all, err := List(ctx, p.server, p.fullConfig)
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return fmt.Errorf("while listing projects for best switch: %w", err)
	}
	err = sortProjects(p.fullConfig, all)
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return fmt.Errorf("while sorting projects for best switch: %w", err)
	}
	for _, o := range all {
		status, err := o.Status()
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
		}
		time, err := o.LastActivity()
		if err != nil && err != ErrSessionNotFound {
			event.Errors = append(event.Errors, err.Error())
		}
		event.SortResult = append(event.SortResult, fmt.Sprintf("%d %v: %s", status, time, o.Name))
	}
	for _, o := range all {
		if o.Name != p.Name {
			event.Selected = o.Name
			_, err = o.Switch(ctx)
			if err != nil {
				event.Errors = append(event.Errors, err.Error())
				return err
			}
			break
		}
	}
	return nil
}

// Switches the active tmux client to the project's tmux session,
// and runs the switch commands.
// If no session is running one will be started.
func (p *Project) Switch(ctx context.Context) (*tmux.Session, error) {
	type projecSwitchEvent struct {
		Name      string         `json:"name"`
		SessionId tmux.SessionId `json:"sessionId"`
		ClientId  tmux.ClientId  `json:"clientId"`
		Errors    []string       `json:"errors,omitempty"`
	}
	event := &projecSwitchEvent{Name: p.Name}
	finish := log.Track(ctx, "projectSwitchEvent", event)
	defer finish()
	session, err := p.Start(ctx)
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return nil, fmt.Errorf("when starting the project for switching: %w", err)
	}
	event.SessionId = session.Id
	client, err := p.server.ActiveClient()
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return nil, fmt.Errorf("when getting active client for switch: %w", err)
	}
	event.ClientId = client.Id
	err = client.Switch(session)
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return nil, fmt.Errorf("when switching to the session: %w", err)
	}
	return session, p.RunSwitchCommands(ctx)
}

// Runs the projects switch commands in it's active session.
// If no session exists, it returns ErrSessionNotFound.
func (p *Project) RunSwitchCommands(ctx context.Context) error {
	type projecRunSwitchCommandsEvent struct {
		Name      string         `json:"name"`
		SessionId tmux.SessionId `json:"sessionId,omitempty"`
		Commands  []string       `json:"commands,omitempty"`
		Errors    []string       `json:"errors,omitempty"`
	}
	event := &projecRunSwitchCommandsEvent{Name: p.Name}
	finish := log.Track(ctx, "projecRunSwitchCommandsEvent", event)
	defer finish()

	if p.Config == nil {
		return nil
	}
	session, err := p.Session()
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return err
	}
	event.SessionId = session.Id
	windows, err := session.Windows()
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return fmt.Errorf("when getting windows for switch commands: %w", err)
	}
	panes, err := windows[0].Panes()
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return fmt.Errorf("when getting panes for switch commands: %w", err)
	}
	pane := panes[0]
	for _, cmd := range p.Config.SwitchCommands {
		cmdPane, err := pane.SplitHorizontally()
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			return fmt.Errorf("when splitting command pane: %w", err)
		}
		event.Commands = append(event.Commands, cmd)
		err = cmdPane.SendKeys(cmd + " && exit")
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			return fmt.Errorf("when sending command to pane: %w", err)
		}
	}
	return nil
}

// Resets a projects session to it's initial configured state.
// Effectively calling stop and start on it.
//
// Returns a the new session and a cleanup function that the caller must call.
//
// If the session is currently attached, it would imediently kill the session
// but will instead randomly rename it. The cleanup function will delete it.
// this is done so the running process is not killed.
//
// When reseting an attached session, the switch commands will be rerun.
//
// If no session exists, will return ErrSessionNotFound.
func (p *Project) Reset(ctx context.Context) (*tmux.Session, func() error, error) {
	type projecResetEvent struct {
		Name             string         `json:"name"`
		InitialSessionId tmux.SessionId `json:"initialSessionId,omitempty"`
		TempSessionName  string         `json:"tempSessionName,omitempty"`
		InitialStatus    ProjectStatus  `json:"initialStatus"`
		ClientId         tmux.ClientId  `json:"clientId,omitempty"`
		FinalSessionId   tmux.SessionId `json:"finalSessionId,omitempty"`
		Errors           []string       `json:"errors,omitempty"`
	}
	event := &projecResetEvent{Name: p.Name}
	finish := log.Track(ctx, "projecResetEvent", event)
	defer finish()

	cleanup := func() error { return nil }

	status, err := p.Status()
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return nil, cleanup, fmt.Errorf("when checking status for reset: %w", err)
	}
	event.InitialStatus = status
	session, err := p.Session()
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return nil, cleanup, fmt.Errorf("when getting session to reset: %w", err)
	}
	event.InitialSessionId = session.Id

	// get rid of the current session.
	if status == PROJECT_STATUS_ATTACHED {
		client, err := p.server.ActiveClient()
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			return nil, cleanup, fmt.Errorf("when geting active client for reset: %w", err)
		}
		event.ClientId = client.Id
		name, err := randomlyRename(session)
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			return nil, cleanup, fmt.Errorf("when temporarly renaming current session: %w", err)
		}
		event.TempSessionName = name
		cleanup = session.Kill
	} else {
		_, err = p.Kill(ctx)
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			return nil, cleanup, fmt.Errorf("when killing for reset: %w", err)
		}
	}

	// Start a new session
	s, err := p.Start(ctx)
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return nil, cleanup, fmt.Errorf("when starting for reset: %w", err)
	}
	event.FinalSessionId = s.Id
	if status == PROJECT_STATUS_ATTACHED {
		_, err = p.Switch(ctx)
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			return nil, cleanup, fmt.Errorf("when switching back after reset: %w", err)
		}
	}
	return s, cleanup, nil
}

func randomlyRename(s *tmux.Session) (string, error) {
	uuid := uuid.NewString()
	return uuid, s.Rename(uuid)
}
