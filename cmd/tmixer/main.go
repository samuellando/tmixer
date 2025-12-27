package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/flags"
	"samuellando.com/tmixer/internal/fzf"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/tmux"
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
	if len(args) >= 1 {
		command = args[0]
	}
	if command == "notify-switch" {
		if len(args) < 2 || args[1] == tmux.CONTROL_SESSION_NAME {
			return nil
		}
	}
	tmux := tmux.Tmux()
	tmux.StartControlMode()
	defer tmux.StopControlMode()
	projects, err := project.List(tmux, config)
	if err != nil {
		return err
	}
	var selection *project.Project
	if command == "start" {
		if len(args) < 2 {
			if config.DefaultProject != nil {
				for _, p := range projects {
					if strings.HasPrefix(p.Name, *config.DefaultProject) {
						selection = p
						break
					}
				}
			}
		} else {
			for _, p := range projects {
				if strings.HasPrefix(p.Name, args[1]) {
					selection = p
					break
				}
			}
		}
	} else {
		if len(args) < 2 {
			selection, _ = fzf.PickProject(projects)
		} else {
			for _, p := range projects {
				if strings.HasPrefix(p.Name, args[1]) {
					selection = p
					break
				}
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
		err = startClient(selection)
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

func startClient(p *project.Project) error {
	if _, is_set := os.LookupEnv("TMUX"); is_set {
		return fmt.Errorf("Already in TMUX")
	}
	p.Start()
	cmd := exec.Command("tmux", "-u", "attach", "-t", p.TmuxSessionName())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return err
	}
	p.RunSwitchCommands()
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

All commands will by default open fzf if no project_name is provided. Except for 
the start command, which will start the configued default project in a new tmux client.

switch (default)
	switch the active tmux client to the project. It will eitehr switch to the 
	existing session and run it's configured switch comands or it will start
	a session for the project, open it's configured startup windows and 
	then switch to it.

	Note that a projects switch commands are automatically run any time you switch 
	to the session in tmux, even without this command. For example with leader-b.

start
	Equivalent to starting tmux normally, but will open into a project.

stop
	kill the session/project.

reset
	stop and restart the project session.

notify-switch 
	Internal command used to hook into tmux for when it switches session.
	tmixer automatically sets up the tmux hooks when it is run.

Flags: `)
	fmt.Println()
	fmt.Print(flags.HelpMessage(FLAGS))
}
