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
)

var CONFIG_FILES = []string{"~/.tmixer.yml", "~/.config/tmixer/config.yml"}

// Command line flags
var help bool
var verbose bool
var reset bool
var config string
var inputProject string

func parseCommandLine() {
	flag.BoolVar(&help, "h", false, "Display this help message")
	flag.BoolVar(&verbose, "v", false, "Display debug logs")
	flag.BoolVar(&reset, "r", false, "Reset the session for the project, will use the current project's session if not specified")
	flag.StringVar(&config, "c", "", "Provide an additional config.yml file will by default read ~/.tmixer.yml and ~/.config/tmixer/config.yml")
	flag.Parse()
	inputProject = flag.Arg(0)
}

func displayHelpMessage() {
	fmt.Println("**************** Tmixer ****************")
	fmt.Println("Quickly switch between your projects")
	fmt.Println("----------------------------------------")
	fmt.Println()
	fmt.Println("tmixer [flags] [project_name]")
	fmt.Println()
	fmt.Println("flags:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println()
	fmt.Print(`Config file example (~/.config.yml):
-------------------------------------------
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
-------------------------------------------`)
}

func Fzf(sessions map[string]*tmux.Session) *tmux.Session {
	cmd := exec.Command("fzf")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(stdin)
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
	err := tmux.StartServer()
	if err != nil {
		panic(err)
	}

	sessions := tmux.Sessions()

	projects := project.LoadCofig(append(CONFIG_FILES, config)...)

	tmux.LinkProjectsToSessions(sessions, projects)

	var selected *tmux.Session
	if inputProject != "" {
		selected = sessions[inputProject]
	} else {
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
