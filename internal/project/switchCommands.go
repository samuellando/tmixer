package project

import (
	"context"
	"fmt"

	"samuellando.com/tmixer/internal/log"
)

// Runs the projects switch commands in it's active session.
// If no session exists, it returns ErrSessionNotFound.
func (p *Project) RunSwitchCommands(ctx context.Context) error {
	logEvent := log.Track(ctx, "projectRunSwitchCommands")
	defer logEvent.Done()
	logEvent.Log("name", p.Name)

	if p.Config == nil {
		return nil
	}
	session, err := p.Session()
	if err != nil {
		logEvent.Error(err)
		return err
	}
	logEvent.Log("sessionId", session.Id)
	windows, err := session.Windows()
	if err != nil {
		logEvent.Error(err)
		return fmt.Errorf("when getting windows for switch commands: %w", err)
	}
	panes, err := windows[0].Panes()
	if err != nil {
		logEvent.Error(err)
		return fmt.Errorf("when getting panes for switch commands: %w", err)
	}
	pane := panes[0]
	commands := make([][]string, 0)
	for _, cmd := range p.Config.SwitchCommands {
		cmdPane, err := pane.SplitHorizontally()
		if err != nil {
			logEvent.Error(err)
			return fmt.Errorf("when splitting command pane: %w", err)
		}
		cmdCopy := append([]string(nil), cmd...)
		err = cmdPane.SendCommand(append(cmdCopy, "&&", "exit"))
		if err != nil {
			logEvent.Error(err)
			return fmt.Errorf("when sending command to pane: %w", err)
		}
		commands = append(commands, cmd)
	}
	logEvent.Log("commands", commands)
	return nil
}
