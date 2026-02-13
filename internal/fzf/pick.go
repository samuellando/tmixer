package fzf

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
)

func PickProject(ctx context.Context, config *config.Config, projects []*project.Project) (*project.Project, error) {
	type pickProjectEvent struct {
		Args         []string `json:"args"`
		Output       string   `json:"output"`
		ParsedOutput string   `json:"parsedOutput"`
		Errors       []string `json:"errors,omitempty"`
	}
	event := &pickProjectEvent{}
	finish := log.Track(ctx, "pickProjectEvent", event)
	defer finish()

	// Make a copy since we do some sorting
	input := projects
	projects = make([]*project.Project, len(input))
	copy(projects, input)

	// We need to use the raw stdin for the fzf command
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			err := fmt.Errorf("while getting raw stdin: %w", err)
			event.Errors = append(event.Errors, err.Error())
			return nil, err
		}
		defer term.Restore(fd, oldState)
	}

	// Run the command in a ptx for consistency accross envs
	cmd := exec.Command("fzf", config.FzfFlags...)
	event.Args = cmd.Args
	ptx, err := startPty(cmd, os.Stdin, os.Stdout)
	if err != nil {
		err := fmt.Errorf("while opening pty for fzf: %w", err)
		event.Errors = append(event.Errors, err.Error())
		return nil, err
	}
	defer ptx.Close()

	// Now pipe in the projects to fzf
	displayErrs := make(chan error)
	go func() {
		err := DisplayProjects(ctx, projects, ptx)
		ptx.CloseInPipe()
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
		}
		displayErrs <- err
	}()

	// Wait for the command to exit
	out, err := io.ReadAll(ptx)
	err = cmd.Wait()
	if err != nil {
		err := fmt.Errorf("fzf command error: %w", err)
		event.Errors = append(event.Errors, err.Error())
		return nil, err
	}
	if err = <-displayErrs; err != nil {
		event.Errors = append(event.Errors, err.Error())
		return nil, err
	}

	// Return the selected project
	event.Output = string(out)
	name, err := parseOutput(string(out))
	if err != nil {
	}
	event.ParsedOutput = name
	selected := getSelectedProject(name, projects)
	if selected == nil {
		return nil, fmt.Errorf("No project selected")
	} else {
		return selected, nil
	}
}

func getSelectedProject(name string, projects []*project.Project) *project.Project {
	for _, project := range projects {
		if project.Name == name {
			return project
		}
	}
	return nil
}

func parseOutput(out string) (string, error) {
	parts := strings.Split(out, " ")
	if len(parts) != 2 {
		return "", fmt.Errorf("output should have 2 parts")
	}
	return strings.TrimSpace(parts[1]), nil
}
