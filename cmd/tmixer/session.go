package main

import (
	"errors"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/project"
)

type session struct {
	cleanupFuncs []func() error
	projects     []*project.Project
	config       *config.Config
}

func newSession() *session {
	return &session{cleanupFuncs: make([]func() error, 0)}
}

func (s *session) close() error {
	return s.runCleanupFunctions()
}

func (s *session) addCleanup(f func() error) {
	s.cleanupFuncs = append(s.cleanupFuncs, f)
}

func (s *session) runCleanupFunctions() error {
	var err error = nil
	for _, f := range s.cleanupFuncs {
		if f != nil {
			err = errors.Join(f())
		}
	}
	return err
}
