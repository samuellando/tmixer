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
	LogLevel         *int             `yaml:"logLevel"`
	LogRetentionDays *int             `yaml:"logRetentionDays"`
	FzfFlags         []string         `yaml:"fzfFlags"`
	TmuxSocketPath   *string          `yaml:"tmuxSocketPath"`
	ConfigFiles      []string         `yaml:"configFiles"`
	CombineProjects  *bool            `yaml:"combineProjects"`
	Projects         []*ProjectConfig `yaml:"projects"`
	DisplayHelp      *bool            `yaml:"displayHelp"`
}

func mergeConfigs(a, b *Config) *Config {
	defaultProject := a.DefaultProject
	if b.DefaultProject != nil {
		defaultProject = b.DefaultProject
	}
	logFile := a.LogFile
	if b.LogFile != nil {
		logFile = b.LogFile
	}
	ttl := a.Ttl
	if b.Ttl != nil {
		ttl = b.Ttl
	}
	logLevel := a.LogLevel
	if b.LogLevel != nil {
		logLevel = b.LogLevel
	}
	logRetentionDays := a.LogRetentionDays
	if b.LogRetentionDays != nil {
		logRetentionDays = b.LogRetentionDays
	}
	fzfFlags := a.FzfFlags
	if b.FzfFlags != nil {
		fzfFlags = b.FzfFlags
	}
	tmuxSocketPath := a.TmuxSocketPath
	if b.TmuxSocketPath != nil {
		tmuxSocketPath = b.TmuxSocketPath
	}
	configFiles := a.ConfigFiles
	if b.ConfigFiles != nil {
		configFiles = b.ConfigFiles
	}
	combineProjects := a.CombineProjects
	if b.CombineProjects != nil {
		combineProjects = b.CombineProjects
	}
	projects := a.Projects
	if b.Projects != nil {
		projects = b.Projects
	}
	displayHelp := a.DisplayHelp
	if b.DisplayHelp != nil {
		displayHelp = b.DisplayHelp
	}
	return &Config{
		DefaultProject:   defaultProject,
		LogFile:          logFile,
		Ttl:              ttl,
		LogLevel:         logLevel,
		LogRetentionDays: logRetentionDays,
		FzfFlags:         fzfFlags,
		TmuxSocketPath:   tmuxSocketPath,
		ConfigFiles:      configFiles,
		CombineProjects:  combineProjects,
		Projects:         projects,
		DisplayHelp:      displayHelp,
	}
}

// Loads all the configs from the configFiles field
// If combine projects is true it will combine all the projects from all the
// config files, otherwise it treats them as global options.
// Global options are overridden, in the order that the files are listed.
// The initial values in the config overrides all other config options if set.
func (config *Config) LoadFiles(ctx context.Context) error {
	event := log.Track(ctx, "configLoadEvent")
	defer event.Done()

	allProjects := make([][]*ProjectConfig, 0)
	allProjects = append(allProjects, config.Projects)
	var errs error // Keep track of all errors and report at the end

	loadedConfig := &Config{}
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
		loadedConfig = mergeConfigs(loadedConfig, fileConfig)
		allProjects = append(allProjects, loadedConfig.Projects)
	}

	*config = *mergeConfigs(loadedConfig, config)

	if config.CombineProjects != nil && *config.CombineProjects {
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
