package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ProjectConfig struct {
	Name           string         `yaml:"name"`
	Directory      string         `yaml:"directory"`
	SubDirectories bool           `yaml:"subDirectories"`
	Windows        []WindowConfig `yaml:"windows"`
	SwitchCommands [][]string     `yaml:"switchCommands"`
}

func convertToAbsolutePaths(projects []*ProjectConfig) error {
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
