package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/display"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/fzf"
	logV2 "samuellando.com/tmixer/internal/log/v2"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/tmux"
)

var ErrNoSelection = errors.New("NO SELECTION MADE")
var ErrProjectNotFound = errors.New("PROJECT NOT FOUND")
var ErrCommandNotRecognized = errors.New("COMMAND NOT RECOGNIZED")

func main() {
	err := run(os.Args[1:]...)
	if err != nil {
		os.Exit(1)
	}
}

func run(args ...string) (err error) {
	session := newSession()
	defer func() {
		err = errors.Join(err, session.close())
	}()
	// init the logging event
	ctx := logV2.ContextLogger(context.Background())
	// Parse the arguments before setting up logging in case there's a extra log file
	session.config, args, err = flags.ParseArgs(ctx, args, FLAGS)
	if err != nil {
		fmt.Println(err)
		fmt.Println()
		displayHelp()
		return err
	}
	_ = session.config.LoadFiles(ctx)
	// Set up the logging
	err = setupLogging(ctx, session.config)
	if err != nil {
		fmt.Println(err)
		return err
	}
	// --help flag
	if session.config.DisplayHelp != nil && *session.config.DisplayHelp {
		displayHelp()
		return logV2.Done(ctx)
	}
	// And run, and output the logs
	err = runTmixer(ctx, args, session)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return logV2.Fatal(ctx, err)
	}
	return logV2.Done(ctx)
}

func runTmixer(ctx context.Context, args []string, session *session) error {
	logEvent := logV2.Track(ctx, "run")
	defer logEvent.Done()
	// Parse the config files, and the args, again
	srv, err := startTmuxServer(ctx, session)
	if err != nil {
		logEvent.Error(err)
		err = errors.Join(srv.DisplayMessage(err.Error()), err)
		return err
	}
	// Determine the command
	command := "switch"
	if len(args) >= 1 {
		command = args[0]
	}
	logEvent.Log("command", command)
	// We should ignore switch notifications on the control mode session
	if command == "notify-switch" {
		if len(args) < 2 || args[1] == tmux.CONTROL_SESSION_NAME {
			return nil
		}
	}
	// Now load all the projects
	session.projects, err = project.List(ctx, srv, session.config)
	if err != nil {
		logEvent.Error(err)
		return err
	}
	// Cleanup any projects that have passed the ttl
	err = cleanupStaleProjects(ctx, session.projects)
	if err != nil {
		logEvent.Error(err)
		return err
	}
	// Finally run the command
	query := ""
	if len(args) >= 2 {
		query = args[1]
	}
	err = disableHooks(srv)
	if err != nil {
		logEvent.Error(err)
		return err
	}
	err = executeCommand(ctx, srv, command, query, session)
	if err != nil {
		if err != ErrNoSelection {
			displayError := srv.DisplayMessage(err.Error())
			if displayError != nil {
				err = errors.Join(displayError, err)
			}
			logEvent.Error(err)
		}
		return err
	}
	return setupHooks(srv)
}

func executeCommand(ctx context.Context, srv *tmux.Server, command, query string, session *session) error {
	switch command {
	// Internal (undocumented) commands
	case "list":
		return list(ctx, session)
	case "notify-switch":
		return notifySwitch(ctx, query, session)
	case "start":
		return start(ctx, srv, query, session)
	case "switch":
		return runSwitch(ctx, query, session)
	case "kill":
		return kill(ctx, query, session)
	case "reset":
		return reset(ctx, query, session)
	default:
		return ErrCommandNotRecognized
	}
}

func list(ctx context.Context, session *session) error {
	infos, err := display.Projects(ctx, session.projects)
	if err != nil {
		return err
	}
	for _, i := range infos {
		fmt.Println(i)
	}
	return nil
}

func notifySwitch(ctx context.Context, query string, session *session) error {
	if query == "" {
		return ErrNoSelection
	}
	selection := getProject(query, session)
	if selection == nil {
		return ErrProjectNotFound
	}
	return selection.RunSwitchCommands(ctx)
}

func start(ctx context.Context, srv *tmux.Server, query string, session *session) (err error) {
	var selection *project.Project
	if query != "" {
		selection = getProject(query, session)
	} else {
		if session.config.DefaultProject != nil {
			selection = getProject(*session.config.DefaultProject, session)
		} else {
			selection, err = selectProject(ctx, session.config, session.projects)
			if err != nil {
				return err
			}
		}
	}
	if selection == nil {
		return ErrNoSelection
	}
	return startClient(ctx, srv, selection)
}

func runSwitch(ctx context.Context, query string, session *session) (err error) {
	var selection *project.Project
	if query != "" {
		selection = getProject(query, session)
	} else {
		selection, err = selectProject(ctx, session.config, session.projects)
		if err != nil {
			return err
		}
	}
	if selection == nil {
		return ErrNoSelection
	}
	_, err = selection.Switch(ctx)
	return err
}

func kill(ctx context.Context, query string, session *session) (err error) {
	var selection *project.Project
	if query != "" {
		selection = getProject(query, session)
	} else {
		selection, err = selectProject(ctx, session.config, session.projects)
		if err != nil {
			return err
		}
	}
	if selection == nil {
		return ErrNoSelection
	}
	cleanup, err := selection.Kill(ctx)
	session.addCleanup(cleanup)
	return err
}

func reset(ctx context.Context, query string, session *session) error {
	var selection *project.Project
	if query != "" {
		selection = getProject(query, session)
	} else {
		for _, p := range session.projects {
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
	session.addCleanup(cleanup)
	return err
}

func startTmuxServer(ctx context.Context, session *session) (*tmux.Server, error) {
	var srv *tmux.Server
	if session.config.TmuxSocketPath != nil {
		srv = tmux.Tmux(ctx, *session.config.TmuxSocketPath)
	} else {
		srv = tmux.Tmux(ctx)
	}
	err := srv.StartControlMode()
	if err != nil {
		return srv, err
	} else {
		session.addCleanup(srv.StopControlMode)
	}
	return srv, nil
}

func getProject(query string, session *session) *project.Project {
	for _, p := range session.projects {
		if p.Name == query {
			return p
		}
	}
	return nil
}

func selectProject(ctx context.Context, config *config.Config, projects []*project.Project) (*project.Project, error) {
	list, err := display.Projects(ctx, projects)
	if err != nil {
		return nil, err
	}
	out, err := fzf.Pick(ctx, config, list)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, ErrNoSelection
	}
	return display.GetProjectFromOutput(*out, projects)
}

func startClient(ctx context.Context, srv *tmux.Server, p *project.Project) error {
	_, err := p.Start(ctx)
	if err != nil {
		return err
	}
	if _, is_set := os.LookupEnv("TMUX"); is_set {
		_, err := p.Start(ctx)
		return err
	}
	cmd := exec.Command("tmux", "-u", "attach", "-t", p.TmuxSessionName())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return err
	}
	var errs error
	err = p.RunSwitchCommands(ctx)
	if err != nil {
		errs = errors.Join(errs, err)
		err = srv.DisplayMessage(err.Error())
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}
	err = cmd.Wait()
	if err != nil {
		errs = errors.Join(errs, err)
	}
	return errs
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

func setupLogging(ctx context.Context, config *config.Config) error {
	if config.LogFile != nil {
		// Use log rotation with daily retention (24 hours)
		f, err := os.OpenFile(*config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		if err := logV2.AddSink(ctx, f); err != nil {
			err = errors.Join(err, f.Close())
			return fmt.Errorf("while adding log sink: %w", err)
		}
	}
	return nil
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
	logEvent := logV2.Track(ctx, "cleanupStaleProjects")
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
