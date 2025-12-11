package tmux

import (
	"fmt"
	"log/slog"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"samuellando.com/tmixer/internal/project"
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

func (s *Session) Swap() {
	if s.Attached {
		slog.Debug(fmt.Sprintf("Allready attached to session %s", s.Name))
		return
	}
	if !s.Active {
		slog.Debug(fmt.Sprintf("Session %s is not active, starting..", s.Name))
		s.Reset()
	}
	tmux("switchc", "-c", getCleint(), "-t", s.Name)
	slog.Debug(fmt.Sprintf("Switched to session %s", s.Name))
}

func (s *Session) Reset() {
	uuid := uuid.NewString()
	if s.Attached {
		// Crate a temp session to switch to temporarly
		_, err := tmux("new", "-s", uuid, "-d")
		if err != nil {
			panic(err)
		}
		_, err = tmux("switchc", "-c", getCleint(), "-t", uuid)
		if err != nil {
			panic(err)
		}
	}
	if s.Active {
		_, err := tmux("kill-session", "-t", s.Name, "-d")
		if err != nil {
			panic(err)
		}
	}
	_, err := tmux("new", "-s", s.Name, "-d")
	if err != nil {
		panic(err)
	}
	if s.Attached {
		// Crat a temp session to switch to temporarly
		_, err := tmux("kill-session", "-t", uuid)
		if err != nil {
			panic(err)
		}
		_, err = tmux("switchc", "-c", getCleint(), "-t", s.Name)
		if err != nil {
			panic(err)
		}
	}
}

func RemoveIcon(name string) string {
	return strings.Split(strings.TrimSpace(name), " ")[1]
}

func cleanName(name string) string {
	return strings.ReplaceAll(name, ".", "_")
}

func StartServer() error {
	_, err := tmux("start-server")
	return err
}

func Sessions() map[string]*Session {
	out, err := tmux("list-sessions", "-F", "#{session_name} #{session_attached} #{session_attached_list}")
	if err != nil {
		panic(err)
	}
	sessions := make(map[string]*Session, 0)
	for info := range strings.SplitSeq(string(out), "\n") {
		if len(info) > 0 {
			name := strings.Split(info, " ")[0]
			attached := strings.Split(info, " ")[1]
			if attached == "1" {
				client := strings.Split(info, " ")[2]
				if client != getCleint() {
					attached = "0"
				}
			}
			session := Session{
				Name:     cleanName(name),
				Attached: attached == "1",
				Active:   true,
			}
			slog.Debug(fmt.Sprintf("Detected tmux session %+v", session))
			sessions[name] = &session
		}
	}
	return sessions
}

func LinkProjectsToSessions(sessions map[string]*Session, projects map[string]*project.Project) {
	for name, proj := range projects {
		if session, ok := sessions[cleanName(name)]; ok {
			slog.Debug(fmt.Sprintf("Found active session for project %s", name))
			session.Project = proj
		} else {
			slog.Debug(fmt.Sprintf("Createing inactive session for project %s", name))
			session := Session{
				Name:     cleanName(name),
				Active:   false,
				Attached: false,
				Project:  proj,
			}
			sessions[name] = &session
		}
	}
}

func tmux(args ...string) ([]byte, error) {
	slog.Debug(fmt.Sprintf("Calling tmux command with args: %s", args))
	cmd := exec.Command("tmux", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("Command returned an error")
		slog.Error(string(out))
	}
	return out, err
}

var cachedClient string

func getCleint() string {
	if cachedClient != "" {
		return cachedClient
	}
	out, err := tmux("lsc", "-F", "#{client_tty} #{client_activity}")
	if err != nil {
		panic(err)
	}
	clients := make(map[int]string)
	times := make([]int, 0)
	for info := range strings.SplitSeq(string(out), "\n") {
		if len(info) > 0 {
			data := strings.Split(info, " ")
			name := data[0]
			active, err := strconv.Atoi(data[1])
			if err != nil {
				panic(err)
			}
			clients[active] = name
			times = append(times, active)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(times)))
	cachedClient = clients[times[0]]
	slog.Debug(fmt.Sprintf("Caching client %s for the rest of the execution", cachedClient))
	return cachedClient
}
