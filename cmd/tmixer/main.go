package main

import (
	"errors"
	"log"
	"os"

	"samuellando.com/tmixer/cmd/tmixer/client"
	"samuellando.com/tmixer/cmd/tmixer/server"
)

var ErrNoSelection = errors.New("NO SELECTION MADE")
var ErrProjectNotFound = errors.New("PROJECT NOT FOUND")
var ErrCommandNotRecognized = errors.New("COMMAND NOT RECOGNIZED")

func main() {
	args := os.Args[1:]
	var err error
	if len(args) >= 1 && args[0] == "server" {
		err = server.Run(args...)
	} else {
		err = client.Run(args...)
	}
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

// func cleanupStaleProjects(ctx context.Context, projects []*project.Project) error {
// 	logEvent := log.Track(ctx, "cleanupStaleProjects")
// 	projectsKilled := make([]string, 0)
// 	defer logEvent.Done()
// 	var errs error
// 	for _, p := range projects {
// 		status, err := p.Status()
// 		if err != nil {
// 			logEvent.Error(err)
// 			errs = errors.Join(errs, err)
// 		}
// 		if status == project.PROJECT_STATUS_ACTIVE {
// 			if passed, _ := p.TtlPassed(); passed {
// 				_, err := p.Kill(ctx)
// 				if err != nil {
// 					logEvent.Error(err)
// 					errs = errors.Join(errs, err)
// 				} else {
// 					projectsKilled = append(projectsKilled, p.Name)
// 				}
// 			}
// 		}
// 	}
// 	logEvent.Log("projectsKilled", projectsKilled)
// 	return errs
// }
