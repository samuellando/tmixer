package fzf

import (
	"context"
	"fmt"
	"io"
	"sort"

	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
)

func DisplayProjects(ctx context.Context, projects []*project.Project, w io.Writer) error {
	type displayProjectsEvent struct {
		Errors []string `json:"errors,omitempty"`
	}
	event := &displayProjectsEvent{}
	finish := log.Track(ctx, "displayProjectsEvent", event)
	defer finish()
	var sortError error
	sort.Slice(projects, func(i, j int) bool {
		if sortError == nil {
			res, err := compare(projects, i, j)
			if err == nil {
				return res
			} else {
				event.Errors = append(event.Errors, err.Error())
				sortError = err
			}
		}
		return projects[i].Name < projects[j].Name
	})
	if sortError != nil {
		err := fmt.Errorf("while sorting projects: %w", sortError)
		event.Errors = append(event.Errors, err.Error())
		return err
	}
	for _, project := range projects {
		info, err := display(project)
		if err != nil {
			err := fmt.Errorf("while displaying project: %w", err)
			event.Errors = append(event.Errors, err.Error())
			return err
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
