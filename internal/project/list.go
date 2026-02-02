package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/tmux"
)

var ErrAmbiguousName = errors.New("ambiguous project name detected")

type projectListEvent struct {
	Errors []string `json:"errors,omitempty"`
}
type projectListResult struct {
	Result []*Project `json:"result"`
}

// List all configured projects
//
// - Creates a project for each sub directory
//   - Giving them the name "[project_name]--[subdir_name]"
//
// - Creates projects for existing tmux sessions, if they do not match a config project
//
// Returns an ErrAmbiguousName if there are two configured projects with the same name,
// And other errors if anything else goes wrong with the queries.
//
// If the logging module is setup in ctx, it will log all errors at the Info level and
// Results at the debug level.
func List(ctx context.Context, tmux *tmux.Server, c *config.Config) ([]*Project, error) {
	event, resultEvent, finish := setupListLogEvents(ctx)
	defer finish()
	// The regular projects
	projects := listBareProjects(c)
	// Sub directory projects
	subDirProjects, err := listSubDirProjects(c)
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return nil, err
	}
	projects = append(projects, subDirProjects...)
	// Check for name ambiguity
	if l := getDuplicateNames(projects); len(l) > 0 {
		err = fmt.Errorf("%w: %v", ErrAmbiguousName, l)
		event.Errors = append(event.Errors, err.Error())
		return nil, err
	}
	// Existing tmux sessions
	if tmux != nil {
		sessionProjects, err := listSessionsWithoutProject(tmux, projects)
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			return nil, err
		}
		projects = append(projects, sessionProjects...)
	}
	// Add the internal fileds to all projects
	for _, p := range projects {
		p.server = tmux
		p.fullConfig = c
	}
	// Load project-specific configs
	loadProjectConfigs(ctx, projects)
	resultEvent.Result = projects
	return projects, nil
}

func setupListLogEvents(ctx context.Context) (*projectListEvent, *projectListResult, func()) {
	event := &projectListEvent{}
	finish := log.Track(ctx, "projectListEvent", event)
	resultEvent := &projectListResult{}
	resultFinish := log.TrackLevel(log.LEVEL_DEBUG, ctx, "projectListResult", resultEvent)

	return event, resultEvent, func() {
		finish()
		resultFinish()
	}
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
	for _, projectConfig := range config.Projects {
		if !projectConfig.SubDirectories {
			project := &Project{Name: projectConfig.Name, Config: projectConfig}
			projects = append(projects, project)
		}
	}
	return projects
}

func listSubDirProjects(c *config.Config) ([]*Project, error) {
	projects := make([]*Project, 0)
	for _, projectConfig := range c.Projects {
		if projectConfig.SubDirectories {
			dirEntries, err := os.ReadDir(projectConfig.Directory)
			if err != nil {
				return nil, fmt.Errorf("when checking for sub dirs for project list: %w", err)
			}
			for _, f := range dirEntries {
				if f.IsDir() {
					subName := fmt.Sprintf("%s--%s", projectConfig.Name, f.Name())
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

// loadProjectConfigs loads project-specific .tmixer.yml files from project directories
// and replaces the inherited config with project-specific overrides
func loadProjectConfigs(ctx context.Context, projects []*Project) {
	type loadProjectConfigsEvent struct {
		LoadedProjects []string `json:"loadedProjects,omitempty"`
		Errors         []string `json:"errors,omitempty"`
	}
	event := &loadProjectConfigsEvent{}
	finish := log.Track(ctx, "loadProjectConfigsEvent", event)
	defer finish()

	for _, project := range projects {
		// Skip orphaned sessions that don't have a config
		if project.Config == nil {
			continue
		}

		configPath := filepath.Join(project.Config.Directory, ".tmixer.yml")
		if _, err := os.Stat(configPath); err != nil {
			// No project-specific config, keep inherited config
			continue
		}

		bytes, err := os.ReadFile(configPath)
		if err != nil {
			event.Errors = append(event.Errors, fmt.Sprintf("failed to read %s: %v", configPath, err))
			continue
		}

		var projectConfig config.ProjectConfig
		err = yaml.Unmarshal(bytes, &projectConfig)
		if err != nil {
			event.Errors = append(event.Errors, fmt.Sprintf("failed to parse %s: %v", configPath, err))
			continue
		}

		// Replace the inherited config with project-specific config
		// But keep some values as is:
		projectConfig.Directory = project.Config.Directory
		projectConfig.SubDirectories = project.Config.SubDirectories
		project.Config = &projectConfig
		event.LoadedProjects = append(event.LoadedProjects, project.Name)
	}
}
