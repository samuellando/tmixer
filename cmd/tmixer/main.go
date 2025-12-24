package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"samuellando.com/tmixer/internal/configv2"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/fzf"
	"samuellando.com/tmixer/internal/projectv2"
	"samuellando.com/tmixer/internal/tmuxv2"
)

var ERR_NO_SELECTION = errors.New("No selection made")

func main() {
	config := config.New()
	args, err := flags.ParseArgs(os.Args, FLAGS, config)
	if err != nil {
		fmt.Println(err)
		fmt.Println()
		displayHelp()
		os.Exit(1)
	}
	if OPTION_DISPLAY_HELP {
		displayHelp()
		os.Exit(0)
	}
	f, err := setupLogging(config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if f != nil {
		defer f.Close()
	}
	err = config.LoadFiles()
	if err != nil {
		slog.Debug(err.Error())
	}
	err = run(args, config)
	if err == ERR_NO_SELECTION {
		fmt.Println(err)
		slog.Debug(err.Error())
		os.Exit(1)
	}
	if err != nil {
		fmt.Println(err)
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run(args []string, config *config.Config) error {
	command := "switch"
	if len(args) > 1 {
		command = args[0]
	}
	if command == "notify-switch" {
		if len(args) < 2 || args[1] == tmuxv2.CONTROL_SESSION_NAME {
			return nil
		} 
	}
	tmux := tmuxv2.Tmux()
	tmux.StartControlMode()
	defer tmux.StopControlMode()
	projects, err := projectv2.List(tmux, config)
	if err != nil {
		return err
	}
	var selection *projectv2.Project
	if len(args) < 2 || command == "start" {
		selection, _ = fzf.PickProject(projects)
	} else {
		for _, p := range projects {
			if strings.HasPrefix(p.Name, args[1]) {
				selection = p
				break
			}
		}
	}
	if selection == nil {
		return ERR_NO_SELECTION
	}
	err = disableHooks(tmux)
	if err != nil {
		return err
	}
	switch command {
	case "start":
		_, err = selection.Start()
	case "switch":
		err = selection.Switch()
	case "stop":
		err = selection.Stop()
	case "reset":
		_, err = selection.Reset()
	case "notify-switch":
		err = selection.RunSwitchCommands()
	default:
		err = fmt.Errorf("Command not recognized: %s", command)
	}
	if err != nil {
		return err
	}
	return setupHooks(tmux)
}

func setupHooks(tmux *tmuxv2.Server) error {
	cmd :=`run-shell 'tmixer notify-switch #{session_name}'`
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

func disableHooks(tmux *tmuxv2.Server) error {
	cmd :=``
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

func setupLogging(config *config.Config) (*os.File, error) {
	programLevel := new(slog.LevelVar)
	if config.LogFile != nil {
		programLevel.Set(slog.LevelDebug)
	}
	var logWriter io.Writer
	var file *os.File
	if config.LogFile != nil {
		f, err := os.OpenFile(*config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		logWriter = f
		file = f
	} else {
		logWriter = io.Discard
	}
	logger := slog.New(slog.NewTextHandler(logWriter, &slog.HandlerOptions{
		Level:     programLevel,
		AddSource: true,
	}))
	slog.SetDefault(logger)
	return file, nil
}

func displayHelp() {
	fmt.Println(`tmixer [flags] [command] [project_name]

Commands:

switch (default)
	switch the active tmux client to the session. If it has a configured project
	the project windows are crteated and the switch commands are run.

	Note that switching windows in tmux will also call the switch commands by utilizing 
	a hook, as long as tmixer was called aty some point on the server session.

start
	start the session, create project windows

stop
	kill the session

reset
	reset session, recreate project windows

notify-switch 
	Internal command used to hook into tmux for when it switches session
	tmixer automatically sets up the tmux hook when it is run.

If no project name is provided, fzf will be opened to select the project. For
the start command it will default to the last active session, or the default 
configured project if the tmux server has not started.

Flags: `)
	fmt.Println()
	fmt.Print(flags.HelpMessage(FLAGS))
}
