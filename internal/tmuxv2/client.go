package tmuxv2

func (srv *Server) Switch(s *Session) error {
	_, err := srv.command("switch-client").withTargetSession(s).run()
	return err
}

func (srv *Server) ClientSession() (*Session, error) {
}

func (srv *Server) DisplayMessage(m string) error {
	_, err := srv.command("suspend-client").withArgument(m).run()
	return err
}
