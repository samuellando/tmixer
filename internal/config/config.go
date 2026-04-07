package config

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"samuellando.com/tmixer/internal/log"
)

type Config struct {
	DefaultProject  *string          `yaml:"defaultProject"`
	LogFile         *string          `yaml:"logFile"`
	Ttl             *string          `yaml:"ttl"`
	FzfFlags        []string         `yaml:"fzfFlags"`
	TmuxSocketPath  *string          `yaml:"tmuxSocketPath"`
	CombineProjects *bool            `yaml:"combineProjects"`
	Projects        []*ProjectConfig `yaml:"projects"`
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
	fzfFlags := a.FzfFlags
	if b.FzfFlags != nil {
		fzfFlags = b.FzfFlags
	}
	tmuxSocketPath := a.TmuxSocketPath
	if b.TmuxSocketPath != nil {
		tmuxSocketPath = b.TmuxSocketPath
	}
	combineProjects := a.CombineProjects
	if b.CombineProjects != nil {
		combineProjects = b.CombineProjects
	}
	projects := a.Projects
	if b.Projects != nil {
		projects = b.Projects
	}
	return &Config{
		DefaultProject:  defaultProject,
		LogFile:         logFile,
		Ttl:             ttl,
		FzfFlags:        fzfFlags,
		TmuxSocketPath:  tmuxSocketPath,
		CombineProjects: combineProjects,
		Projects:        projects,
	}
}

type FSref struct {
	FS   fs.FS
	Name string
}

// Loads all the configs from the files provided
// If combine projects is true it will combine all the projects from all the
// config files, otherwise it treats them as global options.
// Global options are overridden, in the order that the files are listed.
// The initial values in the config overrides all other config options if set.
func LoadFiles(ctx context.Context, files []FSref) (*Config, error) {
	event := log.Track(ctx, "configLoadEvent")
	defer event.Done()

	config := &Config{}

	allProjects := make([][]*ProjectConfig, 0)
	allProjects = append(allProjects, config.Projects)
	var errs error // Keep track of all errors and report at the end
	for _, f := range files {
		fileConfig, err := loadFile(f)
		if err != nil {
			if !os.IsNotExist(err) {
				errs = errors.Join(errs, err)
			}
			event.Error(err)
			continue
		}
		// Override the global values
		config = mergeConfigs(config, fileConfig)
		allProjects = append(allProjects, fileConfig.Projects)
	}

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
	return config, errs
}

func loadFile(f FSref) (*Config, error) {
	bytes, err := fs.ReadFile(f.FS, f.Name)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		err = fmt.Errorf("while parsing %s: %w", f.Name, err)
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
