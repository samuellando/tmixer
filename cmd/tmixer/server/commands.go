package server

import (
	"context"
	"fmt"
	"strings"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/display"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/protocol"
	"samuellando.com/tmixer/internal/tmux"
)

func (s *server) runCommand(ctx context.Context, config *config.Config, args ...string) (*protocol.Output, error) {
	logEvent := log.Track(ctx, "runEvent")
	defer logEvent.Done()
	srv, err := s.getTmuxServer(config)
	if err != nil {
		logEvent.Error(err)
		return nil, err
	}
	command := "switch"
	if len(args) >= 1 {
		command = args[0]
	}
	logEvent.Log("command", command)
	projects, err := project.List(ctx, srv, config)
	if err != nil {
		logEvent.Error(err)
		return nil, err
	}
	query := ""
	if len(args) >= 2 {
		query = args[1]
	}
	out, err := executeCommand(ctx, srv, command, query, config, projects)
	if err != nil {
		logEvent.Error(err)
		return out, err
	}
	return out, nil
}

func executeCommand(ctx context.Context, srv *tmux.Server, command, query string, config *config.Config, projects []*project.Project) (*protocol.Output, error) {
	switch command {
	// Internal (undocumented) commands
	case "list":
		projects, err := project.List(ctx, srv, config)
		if err != nil {
			return nil, err
		}
		disp, err := display.Projects(ctx, projects)
		if err != nil {
			return nil, err
		}
		return &protocol.Output{
			Output: ptr(strings.Join(disp, "\n")),
		}, nil
	case "start":
		return nil, start(ctx, srv, query, config, projects)
	case "switch":
		return nil, runSwitch(ctx, query, projects)
	case "kill":
		return nil, kill(ctx, query, projects)
	case "reset":
		return nil, reset(ctx, query, projects)
	default:
		return nil, runSwitch(ctx, command, projects)
	}
}

func start(ctx context.Context, srv *tmux.Server, query string, config *config.Config, projects []*project.Project) error {
	var selection *project.Project
	if query != "" {
		selection = getProject(query, projects)
	} else {
		if config.DefaultProject != nil {
			selection = getProject(*config.DefaultProject, projects)
		} else {
			return ErrNoSelection
		}
	}
	_, err := selection.Start(ctx)
	return err
}

func runSwitch(ctx context.Context, query string, projects []*project.Project) error {
	var selection *project.Project
	if query != "" {
		selection = getProject(query, projects)
	} else {
		return ErrNoSelection
	}
	_, err := selection.Switch(ctx)
	return err
}

func kill(ctx context.Context, query string, projects []*project.Project) error {
	var selection *project.Project
	if query != "" {
		selection = getProject(query, projects)
	} else {
		return ErrNoSelection
	}
	cleanup, err := selection.Kill(ctx)
	cleanup()
	return err
}

func reset(ctx context.Context, query string, projects []*project.Project) error {
	var selection *project.Project
	if query != "" {
		selection = getProject(query, projects)
	} else {
		for _, p := range projects {
			status, err := p.Status()
			if err != nil {
				return fmt.Errorf("while getting project status for reset: %w", err)
			}
			if status == project.PROJECT_STATUS_ATTACHED {
				selection = p
				break
			}
		}
	}
	if selection == nil {
		return ErrNoSelection
	}
	cleanup, err := selection.Reset(ctx)
	cleanup()
	return err
}

func getProject(query string, projects []*project.Project) *project.Project {
	for _, p := range projects {
		if p.Name == query {
			return p
		}
	}
	return nil
}
