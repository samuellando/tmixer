package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/tmux"
	"time"
)

var CONFIG_FILES = []string{"~/.tmixer.yml", "~/.config/tmixer/config.yml"}

// Command line flags
var help bool
var verbose bool
var config string
var command string
var inputProject string

func parseCommandLine() {
	flag.BoolVar(&help, "h", false, "Display this help message")
	flag.BoolVar(&verbose, "v", false, "Display debug logs")
	flag.StringVar(&config, "c", "", "Provide an additional config.yml file will by default read ~/.tmixer.yml and ~/.config/tmixer/config.yml")
	flag.Parse()
	if flag.NArg() == 0 {
		command = "switch"
	}
	command = flag.Arg(0)
	inputProject = flag.Arg(1)
}

func displayHelpMessage() {
	fmt.Println(`tmixer [flags] [command] [project_name]

Commands:
- switch (default)
- start
- stop
- reset

If no project name is provided, fzf will be opened to select the project. For
the start command it will default to the last active session, or the default 
configured project if the tmux server has not started.

Flags:`)
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println(`Config file example (~/.tmixer.yml):
"""
bin:
  diectory: "~/bin"
projects:
  diectory: "~/projects"
  subDirectories: true
  startupWindows:
	- nvim:
		command: "nivm ."
  startupCommands:
	- "ln -sfn /opt/example/config $(pwd)"
	- "ln -sfn $(pwd)/.nvim.lua ~/Projects/.nvim.lua"
"""`)
}

func Fzf(sessions map[string]*tmux.Session) *tmux.Session {
	cmd := exec.Command("fzf")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	go func() {
		defer stdin.Close()
		for _, session := range sessions {
			io.WriteString(stdin, session.String()+"\n")
		}
	}()
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	selected := tmux.RemoveIcon(string(out))
	return sessions[selected]
}

func main() {
	parseCommandLine()
	if help {
		displayHelpMessage()
		return
	}
	programLevel := new(slog.LevelVar)
	if verbose {
		programLevel.Set(slog.LevelDebug)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     programLevel,
		AddSource: true,
	}))
	slog.SetDefault(logger)
	slog.Debug(fmt.Sprintf("Parsed args: -h=%t -v=%t -c=%s %s %s", help, verbose, config, command, inputProject))

	err := tmux.StartServer()
	if err != nil {
		panic(err)
	}

	sessions := tmux.Sessions()

	config := project.LoadCofig(append(CONFIG_FILES, config)...)

	tmux.LinkProjectsToSessions(sessions, config.Projects)

	var selected *tmux.Session
	if inputProject != "" {
		var ok bool
		selected, ok = sessions[tmux.CleanName(inputProject)]
		if !ok {
			slog.Error("Provided project/session not found")
		}
	}

	if command == "start" {
		if selected == nil {
			slog.Debug("No project specified, seaching for last active")
			var lastActive time.Time
			for _, session := range sessions {
				if session.Active() {
					if t := *session.LastActivity(); t.After(lastActive) {
						lastActive = t
						selected = session
					}
				}
			}
		}
		if selected == nil {
			slog.Debug("No active sessions found, checking for default project")
			if config.DefaultProject != nil {
				selected = sessions[*config.DefaultProject]
			}
		}
		if selected == nil {
			slog.Debug("Defualt project not found, picking randomly")
			for _, session := range sessions {
				selected = session
				break
			}
		}
		if selected == nil {
			slog.Error("Cound not find a project or session to attach to")
			return
		}
		if !selected.Active() {
			selected.Start()
		}
		if _, is_set := os.LookupEnv("TMUX"); is_set {
			slog.Error("Already in TMUX")
		} else {
			slog.Debug("Starting tmux")
			wait := make(chan bool)
			go func() {
				cmd := exec.Command("tmux", "-u", "attach", "-t", selected.Name)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				if err != nil {
					panic(err)
				}
				wait <- true
			}()
			<-wait
		}
		return
	}

	if selected == nil {
		slog.Debug("Calling fzf")
		selected = Fzf(sessions)
	}
	if selected != nil {
		slog.Debug(fmt.Sprintf("Session %s was selected", selected.Name))
		selected.Swap()
	} else {
		fmt.Println("No selection")
	}
}
