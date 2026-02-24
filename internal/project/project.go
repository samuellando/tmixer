package project

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/tmux"
)

type ProjectStatus int

const (
	PROJECT_STATUS_INACTIVE ProjectStatus = iota
	PROJECT_STATUS_ACTIVE
	PROJECT_STATUS_ATTACHED
)

var ErrSessionNotFound = errors.New("session not found")

type Project struct {
	Name       string
	Config     *config.ProjectConfig
	server     *tmux.Server
	fullConfig *config.Config
}

// Returns one of PROJECT_STATUS_INACTIVE, PROJECT_STATUS_ACTIVE, and PROJECT_STATUS_ATTACHED
// A project is inactive if it has no running session, active if it does and attached
// if the currently running client is attached to it's session.
func (p *Project) Status() (ProjectStatus, error) {
	var activeSessionName string
	status := PROJECT_STATUS_INACTIVE
	client, err := p.server.ActiveClient()
	if err != nil && err != tmux.ErrNoActiveClient {
		return status, fmt.Errorf("when getting active client for status: %w", err)
	} else if err == nil {
		session, err := client.Session()
		if err != nil {
			return status, fmt.Errorf("when getting active session for status: %w", err)
		}
		activeSessionName, err = session.Name()
		if err != nil {
			return status, fmt.Errorf("when getting active session name status: %w", err)
		}
	}
	if activeSessionName == p.TmuxSessionName() {
		status = PROJECT_STATUS_ATTACHED
	} else if p.server.HasSessionWithName(p.TmuxSessionName()) {
		status = PROJECT_STATUS_ACTIVE
	}
	return status, nil
}

// Return the project's tmux session.
// If there is no session returns ErrSessionNotFound
func (p *Project) Session() (*tmux.Session, error) {
	s, err := p.server.GetSessionWithName(p.TmuxSessionName())
	if err == tmux.ErrSessionNotFound {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("when getting project session: %w", err)
	}
	return s, nil
}

// Return the tmux session Name for the project. Since not all project names are
// compatible with tmux, this will return a compatible name.
func (p *Project) TmuxSessionName() string {
	s := p.Name
	b := strings.Builder{}
	for i := 0; i < len(s); i++ {
		if s[i] == '#' {
			if i < len(s)-1 {
				if s[i+1] == '#' {
					// ##
					b.WriteByte('#')
					i++
					continue
				}
				if s[i+1] == '}' {
					// #}
					b.WriteByte('}')
					i++
					continue
				}
				if s[i+1] == '}' {
					// #,
					b.WriteByte(',')
					i++
					continue
				}
			}
		}
		b.WriteByte(s[i])
	}
	s = b.String()
	out := ""
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch r {
		case '.':
			out += `_`
		case ':':
			out += `_`
		case '\a':
			out += `\a`
		case '\b':
			out += `\b`
		case '\f':
			out += `\f`
		case '\n':
			out += `\n`
		case '\r':
			out += `\r`
		case '\t':
			out += `\t`
		case '\v':
			out += `\v`
		case '\\':
			out += `\\`
		case '"':
			out += `"`
		default:
			if r >= 0x20 && r <= 0x7e { // printable ASCII
				out += string(r)
			} else if (r >= 0x1 && r <= 0x7f) || r == utf8.RuneError {
				// Control characters
				out += fmt.Sprintf(`\%03o`, s[i])
			} else {
				out += string(r)
			}
		}
		i += size
	}
	return out
}

// Return the last activity time of the projects session.
// If no session exists will return ErrSessionNotFound
func (p *Project) LastActivity() (*time.Time, error) {
	session, err := p.Session()
	if err != nil {
		return nil, err
	}
	return session.LastActivity()
}

// Return whether or not the project's configure time to live has passed
// Will always return false if an error occurs internally, as well as returning the error
func (p *Project) TtlPassed() (bool, error) {
	if p.fullConfig.Ttl == nil {
		return false, nil
	}
	ttl, err := time.ParseDuration(*p.fullConfig.Ttl)
	if err != nil {
		return false, fmt.Errorf("while checking if ttl has passed: %w", err)
	}
	la, err := p.LastActivity()
	if err != nil {
		return false, fmt.Errorf("while checking if ttl has passed: %w", err)
	}
	return time.Since(*la) >= ttl, nil
}
