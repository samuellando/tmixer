package projectv2

import (
	"testing"

	"samuellando.com/tmixer/internal/tmuxv2"
	"samuellando.com/tmixer/internal/config"
)

func TestListIncludesAllProjects(t *testing.T) {
	config := config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": &config.ProjectConfig{
				Directory: "/home/test/bin",
			},
			"projects": &config.ProjectConfig{
				Directory: "/home/test/Projects",
			},
		},
	}
}

func TestListIncludesAllSessions(t *testing.T) {

}

func TestListMatchesExistingSessions(t *testing.T) {

}
