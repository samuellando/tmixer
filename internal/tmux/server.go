package tmux

import (
	"errors"
	"fmt"
	"time"
)

var ErrSessionNotFound = errors.New("session not found")

// Tmux servers hold the connection information and possibly a active
// control mode session with the tmux server on a specified socket.
// This is the main entry point for all tmux operations
type Server struct {
	SocketPath        string
	controlModeClient *controlModeClient
	stats             ServerStats
}

// A server session serves the purpose of capturing session specific stats.
// It is derived from a server, and reports stats of all command runs that
// happened since the start of the session. This may be an issue if there are
// concurrent sessions, but for the purposes of logging it is okay.
type ServerSession struct {
	*Server
	beforeStats ServerStats
}

type ServerStats struct {
	CommandRuns int
	TotalTime   time.Duration
}

func Tmux(socketPaths ...string) *Server {
	if len(socketPaths) > 0 {
		return &Server{SocketPath: socketPaths[0]}
	} else {
		return &Server{}
	}
}

func (srv *Server) Session() *ServerSession {
	s := ServerSession{srv, srv.stats}
	return &s
}

func (srv *ServerSession) Stats() ServerStats {
	return ServerStats{
		CommandRuns: srv.stats.CommandRuns - srv.beforeStats.CommandRuns,
		TotalTime:   srv.stats.TotalTime - srv.beforeStats.TotalTime,
	}
}

func (srv *Server) Stats() ServerStats {
	return srv.stats
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
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, ErrSessionNotFound
	}
	if len(lines) > 1 {
		return nil, fmt.Errorf("MULTIPLE MATCHES")
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
