package project

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/tmux"
)

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

func randomlyRename(s *tmux.Session) (string, error) {
	uuid := uuid.NewString()
	return uuid, s.Rename(uuid)
}
