package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/fzf"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/tmux"
)

var ERR_NO_SELECTION = errors.New("No selection made")

var cleanupFuncs = []func() error{}

func main() {
	run(os.Args)
	for _, f := range cleanupFuncs {
		if f != nil {
			f()
		}
	}
}

func run(args []string) {
	ctx := context.Background()
	ctx = log.InitializeWideEvent(ctx, &log.LoggerOptions{Level: log.LEVEL_INFO})
	config := config.New()
	args, err := flags.ParseArgs(ctx, args, FLAGS, config)
	if err != nil {
		fmt.Println(err)
		fmt.Println()
		displayHelp()
		os.Exit(1)
	}
	logger, f, err := setupLogging(ctx, config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if f != nil {
		defer f.Close()
	}
	// Finally run
	if OPTION_DISPLAY_HELP {
		displayHelp()
		logger.Info(ctx)
		os.Exit(0)
	}
	config.LoadFiles(ctx)
	err = runTmixer(ctx, args, config)
	if err != nil {
		fmt.Println(err)
		logger.Error(ctx, err)
		os.Exit(1)
	} else {
		logger.Info(ctx)
	}
}

func runTmixer(ctx context.Context, args []string, config *config.Config) error {
	type runEvent struct {
		Command   string           `json:"command"`
		Selection *project.Project `json:"selection"`
		Errors    []string         `json:"errors"`
	}
	event := &runEvent{}
	finish := log.Track(ctx, "runEvent", event)
	defer finish()
	command := "switch"
	if len(args) >= 1 {
		command = args[0]
	}
	event.Command = command
	if command == "notify-switch" {
		if len(args) < 2 || args[1] == tmux.CONTROL_SESSION_NAME {
			return nil
		}
	}
	tmux := tmux.Tmux(ctx)
	tmux.StartControlMode()
	defer tmux.StopControlMode()
	projects, err := project.List(ctx, tmux, config)
	if err != nil {
		return err
	}
	var selection *project.Project
	if len(args) >= 2 {
		for _, p := range projects {
			if strings.HasPrefix(p.Name, args[1]) {
				selection = p
				break
			}
		}
	} else {
		switch command {
		case "list":
			return fzf.DisplayProjects(ctx, projects, os.Stdout)
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
	err = disableHooks(tmux)
	if err != nil {
		return err
	}
	var cleanup func() error
	switch command {
	case "start":
		err = startClient(ctx, selection)
	case "switch":
		err = selection.Switch(ctx)
	case "kill":
		cleanup, err = selection.Kill(ctx)
		cleanupFuncs = append(cleanupFuncs, cleanup)
	case "reset":
		_, cleanup, err = selection.Reset(ctx)
		cleanupFuncs = append(cleanupFuncs, cleanup)
	case "notify-switch":
		err = selection.RunSwitchCommands(ctx)
	default:
		err = fmt.Errorf("Command not recognized: %s", command)
	}
	if err != nil {
		return err
	}
	err = setupHooks(tmux)
	return err
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

func setupLogging(ctx context.Context, config *config.Config) (*log.Logger, *os.File, error) {
	var logWriter io.Writer
	var file *os.File
	if config.LogFile != nil {
		f, err := os.OpenFile(*config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, nil, err
		}
		logWriter = f
		file = f
	} else {
		logWriter = io.Discard
	}
	_, logger := log.New(ctx, logWriter, nil)
	return logger, file, nil
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
