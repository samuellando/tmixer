package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/fzf"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/tmux"
)

var ERR_NO_SELECTION = errors.New("No selection made")

func main() {
	err := run(os.Args[1:]...)
	if err != nil {
		os.Exit(1)
	}
}

func run(args ...string) error {
	session := newSession()
	defer session.close()
	// init the logging event
	ctx := context.Background()
	ctx = log.InitializeWideEvent(ctx, &log.LoggerOptions{Level: log.LEVEL_INFO})
	// Parse the arguments before seetting up logging in case there's a extra log file
	config := config.New()
	_, err := flags.ParseArgs(ctx, args, FLAGS, config)
	if err != nil {
		fmt.Println(err)
		fmt.Println()
		displayHelp()
		return err
	}
	// Set up the logging
	logger, files, err := setupLogging(ctx, config)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer func() {
		for _, f := range files {
			f.Close()
		}
	}()
	// --help flag
	if config.DisplayHelp {
		displayHelp()
		logger.Info(ctx)
		return nil
	}
	// And run, and output the logs
	err = runTmixer(ctx, args, config, session)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		logger.Error(ctx, err)
		return err
	} else {
		logger.Info(ctx)
	}
	return nil
}

func runTmixer(ctx context.Context, args []string, config *config.Config, session *session) error {
	type runEvent struct {
		Command   string           `json:"command"`
		Selection *project.Project `json:"selection"`
		Errors    []string         `json:"errors"`
	}
	event := &runEvent{}
	finish := log.Track(ctx, "runEvent", event)
	defer finish()
	// Parse the config files, and the args
	err := config.LoadFiles(ctx)
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
	}
	args, err = flags.ParseArgs(ctx, args, FLAGS, config)
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
	}
	// Start the tmux server
	var srv *tmux.Server
	if config.TmuxSocketPath != nil {
		srv = tmux.Tmux(ctx, *config.TmuxSocketPath)
	} else {
		srv = tmux.Tmux(ctx)
	}
	err = srv.StartControlMode()
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
	} else {
		defer srv.StopControlMode()
	}
	// Critical parsing errors for configs should be displayed
	if len(event.Errors) != 0 {
		errs := strings.Join(event.Errors, ", ")
		err := srv.DisplayMessage(errs)
		if err != nil {
			errs = ", " + err.Error()
		}
		return errors.New(errs)
	}
	// Determine the command
	command := "switch"
	if len(args) >= 1 {
		command = args[0]
	}
	// We should ignore switch notifications on the control mode session
	event.Command = command
	if command == "notify-switch" {
		if len(args) < 2 || args[1] == tmux.CONTROL_SESSION_NAME {
			return nil
		}
	}
	// Now load all the projects
	projects, err := project.List(ctx, srv, config)
	if err != nil {
		return err
	}
	// Cleanup any projects that have passed the ttl
	err = cleanupStaleProjects(ctx, projects)
	if err != nil {
		return err
	}
	// Run any command that requires no selection
	if command == "list" {
		return fzf.DisplayProjects(ctx, projects, os.Stdout)
	}
	// Determine the selection
	var selection *project.Project
	if len(args) >= 2 {
		for _, p := range projects {
			if p.Name == args[1] {
				selection = p
				break
			}
		}
	} else {
		switch command {
		case "start":
			if config.DefaultProject != nil {
				for _, p := range projects {
					if strings.HasPrefix(p.Name, *config.DefaultProject) {
						selection = p
						break
					}
				}
			}
		case "reset":
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
		default:
			selection, _ = fzf.PickProject(ctx, config, projects)
		}
	}
	event.Selection = selection
	if selection == nil {
		event.Errors = append(event.Errors, ERR_NO_SELECTION.Error())
		return ERR_NO_SELECTION
	}
	// Execute the commend
	err = disableHooks(srv)
	if err != nil {
		return err
	}
	var cleanup func() error
	switch command {
	case "start":
		err = startClient(ctx, selection)
	case "switch":
		_, err = selection.Switch(ctx)
	case "kill":
		cleanup, err = selection.Kill(ctx)
		session.addCleanup(cleanup)
	case "reset":
		_, cleanup, err = selection.Reset(ctx)
		session.addCleanup(cleanup)
	case "notify-switch":
		err = selection.RunSwitchCommands(ctx)
	default:
		err = fmt.Errorf("Command not recognized: %s", command)
	}
	if err != nil {
		err = srv.DisplayMessage(err.Error())
		if err != nil {
			errors.Join(err, err)
		}
		return err
	}
	return setupHooks(srv)
}

