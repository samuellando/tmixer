package server

import (
	"context"
	"errors"

	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/tmux"
)

func cleanupStaleProjects(tmux *tmux.Server) (err error) {
	if tmux == nil {
		return nil
	}
	ctx, config, err := setup()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, log.Fatal(ctx, err))
		} else {
			err = log.Done(ctx)
		}
		err = errors.Join(err, log.Display(ctx))
	}()
	// Setup a logger for this process.
	logEvent := log.Track(ctx, "cleanupStaleProjects")
	defer logEvent.Done()
	if config.Ttl == nil {
		return nil
	}
	// Check all projects
	projects, err := project.List(ctx, tmux, config)
	if err != nil {
		logEvent.Error(err)
		return err
	}
	projectsKilled := make([]string, 0)
	for _, p := range projects {
		killed, err := killProjectIfStale(ctx, logEvent, p)
		if err != nil {
			logEvent.Error(err)
			return err
		}
		if killed {
			projectsKilled = append(projectsKilled, p.Name)
		}
	}
	logEvent.Log("projectsKilled", projectsKilled)
	return nil
}

func killProjectIfStale(ctx context.Context, logEvent log.Event, p *project.Project) (bool, error) {
	status, err := p.Status()
	if err != nil {
		return false, err
	}
	if status == project.PROJECT_STATUS_ACTIVE {
		if passed, err := p.TtlPassed(); passed {
			_, err := p.Kill(ctx)
			if err != nil {
				return false, err
			}
		} else if err != nil {
			logEvent.Error(err)
			return false, err
		}
	}
	return true, err
}
