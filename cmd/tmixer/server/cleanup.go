package server

import (
	"context"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/tmux"
)

func cleanupStaleProjects(ctx context.Context, config *config.Config, tmux *tmux.Server) (err error) {
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
	for _, p := range projects {
		err := killProjectIfStale(ctx, logEvent, p)
		if err != nil {
			return err
		}
	}
	return nil
}

func killProjectIfStale(ctx context.Context, logEvent log.Event, p *project.Project) error {
	status, err := p.Status()
	if err != nil {
		logEvent.Error(err)
		return err
	}
	projectsKilled := make([]string, 0)
	if status == project.PROJECT_STATUS_ACTIVE {
		if passed, err := p.TtlPassed(); passed {
			_, err := p.Kill(ctx)
			if err != nil {
				logEvent.Error(err)
				return err
			}
			projectsKilled = append(projectsKilled, p.Name)
		} else if err != nil {
			logEvent.Error(err)
			return err
		}
	}
	logEvent.Log("projectsKilled", projectsKilled)
	return nil
}
