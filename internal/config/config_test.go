package config_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"samuellando.com/tmixer/internal/config"
)

func Ptr[T any](v T) *T {
	return &v
}

func TestParseConfig(t *testing.T) {
	configText := `
defaultProject: home
logFile: "/tmp/tmixer-log.json"
ttl: 24h
fzfFlags: ["--ansi", "--bind"]
tmuxSocketPath: "/tmp/sock"
configFiles: ["one", "two"]
combineProjects: false
displayHelp: true

projects:
  - name: home
    directory: "/"

  - name: bin
    directory: "/bin"

  - name: projects
    directory: "/Projects"
    subDirectories: true

    windows:
      - command: ["nvim", "."]
      - name: "shell"
        panes:
          - command: ["test"]
            split: vertical
      - command: ["ls"]
`
	configFile := filepath.Join(t.TempDir(), "config.yml")
	err := os.WriteFile(configFile, []byte(configText), 0755)
	if err != nil {
		t.Fatal(err)
	}
	expected := config.Config{
		DefaultProject:  Ptr("home"),
		LogFile:         Ptr("/tmp/tmixer-log.json"),
		Ttl:             Ptr("24h"),
		FzfFlags:        []string{"--ansi", "--bind"},
		TmuxSocketPath:  Ptr("/tmp/sock"),
		ConfigFiles:     []string{configFile},
		CombineProjects: Ptr(false),
		Projects: []*config.ProjectConfig{
			{
				Name:      "home",
				Directory: "/",
			},
			{
				Name:           "bin",
				Directory:      "/bin",
				SubDirectories: false,
			},
			{
				Name:           "projects",
				Directory:      "/Projects",
				SubDirectories: true,
				Windows: []config.WindowConfig{
					{
						Command: []string{"nvim", "."},
					},
					{
						Name: Ptr("shell"),
						Panes: []config.PaneConfig{
							{
								Command: []string{"test"},
								Split:   Ptr("vertical"),
							},
						},
					},
					{
						Command: []string{"ls"},
					},
				},
			},
		},
		DisplayHelp: Ptr(true),
	}
	config := config.Config{
		ConfigFiles: []string{configFile},
	}
	err = config.LoadFiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(config, expected); diff != "" {
		t.Errorf("Config does not match\n%s", diff)
	}
}

func TestCombineProjects(t *testing.T) {
	configText1 := `
projects:
  - name: home
    directory: "/"
`
	configText2 := `
projects:
  - name: bin
    directory: "/bin"
`
	configFile1 := filepath.Join(t.TempDir(), "config1.yml")
	err := os.WriteFile(configFile1, []byte(configText1), 0755)
	if err != nil {
		t.Fatal(err)
	}
	configFile2 := filepath.Join(t.TempDir(), "config2.yml")
	err = os.WriteFile(configFile2, []byte(configText2), 0755)
	if err != nil {
		t.Fatal(err)
	}
	expected := config.Config{
		ConfigFiles:     []string{configFile1, configFile2},
		CombineProjects: Ptr(true),
		Projects: []*config.ProjectConfig{
			{
				Name:      "home",
				Directory: "/",
			},
			{
				Name:           "bin",
				Directory:      "/bin",
				SubDirectories: false,
			},
		},
	}
	config := config.Config{
		ConfigFiles:     []string{configFile1, configFile2},
		CombineProjects: Ptr(true),
	}
	err = config.LoadFiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(config, expected); diff != "" {
		t.Errorf("Config does not match\n%s", diff)
	}
}

func TestNoCombineProjects(t *testing.T) {
	configText1 := `
projects:
  - name: home
    directory: "/"
`
	configText2 := `
projects:
  - name: bin
    directory: "/bin"
`
	configFile1 := filepath.Join(t.TempDir(), "config1.yml")
	err := os.WriteFile(configFile1, []byte(configText1), 0755)
	if err != nil {
		t.Fatal(err)
	}
	configFile2 := filepath.Join(t.TempDir(), "config2.yml")
	err = os.WriteFile(configFile2, []byte(configText2), 0755)
	if err != nil {
		t.Fatal(err)
	}
	expected := config.Config{
		ConfigFiles:     []string{configFile1, configFile2},
		CombineProjects: Ptr(false),
		Projects: []*config.ProjectConfig{
			{
				Name:           "bin",
				Directory:      "/bin",
				SubDirectories: false,
			},
		},
	}
	config := config.Config{
		ConfigFiles:     []string{configFile1, configFile2},
		CombineProjects: Ptr(false),
	}
	err = config.LoadFiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(config, expected); diff != "" {
		t.Errorf("Config does not match\n%s", diff)
	}
}

func TestGlobalOptionHierarchy(t *testing.T) {
	configText1 := `
defaultProject: "one"
logFile: "one"
ttl: "one"
`
	configText2 := `
defaultProject: "two"
logFile: "two"
`
	configFile1 := filepath.Join(t.TempDir(), "config1.yml")
	err := os.WriteFile(configFile1, []byte(configText1), 0755)
	if err != nil {
		t.Fatal(err)
	}
	configFile2 := filepath.Join(t.TempDir(), "config2.yml")
	err = os.WriteFile(configFile2, []byte(configText2), 0755)
	if err != nil {
		t.Fatal(err)
	}
	expected := config.Config{
		DefaultProject: Ptr("three"),
		LogFile:        Ptr("two"),
		Ttl:            Ptr("one"),
		ConfigFiles:    []string{configFile1, configFile2},
	}
	config := config.Config{
		DefaultProject: Ptr("three"),
		ConfigFiles:    []string{configFile1, configFile2},
	}
	err = config.LoadFiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(config, expected); diff != "" {
		t.Errorf("Config does not match\n%s", diff)
	}
}
