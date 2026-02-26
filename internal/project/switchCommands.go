package project

import (
	"context"
	"fmt"

	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/tmux"
)

// Runs the projects switch commands in it's active session.
// If no session exists, it returns ErrSessionNotFound.
func (p *Project) RunSwitchCommands(ctx context.Context) error {
	type projectRunSwitchCommandsEvent struct {
		Name      string         `json:"name"`
		SessionId tmux.SessionId `json:"sessionId,omitempty"`
		Commands  [][]string     `json:"commands,omitempty"`
		Errors    []string       `json:"errors,omitempty"`
	}
	event := &projectRunSwitchCommandsEvent{Name: p.Name}
	finish := log.Track(ctx, "projectRunSwitchCommandsEvent", event)
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
		cmdCopy := append([]string(nil), cmd...)
		err = cmdPane.SendCommand(append(cmdCopy, "&&", "exit"))
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			return fmt.Errorf("when sending command to pane: %w", err)
		}
	}
	return nil
}
