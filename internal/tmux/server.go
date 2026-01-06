package tmux

import (
	"context"
	"errors"
	"fmt"
)

var ErrSessionNotFound = errors.New("session not found")

type Server struct {
	SocketPath        string
	controlModeClient *controlModeClient
	ctx               context.Context
}

func Tmux(ctx context.Context, socketPaths ...string) *Server {
	if len(socketPaths) > 0 {
		return &Server{ctx: ctx, SocketPath: socketPaths[0]}
	} else {
		return &Server{ctx: ctx}
	}
}

func (srv *Server) SetHook(name, cmd string) error {
	_, err := srv.command("set-hook").withFlag("-g").withArgument(name).withArgument(cmd).run()
	return err
}

func (srv *Server) Kill() error {
	_, err := srv.command("kill-server").run()
	return err
}

func (srv *Server) HasSession(s *Session) bool {
	// Using tmux has-session causes problems for control mode, so it's better to list sessions
	sessions, err := srv.ListSessions()
	if err != nil {
		return false
	}
	for _, session := range sessions {
		if s.Id == session.Id {
			return true
		}
	}
	return false
}

func (srv *Server) GetSessionWithName(name string) (*Session, error) {
	lines, err := srv.command("list-sessions").withFilter(fmt.Sprintf("#{==:#{session_name},%s}", name)).withFormat("#{session_id}").run()
	if len(lines) == 0 {
		return nil, ErrSessionNotFound
	}
	if len(lines) > 1 {
		return nil, fmt.Errorf("Multiple matches")
	}
	id, err := parseSessionId(lines[0])
	if err != nil {
		return nil, err
	}
	return &Session{Id: id, server: srv}, nil
}

func (srv *Server) HasSessionWithName(name string) bool {
	s, err := srv.GetSessionWithName(name)
	return s != nil && err == nil
}
