package tmuxv2

import (
	"samuellando.com/tmixer/internal/project"
)

type Session struct {
	Id string
	Project *project.Project
}

// func (s *Session) New() *Session {
//
// }
//
// func (s *Session) ListSessions() []*Session {
// }
//
// func (s *Session) HasSession() bool {
//
// }
//
// func (s *Session) Rename() *Session {
//
// }
//
// func (s *Session) Windows() {
//
// }
//
// func (s *Session) Kill() {
//
// }
//
// func (s *Session) Lock() {
//
// }
