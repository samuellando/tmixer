package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/tmux"
)

var ErrAmbiguousName = errors.New("Ambiguous project name detected")

// List all configured projects
// - Creates a project for each sub directory
//   - Giving them the name "[project_name]--[subdir_name]"
//
// - Creates projects for existing tmux sessions, if they do not match a config project
//
// Returns an ErrAmbiguousName if there are two configured projects with the same name,
// And other errors if anything else goes wrong with the queries.
func List(tmux *tmux.Server, config *config.Config) ([]*Project, error) {
	projects := listBareProjects(config)
	subDirProjects, err := listSubDirProjects(config)
	if err != nil {
		return nil, err
	}
	projects = append(projects, subDirProjects...)
	if l := getDuplicateNames(projects); len(l) > 0 {
		return nil, fmt.Errorf("%w %v", ErrAmbiguousName, l)
	}
	if tmux != nil {
		sessionProjects, err := listSessionsWithoutProject(tmux, projects)
		if err != nil {
			return nil, err
		}
		projects = append(projects, sessionProjects...)
	}
	// Add the internal fileds to all projects
	for _, p := range projects {
		p.server = tmux
		p.fullConfig = config
	}
	return projects, nil
}

func getDuplicateNames(projects []*Project) []string {
	seen := make(map[string]bool)
	duplicates := make([]string, 0)
	for _, p := range projects {
		if seen[p.Name] {
			duplicates = append(duplicates, p.Name)
		}
		seen[p.Name] = true
	}
	return duplicates
}

// List all projects that are bare ie. don't have any sort of recusion like subdirs.
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
