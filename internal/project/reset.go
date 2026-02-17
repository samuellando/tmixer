package project

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/tmux"
)

// Resets a projects session to it's initial configured state.
//
// Returns a cleanup function that the caller must call.
//
// Links all current windows to a temp session, unlinks them from the session
// and recreates them.
//
// When reseting an attached session, the switch commands will be rerun.
//
// If no session exists, will return ErrSessionNotFound.
// If there is no project config, will return an error.
func (p *Project) Reset(ctx context.Context) (func() error, error) {
	type projecResetEvent struct {
		Name            string         `json:"name"`
		SessionId       tmux.SessionId `json:"sessionId,omitempty"`
		TempSessionName string         `json:"tempSessionName,omitempty"`
		InitialStatus   ProjectStatus  `json:"initialStatus"`
		Errors          []string       `json:"errors,omitempty"`
	}
	event := &projecResetEvent{Name: p.Name}
	finish := log.Track(ctx, "projecResetEvent", event)
	defer finish()

	cleanup := func() error { return nil }

	if p.Config == nil {
		err := fmt.Errorf("Session %s has no project config to reset to", p.Name)
		event.Errors = append(event.Errors, err.Error())
		return cleanup, err
	}

	status, err := p.Status()
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return cleanup, fmt.Errorf("when checking status for reset: %w", err)
	}
	event.InitialStatus = status
	session, err := p.Session()
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return cleanup, fmt.Errorf("when getting session to reset: %w", err)
	}
	event.SessionId = session.Id

	tempName, temp, err := createTempSession(p.server)
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return cleanup, fmt.Errorf("when creating temp session: %w", err)
	}
	event.TempSessionName = tempName
	cleanup = temp.Kill

	// Move all the originalWindows to the temp session
	originalWindows, err := session.Windows()
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return cleanup, fmt.Errorf("When getting session windows: %w", err)
	}
	for _, w := range originalWindows {
		err = w.Link(temp)
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			return cleanup, fmt.Errorf("When linking window to temp session: %w", err)
		}
	}

	windows := p.Config.Windows
	if len(p.Config.Windows) == 0 {
		windows = []config.WindowConfig{{Name: "sh"}}
	}

	err = resetWindows(session, originalWindows, windows)
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return cleanup, fmt.Errorf("when reseting windows: %w", err)
	}

	if status == PROJECT_STATUS_ATTACHED {
		err = p.RunSwitchCommands(ctx)
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			return cleanup, fmt.Errorf("when running switch commands after reset: %w", err)
		}
	}
	return cleanup, nil
}

func resetWindows(session *tmux.Session, originalWindows []*tmux.Window, windows []config.WindowConfig) error {
	// Need to kepp at least one window before unlinking
	_, err := session.NewWindow("temp")
	if err != nil {
		return fmt.Errorf("When creating a temp window: %w", err)
	}
	// Unlink the original windows
	for _, w := range originalWindows {
		err = w.Unlink(session)
		if err != nil {
			return fmt.Errorf("When unlinking window from session: %w", err)
		}
	}
	// Re-create the windows
	err = createWindows(session, windows)
	if err != nil {
		return fmt.Errorf("When recreating windows: %w", err)
	}
	return nil
}

func createTempSession(srv *tmux.Server) (string, *tmux.Session, error) {
	uuid := uuid.NewString()
	s, err := srv.New(uuid)
	return uuid, s, err
}
