package projectv2

import (
	"fmt"
	"os"

	"samuellando.com/tmixer/internal/configv2"
	"samuellando.com/tmixer/internal/tmuxv2"
)


func List(tmux *tmuxv2.Server, config *config.Config) ([]*Project, error) {
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

func listSubDirProjects(config *config.Config) ([]*Project, error) {
	projects := make([]*Project, 0)
	for name, projectConfig := range config.Projects {
		if projectConfig.SubDirectories {
			dirEntries, err := os.ReadDir(projectConfig.Directory)
			if err != nil {
				return nil, fmt.Errorf("when checking for sub dirs for project list: %w", err)
			}
			for _, f := range dirEntries {
				if f.IsDir() {
					subName := fmt.Sprintf("%s--%s", name, f.Name())
					project := &Project{Name: subName, Config: projectConfig}
					projects = append(projects, project)
				}
			}
		}
	}
	return projects, nil
}

func listSessionsWithoutProject(tmux *tmuxv2.Server, configProjects []*Project) ([]*Project, error) {
	projects := make([]*Project, 0)
	sessionsMatched := make(map[string]bool)
	for _, project := range configProjects {
		sessionsMatched[project.TmuxSessionName()] = true
	}
	sessions, err := tmux.ListSessions()
	if err != nil {
		return configProjects, fmt.Errorf("when listing sessions for project list: %w", err)
	}
	for _, session := range sessions {
		name, err := session.Name()
		if err != nil {
			return configProjects, fmt.Errorf("when getting session name for project list: %w", err)
		}
		if name == tmuxv2.CONTROL_SESSION_NAME {
			continue
		}
		if _, ok := sessionsMatched[name]; !ok {
			project := &Project{Name: name, server: tmux}
			projects = append(projects, project)
		}
	}
	return projects, nil
}
