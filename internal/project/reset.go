package project

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/tmux"
)

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
