package main

import (
	"errors"
)

type session struct {
	cleanupFuncs []func() error
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
