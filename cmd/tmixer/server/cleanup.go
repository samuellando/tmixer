package server

import (
	"context"
	"errors"
	"time"

	"samuellando.com/tmixer/cmd/tmixer/options"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
)

func (s *server) monitorAndCleanupStaleProjects() error {
	t := time.NewTicker(time.Minute)
	for {
		err := s.cleanupStaleProjects()
		if err != nil {
			return err
		}
		<-t.C
	}
}

func (s *server) cleanupStaleProjects() (err error) {
	if s.tmux == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Setup a logger for this process.
	ctx := log.ContextLogger(context.Background())
	defer func() {
		if err != nil {
			err = log.Done(ctx)
		} else {
			err = errors.Join(err, log.Fatal(ctx, err))
		}
	}()
	logEvent := log.Track(ctx, "cleanupStaleProjects")
	projectsKilled := make([]string, 0)
	defer logEvent.Done()
	// Reload the configs
	config := options.DEFAULT_CONFIG
	err = config.LoadFiles(ctx)
	if err != nil {
		logEvent.Error(err)
		return err
	}
	if config.Ttl == nil {
		return nil
	}
	// Check all projects
	projects, err := project.List(ctx, s.tmux, &config)
	if err != nil {
		logEvent.Error(err)
		return err
	}
	for _, p := range projects {
		status, err := p.Status()
		if err != nil {
			logEvent.Error(err)
			return err
		}
		if status == project.PROJECT_STATUS_ACTIVE {
			if passed, err := p.TtlPassed(); passed {
				_, err := p.Kill(ctx)
				if err != nil {
					logEvent.Error(err)
					return err
				} else {
					projectsKilled = append(projectsKilled, p.Name)
				}
			} else if err != nil {
				logEvent.Error(err)
				return err
			}
		}
	}
	logEvent.Log("projectsKilled", projectsKilled)
	return nil
}
