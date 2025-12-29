package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/tmux"
)

func List(tmux *tmux.Server, config *config.Config) ([]*Project, error) {
	projects := listBareProjects(config)
	subDirProjects, err := listSubDirProjects(config)
	if err != nil {
		return nil, err
	}
	projects = append(projects, subDirProjects...)
	if tmux != nil {
		sessionProjects, err := listSessionsWithoutProject(tmux, projects)
		if err != nil {
			return nil, err
		}
		projects = append(projects, sessionProjects...)
	}
	// Add the server to all projects
	for _, p := range projects {
		p.server = tmux
		p.fullConfig = config
	}
	return projects, nil
}

func listBareProjects(config *config.Config) []*Project {
	projects := make([]*Project, 0)
	for name, projectConfig := range config.Projects {
		if !projectConfig.SubDirectories {
			project := &Project{Name: name, Config: projectConfig}
			projects = append(projects, project)
		}
	}
	return projects
}

func listSubDirProjects(c *config.Config) ([]*Project, error) {
	projects := make([]*Project, 0)
	for name, projectConfig := range c.Projects {
		if projectConfig.SubDirectories {
			dirEntries, err := os.ReadDir(projectConfig.Directory)
			if err != nil {
				return nil, fmt.Errorf("when checking for sub dirs for project list: %w", err)
			}
			for _, f := range dirEntries {
				if f.IsDir() {
					subName := fmt.Sprintf("%s--%s", name, f.Name())
					project := &Project{Name: subName, Config: &config.ProjectConfig{
						Directory:      filepath.Join(projectConfig.Directory, f.Name()),
						SubDirectories: false,
						Windows:        projectConfig.Windows,
						SwitchCommands: projectConfig.SwitchCommands,
					}}
					projects = append(projects, project)
				}
			}
		}
	}
	return projects, nil
}

func listSessionsWithoutProject(s *tmux.Server, configProjects []*Project) ([]*Project, error) {
	projects := make([]*Project, 0)
	sessionsMatched := make(map[string]bool)
	for _, project := range configProjects {
		sessionsMatched[project.TmuxSessionName()] = true
	}
	sessions, err := s.ListSessions()
	if err != nil {
		return configProjects, fmt.Errorf("when listing sessions for project list: %w", err)
	}
	for _, session := range sessions {
		name, err := session.Name()
		if err != nil {
			return configProjects, fmt.Errorf("when getting session name for project list: %w", err)
		}
		if name == tmux.CONTROL_SESSION_NAME {
			continue
		}
		if _, ok := sessionsMatched[name]; !ok {
			project := &Project{Name: name, server: s}
			projects = append(projects, project)
		}
	}
	return projects, nil
}

// Sort the projects by the last activity time. If inactive, make sure the Default
// Project is the next one.
func sortProjects(config *config.Config, projects []*Project) error {
	var sortErr error
	sort.Slice(projects, func(i, j int) bool {
		_, err1 := projects[i].Session()
		_, err2 := projects[j].Session()
		if err1 != nil && err2 != nil {
			if config.DefaultProject != nil {
				if *config.DefaultProject == projects[i].Name {
					return true
				}
				if *config.DefaultProject == projects[j].Name {
					return false
				}
			}
			return false
		}
		if err1 == ErrSessionNotFound {
			return false
		}
		if err1 != nil {
			sortErr = err1
			return false
		}
		if err2 == ErrSessionNotFound {
			return true
		}
		if err2 != nil {
			sortErr = err2
			return false
		}
		t1, err := projects[i].LastActivity()
		if err != nil {
			sortErr = err
			return false
		}
		t2, err := projects[j].LastActivity()
		if err != nil {
			sortErr = err
			return false
		}
		return t1.After(*t2)
	})
	return sortErr
}
