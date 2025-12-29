package fzf

import (
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/project"
)

func PickProject(config *config.Config, projects []*project.Project) (*project.Project, error) {
	input := projects
	projects = make([]*project.Project, len(input))
	copy(projects, input)
	cmd := exec.Command("fzf", config.FzfFlags...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("while opening stdin pipe to fzf: %w", err)
	}
	result := make(chan error)
	go func() {
		err := DisplayProjects(projects, stdin)
		stdin.Close()
		result <- err
	}()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("fzf command error: %w %s", err, string(out))
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

func DisplayProjects(projects []*project.Project, w io.Writer) error {
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
		io.WriteString(w, info+"\n")
	}
	return nil
}

func display(p *project.Project) (string, error) {
	icon := "\uf114"
	status, err := p.Status()
	if err != nil {
		return p.Name, err
	}
	if status >= project.PROJECT_STATUS_ACTIVE {
		icon = "\uf07b"
	}
	if status == project.PROJECT_STATUS_ATTACHED {
		icon = "\033[31m" + icon + "\033[0m"
	}
	return icon + " " + p.Name, nil
}

func parseOutput(out string) (string, error) {
	parts := strings.Split(out, " ")
	if len(parts) != 2 {
		return "", fmt.Errorf("output should have 2 parts")
	}
	return strings.TrimSpace(parts[1]), nil
}

func compare(projects []*project.Project, i, j int) (bool, error) {
	istatus, err := projects[i].Status()
	if err != nil {
		return false, err
	}
	jstatus, err := projects[j].Status()
	if err != nil {
		return false, err
	}
	if istatus == project.PROJECT_STATUS_ATTACHED {
		return true, nil
	}
	if istatus == jstatus {
		switch istatus {
		case project.PROJECT_STATUS_INACTIVE:
			return projects[i].Name < projects[j].Name, nil
		case project.PROJECT_STATUS_ACTIVE:
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
