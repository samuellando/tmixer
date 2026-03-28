package project

import (
	"context"
	"fmt"

	"samuellando.com/tmixer/internal/log/v2"
	"samuellando.com/tmixer/internal/tmux"
)

// Switches the active tmux client to the project's tmux session,
// and runs the switch commands.
// If no session is running one will be started.
func (p *Project) Switch(ctx context.Context) (*tmux.Session, error) {
	logEvent := log.Track(ctx, "projectSwitch")
	defer logEvent.Done()
	logEvent.Log("name", p.Name)
	session, err := p.Start(ctx)
	if err != nil {
		logEvent.Error(err)
		return nil, fmt.Errorf("when starting the project for switching: %w", err)
	}
	logEvent.Log("sessionId", session.Id)
	client, err := p.server.ActiveClient()
	if err != nil {
		logEvent.Error(err)
		return nil, fmt.Errorf("when getting active client for switch: %w", err)
	}
	logEvent.Log("clientId", client.Id)
	err = client.Switch(session)
	if err != nil {
		logEvent.Error(err)
		return nil, fmt.Errorf("when switching to the session: %w", err)
	}
	return session, p.RunSwitchCommands(ctx)
}
