package tmux

import (
	"fmt"
	"samuellando.com/tmixer/internal/project"
	"strings"
	"os/exec"
)

type Session struct {
	Name     string
	Active   bool
	Attached bool
	Project  *project.Project
}

func (s *Session) String() string {
	icon := "."
	if s.Active {
		icon = "*"
	}
	if s.Attached {
		icon = "+"
	}
	return fmt.Sprintf("%s %s", icon, s.Name)
}

func CleanName(name string) string {
	return strings.Split(strings.TrimSpace(name), " ")[1]
}

func (s *Session) Swap() {
}

func (s *Session) Reset() {
}

func tmux(args ...string) ([]byte, error) {
	cmd := exec.Command("tmux", args...)
	return cmd.Output()
}

func StartServer() error {
	_, err := tmux("start-server")
	return err
}

func Sessions() map[string]*Session {
	out, err := tmux("list-sessions", "-F", "#{session_name} #{session_attached}")
	if err != nil {
		panic(err)
	}
	sessions := make(map[string]*Session, 0)
	for info := range strings.SplitSeq(string(out), "\n") {
		if len(info) > 0 {
			name := strings.Split(info, " ")[0]
			attached := strings.Split(info, " ")[1]
			session := Session{
				Name:     name,
				Attached: attached == "1",
				Active:   true,
			}
			sessions[name] = &session
		}
	}
	return sessions
}

func LinkProjectsToSessions(sessions map[string]*Session, projects map[string]*project.Project) {
	for name, proj := range projects {
		if session, ok := sessions[name]; ok {
			session.Project = proj
		} else {
			session := Session{
				Name: name,
				Active: false,
				Attached: false,
				Project: proj,
			}
			sessions[name] = &session
		}
	}
}
