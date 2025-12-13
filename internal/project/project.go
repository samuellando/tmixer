package project

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	DefaultProject *string             `yaml:"defaultProject"`
	Projects       map[string]*Project `yaml:"projects"`
}

type Project struct {
	Directory      string     `yaml:"directory"`
	SubDirectories bool       `yaml:"subDirectories"`
	StartupWindows []Window   `yaml:"startupWindows"`
	SwitchCommands []string `yaml:"switchCommands"`
}

type Window struct {
	Name    string
	Command string
}

func LoadCofig(files ...string) Config {
	config := Config{}
	projects := make(map[string]*Project)
	for _, f := range files {
		if f != "" {
			bytes, err := os.ReadFile(absPath(f))
			if err != nil {
				slog.Debug(err.Error())
				continue
			}
			err = yaml.Unmarshal(bytes, &config)
			if err != nil {
				slog.Debug(err.Error())
			}
			maps.Copy(projects, config.Projects)
		}
	}
	config.Projects = projects
	convertToAbsolutePaths(config.Projects)
	loadSubDirectories(config.Projects)
	return config
}

func absPath(path string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(homeDir, path[2:])
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		slog.Error(err.Error())
	}
	return abs
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
					p := Project{
						Directory:      filepath.Join(proj.Directory, f.Name()),
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
