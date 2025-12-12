package tmux

import (
	"fmt"
	"log/slog"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"samuellando.com/tmixer/internal/project"
)

type Session struct {
	Name    string
	Project *project.Project
}

type sessionInfo struct {
	active       bool
	lastActivity *time.Time
	attached     bool
}

func (s *Session) Active() bool {
	return s.getInfo().active
}

func (s *Session) Attached() bool {
	return s.getInfo().attached
}

func (s *Session) LastActivity() *time.Time {
	return s.getInfo().lastActivity
}

func (s *Session) String() string {
	icon := "."
	if s.Active() {
		icon = "*"
	}
	if s.Attached() {
		icon = "+"
	}
	return fmt.Sprintf("%s %s", icon, s.Name)
}

func (s *Session) getInfo() sessionInfo {
	out, err := tmux("display", "-p", "-t", "="+s.Name+":", "#{?session_name,true,false}|#{session_attached_list}|#{session_activity}")
	if err != nil {
		slog.Debug("No TMUX server")
		return sessionInfo{attached: false, active: false}
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "|")
	active, err := strconv.ParseBool(parts[0])
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to parse active in %s", string(out)))
		panic(err)
	}
	attached := getClient() != nil && strings.Contains(parts[1], *getClient())
	var lastActivity *time.Time
	if len(parts[2]) > 0 {
		secs, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to parse active in %s", string(out)))
			panic(err)
		}
		t := time.Unix(secs, 0)
		lastActivity = &t
	}
	return sessionInfo{attached: attached, active: active, lastActivity: lastActivity}
}

func (s *Session) Swap() {
	client := getClient()
	if client == nil {
		slog.Error("No client found")
	}
	if s.Attached() {
		slog.Debug(fmt.Sprintf("Allready attached to session %s", s.Name))
		return
	}
	if !s.Active() {
		slog.Debug(fmt.Sprintf("Session %s is not active, starting..", s.Name))
		s.Reset()
	}
	tmux("switchc", "-c", *client, "-t", s.Name)
	slog.Debug(fmt.Sprintf("Switched to session %s", s.Name))
}

func (s *Session) Start() {
	_, err := tmux("new", "-s", s.Name, "-d")
	if err != nil {
		panic(err)
	}
}

func (s *Session) Stop() {
	_, err := tmux("kill-session", "-t", s.Name, "-d")
	if err != nil {
		panic(err)
	}
}

func (s *Session) Reset() {
	var temp *Session
	if s.Attached() {
		temp = switchToTempSession()
	}
	if s.Active() {
		s.Stop()
	}
	s.Start()
	if temp != nil {
		s.Swap()
		temp.Stop()
	}
}

func switchToTempSession() *Session {
	client := getClient()
	if client == nil {
		slog.Error("No client found")
	}
	uuid := uuid.NewString()
	// Crate a temp session to switch to temporarly
	_, err := tmux("new", "-s", uuid, "-d")
	if err != nil {
		panic(err)
	}
	_, err = tmux("switchc", "-c", *client, "-t", uuid)
	if err != nil {
		panic(err)
	}
	return &Session{
		Name: uuid,
	}
}

func RemoveIcon(name string) string {
	return strings.Split(strings.TrimSpace(name), " ")[1]
}

func CleanName(name string) string {
	return strings.ReplaceAll(name, ".", "_")
}

func StartServer() error {
	_, err := tmux("start-server")
	return err
}

func Sessions() map[string]*Session {
	sessions := make(map[string]*Session)
	out, err := tmux("list-sessions", "-F", "#{session_name}")
	if err != nil {
		slog.Debug("TMUX server is not started")
		return sessions
	}
	for name := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		session := Session{
			Name: CleanName(name),
		}
		slog.Debug(fmt.Sprintf("Detected tmux session %+v", session))
		sessions[CleanName(name)] = &session
	}
	return sessions
}

func LinkProjectsToSessions(sessions map[string]*Session, projects map[string]*project.Project) {
	for name, proj := range projects {
		if session, ok := sessions[CleanName(name)]; ok {
			slog.Debug(fmt.Sprintf("Found existing session for project %s", name))
			session.Project = proj
		} else {
			slog.Debug(fmt.Sprintf("Createing inactive session for project %s", name))
			session := Session{
				Name:    CleanName(name),
				Project: proj,
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

var cachedClient *string

func getClient() *string {
	if cachedClient != nil {
		return cachedClient
	}
	out, err := tmux("lsc", "-F", "#{client_tty} #{client_activity}")
	if err != nil {
		panic(err)
	}
	clients := make(map[int]*string)
	times := make([]int, 0)
	for info := range strings.SplitSeq(string(out), "\n") {
		if len(info) > 0 {
			data := strings.Split(info, " ")
			name := data[0]
			active, err := strconv.Atoi(data[1])
			if err != nil {
				panic(err)
			}
			clients[active] = &name
			times = append(times, active)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(times)))
	if len(clients) > 0 {
		cachedClient = clients[times[0]]
		slog.Debug(fmt.Sprintf("Caching client %s for the rest of the execution", cachedClient))
		return cachedClient
	} else {
		return nil
	}
}
