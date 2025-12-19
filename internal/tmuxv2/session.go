package tmuxv2

import (
	"errors"
	"fmt"
)

type Session struct {
	Id      string
	server *Server
}

func (srv *Server) New(name string) (*Session, error) {
	lines, err := srv.command("new-session").withSession(name).print().detached().withFormat("#{session_id}").run()
	if len(lines) == 0 {
		return nil, fmt.Errorf("Got no session id %w", err)
	}
	return &Session{Id: lines[0], server: srv}, err
}

func (s *Session) Name() (string, error) {
	lines, err := s.server.command("display").withFlag("-p").withTargetSession(s).withFormat("#{session_name}").run()
	if len(lines) == 0 {
		return "", errors.Join(fmt.Errorf("Got no name"), err)
	}
	return lines[0], err
}

func (srv *Server) ListSessions() ([]*Session, error) {
	lines, err := srv.command("list-sessions").withFormat("#{session_id}").run()
	sessions := make([]*Session, 0)
	for _, line := range lines {
		sessions = append(sessions, &Session{Id: line, server: srv})
	}
	return sessions, err
}

func (s *Session) Kill() error {
	_, err := s.server.command("kill-session").withTargetSession(s).run()
	return err
}

func (srv *Server) KillSessionWithName(s string) error {
	_, err := srv.command("kill-session").withTargetSessionName(s).run()
	return err
}

func (s *Session) NewWindow(dir, name, command string) (*Window, error) {
	lines, err := s.server.command("new-window").withWorkingDirectory(dir).withName(name).withArgument(command).print().withFormat("#{window_id}").run()
	if len(lines) == 0 {
		return nil, fmt.Errorf("Got no window id %w", err)
	}
	return &Window{Id: lines[0], server: s.server}, err
}

func (s *Session) Windows() ([]*Window, error) {
	lines, err := s.server.command("list-windows").withTargetSession(s).withFormat("#{window_id}").run()
	windows := make([]*Window, 0)
	for _, line := range lines {
		windows = append(windows, &Window{Id: line, server: s.server})
	}
	return windows, err
}
