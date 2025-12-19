package tmuxv2

import (
	"errors"
	"fmt"
)

func (srv *Server) Switch(s *Session) error {
	_, err := srv.command("switch-client").withTargetSession(s).run()
	return err
}

func (srv *Server) ClientSession() (*Session, error) {
	lines, err := srv.command("display-message").withFlag("-p").withFormat("#{client_session}").run()
	if len(lines) == 0 {
		return nil, errors.Join(fmt.Errorf("Got no client session response"), err)
	}
	sessions, err := srv.ListSessions()
	if err != nil {
		return nil, fmt.Errorf("when listing sessions for client session: %w", err)
	}
	for _, session := range sessions {
		name, err := session.Name()
		if err != nil {
			return nil, fmt.Errorf("when getting session name for client session: %w", err)
		}
		if name == lines[0] {
			return &Session{Id: session.Id, server: srv}, nil
		}
	}
	return nil, fmt.Errorf("No session found")
}
