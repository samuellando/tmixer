package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"samuellando.com/tmixer/cmd/tmixer/client"
	"samuellando.com/tmixer/cmd/tmixer/server"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
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
		os.Exit(1)
	}
}

func displayHelp() {
	fmt.Println(`tmixer [flags] [command] [project_name]

Commands:

All commands will by default open fzf if no project_name is provided. Except for 
the start command, which will start the configured default project in a new tmux client, 
and the reset command, which will reset the attached session.

switch (default)
	switch the active tmux client to the project. It will either switch to the 
	existing session or start a session for the project.

	After switching to a project/session, tmixer will run it's configured 
	switch commands.

	Note that a project's switch commands are automatically run any time you switch 
	to the session in tmux. For example even with leader-b.

start
	Equivalent to starting tmux normally, but will open into a project. By
	default it starts the default configured project.

kill
	kill the project.

reset
	reset the state of the project's session to it's configured state.

Flags: `)
	fmt.Println()
	fmt.Print(flags.HelpMessage(FLAGS))
}

func cleanupStaleProjects(ctx context.Context, projects []*project.Project) error {
	logEvent := log.Track(ctx, "cleanupStaleProjects")
	projectsKilled := make([]string, 0)
	defer logEvent.Done()
	var errs error
	for _, p := range projects {
		status, err := p.Status()
		if err != nil {
			logEvent.Error(err)
			errs = errors.Join(errs, err)
		}
		if status == project.PROJECT_STATUS_ACTIVE {
			if passed, _ := p.TtlPassed(); passed {
				_, err := p.Kill(ctx)
				if err != nil {
					logEvent.Error(err)
					errs = errors.Join(errs, err)
				} else {
					projectsKilled = append(projectsKilled, p.Name)
				}
			}
		}
	}
	logEvent.Log("projectsKilled", projectsKilled)
	return errs
}
