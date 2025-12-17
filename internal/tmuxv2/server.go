package tmuxv2

type Server struct {
	socketPath        string
	controlModeClient *controlModeClient
}

func Tmux(socketPaths ...string) *Server {
	if len(socketPaths) > 0 {
		return &Server{socketPath: socketPaths[0]}
	} else {
		return &Server{}
	}
}

func (srv *Server) Kill() error {
	_, err := srv.command("kill-server").run()
	return err
}

func (srv *Server) HasSession(s *Session) bool {
	_, err := srv.command("has-session").withTargetSession(s).run()
	return err == nil
}

func (srv *Server) HasSessionWithName(s string) bool {
	_, err := srv.command("has-session").withTargetSessionName(s).run()
	return err == nil
}
