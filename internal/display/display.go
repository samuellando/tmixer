package display

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
)

func Projects(ctx context.Context, projects []*project.Project) ([]string, error) {
	logEvent := log.Track(ctx, "displayProjects")
	defer logEvent.Done()

	projects, err := sortProjects(projects)
	if err != nil {
		logEvent.Error(err)
		return nil, err
	}

	result := make([]string, 0, len(projects))
	for _, project := range projects {
		info, err := display(project)
		if err != nil {
			err := fmt.Errorf("while generating display string for project: %w", err)
			logEvent.Error(err)
			return nil, err
		}
		result = append(result, info)
	}
	logEvent.Log("result", result)
	return result, nil
}

func GetProjectNameFromOutput(info string) (string, error) {
	return parseOutput(info)
}

func sortProjects(projects []*project.Project) ([]*project.Project, error) {
	// Clone before sorting
	projects = slices.Clone(projects)
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
		err := fmt.Errorf("while sorting projects: %w", sortError)
		return nil, err
	}
	return projects, nil
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
	iStatus, err := projects[i].Status()
	if err != nil {
		return false, err
	}
	jStatus, err := projects[j].Status()
	if err != nil {
		return false, err
	}
	// The attached project is always first
	if iStatus == project.PROJECT_STATUS_ATTACHED {
		return true, nil
	}
	// Otherwise we compare based on the status.
	// if the statuses are the same, we should compare by name for inactive projects
	// and by the last activity time for active projects.
	if iStatus == jStatus {
		switch iStatus {
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
	return iStatus > jStatus, nil
}
