package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"samuellando.com/tmixer/internal/log/v2"
)

type Config struct {
	DefaultProject   *string          `yaml:"defaultProject"`
	LogFile          *string          `yaml:"logFile"`
	Ttl              *string          `yaml:"ttl"`
	LogLevel         int              `yaml:"logLevel"`
	LogRetentionDays *int             `yaml:"logRetentionDays"`
	FzfFlags         []string         `yaml:"fzfFlags"`
	TmuxSocketPath   *string          `yaml:"tmuxSocketPath"`
	ConfigFiles      []string         `yaml:"configFiles"`
	CombineProjects  bool             `yaml:"combineProjects"`
	Projects         []*ProjectConfig `yaml:"projects"`
	DisplayHelp      bool             `yaml:"displayHelp"`
}

func (config *Config) LoadFiles(ctx context.Context) error {
	event := log.Track(ctx, "configLoadEvent")
	defer event.Done()

	allProjects := make([][]*ProjectConfig, 0)
	var errs error // Keep track of all errors and report at the end

	for _, f := range config.ConfigFiles {
		fileConfig, err := loadFile(f)
		if err != nil {
			if !os.IsNotExist(err) {
				errs = errors.Join(errs, err)
			}
			event.Error(err)
			continue
		}
		// Override the global values
		*config = *fileConfig
		allProjects = append(allProjects, config.Projects)
	}

	if config.CombineProjects {
		resultProjects := make([]*ProjectConfig, 0)
		for _, projects := range allProjects {
			resultProjects = append(resultProjects, projects...)
		}
		config.Projects = resultProjects
	}

	err := convertToAbsolutePaths(config.Projects)
	if err != nil {
		errs = errors.Join(errs, err)
		event.Error(err)
	}

	event.Log("result", config)
	return errs
}

func loadFile(f string) (*Config, error) {
	path, err := absPath(f)
	if err != nil {
		return nil, err
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		err = fmt.Errorf("while parsing %s: %w", path, err)
		return nil, err
	}
	return config, err
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