func startClient(ctx context.Context, p *project.Project) error {
	_, err := p.Start(ctx)
	if err != nil {
		return err
	}
	if _, is_set := os.LookupEnv("TMUX"); is_set {
		return fmt.Errorf("Already in TMUX")
	}
	cmd := exec.Command("tmux", "-u", "attach", "-t", p.TmuxSessionName())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return err
	}
	p.RunSwitchCommands(ctx)
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

func setupHooks(tmux *tmux.Server) error {
	cmd := `run-shell 'tmixer notify-switch #{session_name}'`
	err := tmux.SetHook("client-session-changed[2000]", cmd)
	if err != nil {
		return err
	}
	err = tmux.SetHook("client-attached[2000]", cmd)
	if err != nil {
		return err
	}
	return nil
}

func disableHooks(tmux *tmux.Server) error {
	cmd := ``
	err := tmux.SetHook("client-session-changed[2000]", cmd)
	if err != nil {
		return err
	}
	err = tmux.SetHook("client-attached[2000]", cmd)
	if err != nil {
		return err
	}
	return nil
}

func setupLogging(ctx context.Context, config *config.Config) (*log.Logger, []*os.File, error) {
	files := []*os.File{}
	_, logger := log.New(ctx, &log.LoggerOptions{Level: config.LogLevel})

	retention := 24 * time.Hour * time.Duration(*config.LogRetentionDays)

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	logDir := filepath.Join(home, ".local/state/tmixer/logs")

	f, err := log.RotateLogFile(logDir, retention)
	if err != nil {
		return nil, nil, err
	}
	if f != nil {
		logger.AddSink(f)
		files = append(files, f)
	}

	if config.LogFile != nil {
		// Use log rotation with daily retention (24 hours)
		f, err := os.OpenFile(*config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, nil, err
		}
		logger.AddSink(f)
		files = append(files, f)
	}

	return logger, files, nil
}

func displayHelp() {
	fmt.Println(`tmixer [flags] [command] [project_name]

Commands:

All commands will by default open fzf if no project_name is provided. Except for 
the start command, which will start the configued default project in a new tmux client, 
and the reset command, which will reset the attached session.

switch (default)
	switch the active tmux client to the project. It will eitehr switch to the 
	existing session or start a session for the project, open it's configured 
	startup windows and then switch to it.

	After switching to a project/session, tmixer will run it's configured 
	switch commands.

	Note that a project's switch commands are automatically run any time you switch 
	to the session in tmux, even without this command. For example with leader-b.

start
	Equivalent to starting tmux normally, but will open into a project. By
	default it starts the defualt configured project.

kill
	kill the session/project.

reset
	kill and restart the project session. By default will reset the attached session.

Flags: `)
	fmt.Println()
	fmt.Print(flags.HelpMessage(FLAGS))
}

func cleanupStaleProjects(ctx context.Context, projects []*project.Project) error {
	type cleanupStaleProjectsEvent struct {
		ProjectsKilled []string `json:"projectsKilled,omitempty"`
		Errors         []string `json:"errors,omitempty"`
	}
	event := &cleanupStaleProjectsEvent{}
	finish := log.Track(ctx, "cleanupStaleProjectsEvent", event)
	defer finish()
	var errs error
	for _, p := range projects {
		status, err := p.Status()
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
			errs = errors.Join(errs, err)
		}
		if status == project.PROJECT_STATUS_ACTIVE {
			if passed, _ := p.TtlPassed(); passed {
				_, err := p.Kill(ctx)
				if err != nil {
					event.Errors = append(event.Errors, err.Error())
					errs = errors.Join(errs, err)
				} else {
					event.ProjectsKilled = append(event.ProjectsKilled, p.Name)
				}
			}
		}
	}
	return errs
}
