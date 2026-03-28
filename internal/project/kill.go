package project

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"samuellando.com/tmixer/internal/log/v2"
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
	logEvent := log.Track(ctx, "projectKill")
	logEvent.Log("name", p.Name)
	defer logEvent.Done()

	cleanup := func() error { return nil }

	status, err := p.Status()
	if err != nil {
		logEvent.Error(err)
		return cleanup, fmt.Errorf("when killing the session: %w", err)
	}
	logEvent.Log("initialStatus", status)
	session, err := p.Session()
	if err != nil {
		logEvent.Error(err)
		return cleanup, fmt.Errorf("when killing the session: %w", err)
	}
	logEvent.Log("sessionId", session.Id)
	if status == PROJECT_STATUS_ATTACHED {
		err = switchToBestProject(ctx, p)
		if err != nil {
			logEvent.Error(err)
			return cleanup, fmt.Errorf("when killing the session: %w", err)
		}
		name, err := randomlyRename(session)
		if err != nil {
			logEvent.Error(err)
			return cleanup, fmt.Errorf("when randomly renaming attached session: %w", err)
		}
		logEvent.Log("tempSessionName", name)
		return session.Kill, nil
	} else {
		err = session.Kill()
		if err != nil {
			logEvent.Error(err)
			return cleanup, fmt.Errorf("when killing the session: %w", err)
		}
		return cleanup, nil
	}
}

func switchToBestProject(ctx context.Context, p *Project) error {
	logEvent := log.Track(ctx, "switchToBestProject")
	defer logEvent.Done()
	logEvent.Log("name", p.Name)

	all, err := List(ctx, p.server, p.fullConfig)
	if err != nil {
		logEvent.Error(err)
		return fmt.Errorf("while listing projects for best switch: %w", err)
	}
	err = sortProjects(p.fullConfig, all)
	if err != nil {
		logEvent.Error(err)
		return fmt.Errorf("while sorting projects for best switch: %w", err)
	}
	sortResult := make([]string, 0)
	for _, o := range all {
		status, err := o.Status()
		if err != nil {
			logEvent.Error(err)
		}
		time, err := o.LastActivity()
		if err != nil && err != ErrSessionNotFound {
			logEvent.Error(err)
		}
		sortResult = append(sortResult, fmt.Sprintf("%d %v: %s", status, time, o.Name))
	}
	logEvent.Log("sortResult", sortResult)
	for _, o := range all {
		if o.Name != p.Name {
			logEvent.Log("selected", o.Name)
			_, err = o.Switch(ctx)
			if err != nil {
				logEvent.Error(err)
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
