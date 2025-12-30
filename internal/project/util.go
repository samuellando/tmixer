package project

import (
	"sort"

	"samuellando.com/tmixer/internal/config"
)

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
