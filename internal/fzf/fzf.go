package fzf

import (
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"

	"samuellando.com/tmixer/internal/projectv2"
)

func PickProject(projects []*projectv2.Project) (*projectv2.Project, error) {
	input := projects
	projects = make([]*projectv2.Project, len(input))
	copy(projects, input)
	cmd := exec.Command("fzf", "--ansi")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("while opening stdin pipe to fzf: %w", err)
	}
	result := make(chan error)
	go func() {
		err := displayProjects(projects, stdin)
		result <- err
	}()
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	if err = <-result; err != nil {
		return nil, err
	}
	selected, err := parseOutput(string(out))
	if err != nil {
		return nil, err
	}
	for _, project := range projects {
		if project.Name == selected {
			return project, nil
		}
	}
	return nil, fmt.Errorf("No project selected")
}

func displayProjects(projects []*projectv2.Project, stdin io.WriteCloser) error {
	defer stdin.Close()
	var sortError error
	sort.Slice(projects, func(i, j int) bool {
		if sortError == nil {
			res, err := compare(projects, i, j)
			if err == nil {
				return res
			} else {
				sortError = err
			}
		}
		return projects[i].Name < projects[j].Name
	})
	if sortError != nil {
		return fmt.Errorf("while sorting projects: %w", sortError)
	}
	for _, project := range projects {
		info, err := display(project)
		if err != nil {
			return fmt.Errorf("while displaying project: %w", err)
		}
		io.WriteString(stdin, info+"\n")
	}
	return nil
}

func display(project *projectv2.Project) (string, error) {
	icon := "\uf114"
	status, err := project.Status()
	if err != nil {
		return project.Name, err
	}
	if status >= projectv2.PROJECT_STATUS_ACTIVE {
		icon = "\uf07b"
	}
	if status == projectv2.PROJECT_STATUS_ATTACHED {
		icon = "\033[31m" + icon + "\033[0m"
	}
	return icon + " " + project.Name, nil
}

func parseOutput(out string) (string, error) {
	parts := strings.Split(out, " ")
	if len(parts) != 2 {
		return "", fmt.Errorf("output should have 2 parts")
	}
	return strings.TrimSpace(parts[1]), nil
}

func compare(projects []*projectv2.Project, i, j int) (bool, error) {
	istatus, err := projects[i].Status()
	if err != nil {
		return false, err
	}
	jstatus, err := projects[j].Status()
	if err != nil {
		return false, err
	}
	if istatus == projectv2.PROJECT_STATUS_ATTACHED {
		return true, nil
	}
	if istatus == jstatus {
		switch istatus {
		case projectv2.PROJECT_STATUS_INACTIVE:
			return projects[i].Name < projects[j].Name, nil
		case projectv2.PROJECT_STATUS_ACTIVE:
			ila, err := projects[i].LastActivity()
			if err != nil {
				return false, err
			}
			jla, err := projects[j].LastActivity()
			if err != nil {
				return false, err
			}
			return ila.After(*jla), nil
		}
	}
	return istatus > jstatus, nil
}
