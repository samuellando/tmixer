package project

import (
	"context"
	"fmt"

	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/tmux"
)

// Switches the active tmux client to the project's tmux session,
// and runs the switch commands.
// If no session is running one will be started.
func (p *Project) Switch(ctx context.Context) (*tmux.Session, error) {
	type projectSwitchEvent struct {
		Name      string         `json:"name"`
		SessionId tmux.SessionId `json:"sessionId"`
		ClientId  tmux.ClientId  `json:"clientId"`
		Errors    []string       `json:"errors,omitempty"`
	}
	event := &projectSwitchEvent{Name: p.Name}
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
