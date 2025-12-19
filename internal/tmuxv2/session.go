package tmuxv2

import (
	"errors"
	"fmt"
)

type sessionId string

func parseSessionId(s string) (sessionId, error) {
	if len(s) == 0 || s[0] != '$' {
		return "", fmt.Errorf("invalid session id: %q", s)
	}
	return sessionId(s), nil
}

type Session struct {
	Id     sessionId
	server *Server
}

func (srv *Server) New(name string) (*Session, error) {
	lines, err := srv.command("new-session").withSession(name).print().detached().withFormat("#{session_id}").run()
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("Got no session")
	}
	id, err := parseSessionId(lines[0])
	if err != nil {
		return nil, err
	}
	return &Session{Id: id, server: srv}, err
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
		id, err := parseSessionId(line)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, &Session{Id: id, server: srv})
	}
	return sessions, err
}

func (s *Session) Kill() error {
	_, err := s.server.command("kill-session").withTargetSession(s).run()
	return err
}

func (s *Session) NewWindow(dir, name, command string) (*Window, error) {
	lines, err := s.server.command("new-window").withTargetSession(s).withWorkingDirectory(dir).withName(name).withArgument(command).print().withFormat("#{window_id}").run()
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("Got no window")
	}
	id, err := parseWindowId(lines[0])
	if err != nil {
		return nil, err
	}
	return &Window{Id: id, server: s.server}, err
}

func (s *Session) Windows() ([]*Window, error) {
	lines, err := s.server.command("list-windows").withTargetSession(s).withFormat("#{window_id}").run()
	windows := make([]*Window, 0)
	for _, line := range lines {
		id, err := parseWindowId(line)
		if err != nil {
			return nil, err
		}
		windows = append(windows, &Window{Id: id, server: s.server})
	}
	return windows, err
}
