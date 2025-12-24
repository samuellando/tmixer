package tmuxv2

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

type sessionId string

func parseSessionId(s string) (sessionId, error) {
	if len(s) == 0 || s[0] != '$' {
		return "", fmt.Errorf("invalid session id: %q", s)
	}
	return sessionId(s), nil
}

type Session struct {
	Id        sessionId
	server    *Server
}

func (srv *Server) New(name string, dir ...string) (*Session, error) {
	c := srv.command("new-session").withSession(name).print().detached().withFormat("#{session_id}")
	if len(dir) > 0 {
		c = c.withWorkingDirectory(dir[0])
	}
	lines, err := c.run()
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
    s := &Session{Id: id, server: srv}
	if len(dir) > 0 {
		return s, s.SetOption("@working_dir", dir[0])
	}
	return s, nil
}

func (s *Session) SetOption(key, value string) error {
	_, err := s.server.command("set-option").withTargetSession(s).withArgument(key).withArgument(value).run()
	return err
}

func (s *Session) GetOption(key string) (string, error) {
	lines, err := s.server.command("show-options").withTargetSession(s).withArgument(key).withFlag("-v").withFlag("-q").run()
	if len(lines) == 0 {
		return "", fmt.Errorf("Got no output: %w", err)
	}
	return lines[0], err
}

func (s *Session) Name() (string, error) {
	lines, err := s.server.command("display").withFlag("-p").withTargetSession(s).withFormat("#{session_name}").run()
	if len(lines) == 0 {
		return "", errors.Join(fmt.Errorf("Got no name"), err)
	}
	return lines[0], err
}

func (s *Session) LastActivity() (*time.Time, error) {
	lines, err := s.server.command("display").withFlag("-p").withTargetSession(s).withFormat("#{session_activity}").run()
	if len(lines) == 0 {
		return nil, errors.Join(fmt.Errorf("Got no time"), err)
	}
	unix, err := strconv.ParseInt(lines[0], 10, 64)
	if err != nil {
		return nil, err
	}
	t := time.Unix(unix, 0)
	return &t, nil
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

func (s *Session) NewWindow(name string) (*Window, error) {
	c := s.server.command("new-window").withTargetSession(s).withFlag("-d").withName(name).print().withFormat("#{window_id}")
	if dir, err := s.GetOption("@working_dir"); err == nil {
		c = c.withWorkingDirectory(dir)
	}
	lines, err := c.run()
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
	w :=  &Window{Id: id, server: s.server}
	return w, nil
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
