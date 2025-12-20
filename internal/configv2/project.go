package config

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	DefaultProject *string                   `yaml:"defaultProject"`
	Projects       map[string]*ProjectConfig `yaml:"projects"`
}

type ProjectConfig struct {
	Directory      string         `yaml:"directory"`
	SubDirectories bool           `yaml:"subDirectories"`
	StartupWindows []WindowConfig `yaml:"startupWindows"`
	SwitchCommands []string       `yaml:"switchCommands"`
}

type WindowConfig struct {
	Name    string
	Command string
}

func LoadCofig(files ...string) (*Config, error) {
	config := &Config{}
	projects := make(map[string]*ProjectConfig)
	for _, f := range files {
		if f != "" {
			path, err := absPath(f)
			if err != nil {
				return nil, fmt.Errorf("while getting abs path of config file: %w", err)
			}
			bytes, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("while reading config file: %w", err)
			}
			err = yaml.Unmarshal(bytes, &config)
			if err != nil {
				return nil, fmt.Errorf("while unmarshaling yaml: %w", err)
			}
			maps.Copy(projects, config.Projects)
		}
	}
	config.Projects = projects
	err := convertToAbsolutePaths(config.Projects)
	if err != nil {
		return nil, err
	}
	return config, nil
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
