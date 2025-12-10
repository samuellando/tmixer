package main

import (
	"flag"
	"strings"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"log/slog"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

var CONFIG_FILES = []string{"~/.tmixer.yml", "~/.config/tmixer/config.yml"}

// Command line flags
var help bool
var verbose bool
var reset bool
var config string
var inputProject string

type Session struct {
	Name     string
	Active   bool
	Attached bool
	Project  *Project
}

type Project struct {
	Directory      string              `yaml:"directory"`
	SubDirectories bool                `yaml:"subDirectories"`
	StartupWindows []map[string]Window `yaml:"startupWindows"`
	SwitchCommands [][]string          `yaml:"switchCommands"`
}

type Window struct {
	Command string
}

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

func tmux(args ...string) ([]byte, error) {
	cmd := exec.Command("tmux", args...)
	return cmd.Output()
}

func tmuxStartServer() error {
	_, err := tmux("start-server")
	return err
}

func loadCofigProjects(files ...string) map[string]*Project {
	projects := make(map[string]*Project)
	for _, f := range files {
		if f != "" {
			bytes, err := os.ReadFile(f)
			if err != nil {
				slog.Debug(err.Error())
				continue
			}
			data := make(map[string]*Project)
			err = yaml.Unmarshal(bytes, &data)
			if err != nil {
				slog.Debug(err.Error())
			}
			maps.Copy(projects, data)
		}
	}
	convertToAbsolutePaths(projects)
	loadSubDirectories(projects)
	return projects
}

func convertToAbsolutePaths(projects map[string]*Project) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	for name, proj := range projects {
		if strings.HasPrefix(proj.Directory, "~/") {
			proj.Directory = filepath.Join(homeDir, proj.Directory[2:])
		}
		abs, err := filepath.Abs(proj.Directory)
		if err != nil {
			delete(projects, name)
			slog.Error(err.Error())
		}
		proj.Directory = abs
	}
}

func loadSubDirectories(projects map[string]*Project) {
	for name, proj := range projects {
		if proj.SubDirectories {
			delete(projects, name)
			dirEntries, err := os.ReadDir(proj.Directory)
			if err != nil {
				slog.Error(err.Error())
				continue
			}
			for _, f := range dirEntries {
				if f.IsDir() {
					p := Project {
						Directory: filepath.Join(proj.Directory, f.Name()),
						SubDirectories: false,
						StartupWindows: proj.StartupWindows,
						SwitchCommands: proj.SwitchCommands,
					}
					projects[fmt.Sprintf("%s--%s", name, f.Name())] = &p
				}
			}
		}
	}
}

func tmuxSessions() []Session {
	return nil
}

func main() {
	parseCommandLine()
	if help {
		displayHelpMessage()
		return
	}
	if verbose {
		programLevel := new(slog.LevelVar)
		programLevel.Set(slog.LevelDebug)
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: programLevel}))
		slog.SetDefault(logger)
	}

	projects := loadCofigProjects(append(CONFIG_FILES, config)...)

	for n, p := range projects {
		fmt.Println(n, *p)
	}

	err := tmuxStartServer()
	if err != nil {
		panic(err)
	}
	if inputProject != "" {
		fmt.Println("Swap project")
	} else {
		fmt.Println("Run fzf")
	}
}
