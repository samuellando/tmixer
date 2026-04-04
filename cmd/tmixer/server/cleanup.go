package server

import (
	"context"
	"errors"

	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
)

func cleanupStaleProjects(ctx context.Context, projects []*project.Project) error {
	logEvent := log.Track(ctx, "cleanupStaleProjects")
	projectsKilled := make([]string, 0)
	defer logEvent.Done()
	var errs error
	for _, p := range projects {
		status, err := p.Status()
		if err != nil {
			logEvent.Error(err)
			errs = errors.Join(errs, err)
		}
		if status == project.PROJECT_STATUS_ACTIVE {
			if passed, _ := p.TtlPassed(); passed {
				_, err := p.Kill(ctx)
				if err != nil {
					logEvent.Error(err)
					errs = errors.Join(errs, err)
				} else {
					projectsKilled = append(projectsKilled, p.Name)
				}
			}
		}
	}
	logEvent.Log("projectsKilled", projectsKilled)
	return errs
}
