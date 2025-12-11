package project

import (
	"log/slog"
	"maps"
	"github.com/goccy/go-yaml"
	"path/filepath"
	"os"
	"fmt"
	"strings"
)

type Project struct {
	Directory      string                   `yaml:"directory"`
	SubDirectories bool                     `yaml:"subDirectories"`
	StartupWindows []map[string]Window `yaml:"startupWindows"`
	SwitchCommands [][]string               `yaml:"switchCommands"`
}

type Window struct {
	Command string
}


func LoadCofig(files ...string) map[string]*Project {
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
