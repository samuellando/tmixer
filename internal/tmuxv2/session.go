package tmuxv2

import (
	"fmt"

	"samuellando.com/tmixer/internal/project"
)

type Session struct {
	Id      string
	server *Server
	Project *project.Project
}

func (srv *Server) New(name string) (*Session, error) {
	lines, err := srv.command("new-session").withSession(name).print().detached().withFormat("#{session_id}").run()
	if len(lines) == 0 {
		return nil, fmt.Errorf("Got no session id %w", err)
	}
	return &Session{Id: lines[0], server: srv}, err
}

func (srv *Server) ListSessions() ([]*Session, error) {
	lines, err := srv.command("list-sessions").withFormat("#{session_id}").run()
	sessions := make([]*Session, 0)
	for _, line := range lines {
		sessions = append(sessions, &Session{Id: line, server: srv})
	}
	return sessions, err
}

func (s *Session) Windows() ([]*Window, error) {
	lines, err := s.server.command("list-windows").withTargetSession(s).withFormat("#{window_id}").run()
	window := make([]*Window, 0)
	for _, line := range lines {
		window = append(window, &Window{Id: line})
	}
	return window, err

}

func (s *Session) Kill() error {
	_, err := s.server.command("kill-session").withTargetSession(s).run()
	return err
}

func (srv *Server) KillSessionWithName(s string) error {
	_, err := srv.command("kill-session").withTargetSessionName(s).run()
	return err
}
