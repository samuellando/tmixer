package tmuxv2

type Server struct {
	SocketPath        string
	controlModeClient *controlModeClient
}

func Tmux(socketPaths ...string) *Server {
	if len(socketPaths) > 0 {
		return &Server{SocketPath: socketPaths[0]}
	} else {
		return &Server{}
	}
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

func (srv *Server) HasSessionWithName(s string) bool {
	// Using tmux has-session causes problems for control mode, so it's better to list sessions
	sessions, err := srv.ListSessions()
	if err != nil {
		return false
	}
	for _, session := range sessions {
		name, err := session.Name()
		if err != nil {
			return false
		}
		if name == s {
			return true
		}
	}
	return false
}
