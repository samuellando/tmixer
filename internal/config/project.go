package config

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"samuellando.com/tmixer/internal/log"
)

type Config struct {
	DefaultProject  *string                   `yaml:"defaultProject"`
	LogFile         *string                   `yaml:"logFile"`
	FzfFlags        []string                  `yaml:"fzfFlags"`
	ConfigFiles     []string                  `yaml:"configFiles"`
	CombineProjects bool                      `yaml:"combineProjects"`
	Projects        map[string]*ProjectConfig `yaml:"projects"`
}

type ProjectConfig struct {
	Directory      string         `yaml:"directory"`
	SubDirectories bool           `yaml:"subDirectories"`
	Windows        []WindowConfig `yaml:"windows"`
	SwitchCommands []string       `yaml:"switchCommands"`
}

type WindowConfig struct {
	Name    string       `yaml:"name"`
	Command *string      `yaml:"command"`
	Panes   []PaneConfig `yaml:"panes"`
}

type PaneConfig struct {
	Command *string `yaml:"command"`
	Split   *string `yaml:"split"`
}

func New() *Config {
	return &Config{
		DefaultProject: nil,
		LogFile:        nil,
		FzfFlags: []string{
			"--ansi",
			"--bind", "ctrl-k:execute(tmixer kill {2})+reload(tmixer list)",
			"--bind", "ctrl-r:execute(tmixer reset {2})+reload(tmixer list)",
			"--bind", "ctrl-s:execute(tmixer start {2})+reload(tmixer list)",
		},
		ConfigFiles:     []string{"~/.config/tmixer/config.yml", "~/.tmixer.yml"},
		CombineProjects: true,
		Projects:        make(map[string]*ProjectConfig),
	}
}

func (config *Config) LoadFiles(ctx context.Context) error {
	type configLoadEvent struct {
		Result *Config  `json:"result"`
		Errors []string `json:"errors"`
	}
	event := &configLoadEvent{}
	finish := log.Track(ctx, "configLoadEvent", event)
	defer finish()

	allProjects := make([]map[string]*ProjectConfig, 0)
	var errs error
	for _, f := range config.ConfigFiles {
		path, err := absPath(f)
		if err != nil {
			errs = errors.Join(errs, err)
			event.Errors = append(event.Errors, err.Error())
			continue
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			errs = errors.Join(errs, err)
			event.Errors = append(event.Errors, err.Error())
			continue
		}
		err = yaml.Unmarshal(bytes, config)
		if err != nil {
			errs = errors.Join(errs, err)
			event.Errors = append(event.Errors, err.Error())
			continue
		}
		allProjects = append(allProjects, config.Projects)
	}
	if config.CombineProjects {
		resultProjects := make(map[string]*ProjectConfig)
		for _, projects := range allProjects {
			maps.Copy(resultProjects, projects)
		}
		config.Projects = resultProjects
	}
	err := convertToAbsolutePaths(config.Projects)
	if err != nil {
		errs = errors.Join(errs, err)
		event.Errors = append(event.Errors, err.Error())
	}
	event.Result = config
	return errs
}

func convertToAbsolutePaths(projects map[string]*ProjectConfig) error {
	for _, proj := range projects {
		path, err := absPath(proj.Directory)
		if err != nil {
			return err
		}
		proj.Directory = path
	}
	return nil
}

func absPath(path string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("while getting home dir: %w", err)
	}
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(homeDir, path[2:])
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("while converting to abs path: %w", err)
	}
	return abs, nil
}
