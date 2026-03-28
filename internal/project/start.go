package project

import (
	"context"
	"fmt"
	"strings"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/tmux"
)

// Starts a tmux session for the project, and creates all it's configured windows and panes.
// If a session for the project already exists, it is returned and nothing else is done.
func (p *Project) Start(ctx context.Context) (*tmux.Session, error) {
	logEvent := log.Track(ctx, "projectStart")
	defer logEvent.Done()
	logEvent.Log("name", p.Name)

	status, err := p.Status()
	if err != nil {
		err := fmt.Errorf("when getting project status: %w", err)
		logEvent.Error(err)
		return nil, err
	}

	logEvent.Log("initialStatus", status)
	if status >= PROJECT_STATUS_ACTIVE {
		session, err := p.Session()
		if err != nil {
			logEvent.Error(err)
			return nil, err
		}
		logEvent.Log("sessionId", session.Id)
		return session, nil
	}

	s, err := p.server.New(p.TmuxSessionName(), p.Config.Directory)
	if err != nil {
		err := fmt.Errorf("when starting project: %w", err)
		logEvent.Error(err)
		return nil, err
	}
	err = p.createWindows(s)
	if err != nil {
		err := fmt.Errorf("when starting project: %w", err)
		logEvent.Error(err)
		return nil, err
	}
	logEvent.Log("sessionId", s.Id)
	return s, nil
}

func (p *Project) createWindows(s *tmux.Session) error {
	if p.Config == nil {
		return nil
	}
	if len(p.Config.Windows) > 0 {
		err := createWindows(s, p.Config.Windows)
		if err != nil {
			return err
		}
	}
	return nil
}

func createWindows(s *tmux.Session, windows []config.WindowConfig) error {
	if len(windows) == 0 {
		return nil
	}
	// Get the original windows
	originalWindows, err := s.Windows()
	if err != nil {
		return fmt.Errorf("when getting session windows: %w", err)
	}
	// Create the windows
	var firstWindow *tmux.Window
	for _, windowConfig := range windows {
		w, err := s.NewWindow(windowConfig.Name)
		if firstWindow == nil {
			firstWindow = w
		}
		if err != nil {
			return fmt.Errorf("when creating window: %w", err)
		}
		err = createPanes(w, windowConfig)
		if err != nil {
			return err
		}
	}
	// Kill the original windows
	for _, w := range originalWindows {
		err = w.Kill()
		if err != nil {
			return fmt.Errorf("when killing original window: %w", err)
		}
	}
	// Select the first window
	return firstWindow.Select()
}

func createPanes(w *tmux.Window, config config.WindowConfig) error {
	if len(config.Command) == 0 && len(config.Panes) == 0 {
		return nil
	}
	panes, err := w.Panes()
	if err != nil {
		return fmt.Errorf("when creating window, getting window pane: %w", err)
	}
	firstPane := panes[0]
	// If the window has a command, we need to run that command, and keep the pane
	keepalive := false
	if len(config.Command) > 0 {
		keepalive = true
		err = firstPane.SendCommand(config.Command)
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
		if len(pane.Command) > 0 {
			err = p.SendCommand(pane.Command)
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
