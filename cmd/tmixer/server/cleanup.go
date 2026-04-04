package server

import (
	"context"
	"errors"
	"time"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
)

func (s *server) cleanupStaleProjects(ctx context.Context) (*time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	logEvent := log.Track(ctx, "cleanupStaleProjects")
	projectsKilled := make([]string, 0)
	defer logEvent.Done()
	var errs error
	var next *time.Duration
	for _, tmux := range s.tmuxServers {
		conf := s.serverConfig[tmux]
		if conf == nil || conf.Ttl == nil {
			continue
		}
		// Reload the files
		config := &config.Config{ConfigFiles: conf.ConfigFiles}
		config.LoadFiles(ctx)
		ttl, err := time.ParseDuration(*config.Ttl)
		if err != nil {
			logEvent.Error(err)
			errs = errors.Join(errs, err)
			continue
		}
		projects, err := project.List(ctx, tmux, config)
		if err != nil {
			logEvent.Error(err)
			errs = errors.Join(errs, err)
			continue
		}
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
				} else {
					la, err := p.LastActivity()
					if err != nil {
						logEvent.Error(err)
						errs = errors.Join(errs, err)
					}
					deadline := ttl - time.Since(*la)
					if next == nil || deadline > *next {
						next = &deadline
					}
				}
			}
		}
	}
	logEvent.Log("projectsKilled", projectsKilled)
	return next, errs
}
