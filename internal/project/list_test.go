package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

type listTestCase struct {
	config   *config.Config
	projects []string
}

func runAllListTestCases(t *testing.T, f func(ctx context.Context, srv *tmux.Server, tc listTestCase)) {
	t.Parallel()
	// Need to reset in bettwen for each test case to avoid switches
	for name, tc := range getAllListTestCases() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
				f(ctx, srv, tc(t))
			})
		})
	}
}

func getAllListTestCases() map[string]func(*testing.T) listTestCase {
	n := 10
	createProjects := func(t *testing.T, dir string) {
		for i := range n {
			os.Mkdir(filepath.Join(dir, strconv.Itoa(i)), 0o700)
			f, err := os.OpenFile(filepath.Join(dir, "file"+strconv.Itoa(i)), os.O_RDONLY|os.O_CREATE, 0o644)
			if err != nil {
				t.Error(err)
			}
			f.Close()
		}
	}

	var testCases = map[string]func(t *testing.T) listTestCase{
		"no conig": func(t *testing.T) listTestCase {
			return listTestCase{
				config:   &config.Config{},
				projects: []string{testutil.DEFAULT_TEST_SESSION},
			}
		},
		"one config": func(t *testing.T) listTestCase {
			return listTestCase{
				config: &config.Config{
					Projects: []*config.ProjectConfig{
						{
							Name:      "bin",
							Directory: "/home/test/bin",
						},
					},
				},
				projects: []string{testutil.DEFAULT_TEST_SESSION, "bin"},
			}
		},
		"subdir config": func(t *testing.T) listTestCase {
			dir := t.TempDir()
			createProjects(t, dir)
			return listTestCase{
				config: &config.Config{
					Projects: []*config.ProjectConfig{
						{
							Name:           "projects",
							Directory:      dir,
							SubDirectories: true,
						},
					},
				},
				projects: []string{
					testutil.DEFAULT_TEST_SESSION,
					"projects--0",
					"projects--1",
					"projects--2",
					"projects--3",
					"projects--4",
					"projects--5",
					"projects--6",
					"projects--7",
					"projects--8",
					"projects--9",
				},
			}
		},
		"combined config": func(t *testing.T) listTestCase {
			dir := t.TempDir()
			createProjects(t, dir)
			return listTestCase{
				config: &config.Config{
					Projects: []*config.ProjectConfig{
						{
							Name:      "bin",
							Directory: "/home/test/bin",
						},
						{
							Name:           "projects",
							Directory:      dir,
							SubDirectories: true,
						},
					},
				},
				projects: []string{
					testutil.DEFAULT_TEST_SESSION,
					"bin",
					"projects--0",
					"projects--1",
					"projects--2",
					"projects--3",
					"projects--4",
					"projects--5",
					"projects--6",
					"projects--7",
					"projects--8",
					"projects--9",
				},
			}
		},
	}
	return testCases
}

func TestListsAll(t *testing.T) {
	runAllListTestCases(t, func(ctx context.Context, srv *tmux.Server, tc listTestCase) {
		projects, err := List(ctx, srv, tc.config)
		if err != nil {
			t.Error(err)
		}
		if len(projects) != len(tc.projects) {
			t.Errorf("Project list does not match expected len: %d != %d", len(projects), len(tc.projects))
		}
		matched := make(map[string]bool)
		for _, p := range projects {
			matched[p.Name] = true
		}
		for _, p := range tc.projects {
			if !matched[p] {
				t.Errorf("Project %s not listed", p)
			}
		}
	})
}

func TestSetsFields(t *testing.T) {
	runAllListTestCases(t, func(ctx context.Context, srv *tmux.Server, tc listTestCase) {
		projects, err := List(ctx, srv, tc.config)
		if err != nil {
			t.Error(err)
		}
		for _, p := range projects {
			if p.Name == "" {
				t.Error("Project has empty name")
			}
			if p.Name == testutil.DEFAULT_TEST_SESSION {
				if p.Config != nil {
					t.Error("session project should not have config")
				}
			} else if p.Config == nil {
				t.Error("Project has empty config")
			}
		}
	})
}

func TestListIncludesAllSessions(t *testing.T) {
	n := 10
	runAllListTestCases(t, func(ctx context.Context, srv *tmux.Server, tc listTestCase) {
		for i := range n {
			srv.New("test-" + strconv.Itoa(i))
		}
		projects, err := List(ctx, srv, tc.config)
		if err != nil {
			t.Error(err)
		}
		if len(projects) != len(tc.projects)+n {
			t.Errorf("Got incorrect number of projects: %d != %d + %d", len(projects), len(tc.projects), n)
		}
		matched := make(map[string]bool)
		for _, p := range projects {
			matched[p.Name] = true
		}
		for i := range n {
			if !matched["test-"+strconv.Itoa(i)] {
				t.Error("missing session project")
			}
		}
	})
}

func TestListMatchesExistingSessions(t *testing.T) {
	runAllListTestCases(t, func(ctx context.Context, srv *tmux.Server, tc listTestCase) {
		projects, err := List(ctx, srv, tc.config)
		if err != nil {
			t.Error(err)
		}
		for _, p := range projects {
			if p.Name != testutil.DEFAULT_TEST_SESSION {
				_, err := srv.New(p.Name)
				if err != nil {
					t.Error(err)
				}
			}
		}
		projects, err = List(ctx, srv, tc.config)
		if err != nil {
			t.Error(err)
		}
		if len(projects) != len(tc.projects) {
			t.Errorf("Project list does not match expected len: %d != %d", len(projects), len(tc.projects))
		}
		matched := make(map[string]bool)
		for _, p := range projects {
			matched[p.Name] = true
		}
		for _, p := range tc.projects {
			if !matched[p] {
				t.Errorf("Project %s not listed", p)
			}
		}
	})
}

func TestAmbiguousNames(t *testing.T) {
	dir := t.TempDir()
	n := 10
	for i := range n {
		os.Mkdir(filepath.Join(dir, strconv.Itoa(i)), 0o700)
	}
	config := &config.Config{
		Projects: []*config.ProjectConfig{
			{
				Name:      "projects--5",
				Directory: "/home/test/bin",
			},
			{
				Name:           "projects",
				Directory:      dir,
				SubDirectories: true,
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		_, err := List(ctx, srv, config)
		if !errors.Is(err, ErrAmbiguousName) {
			t.Errorf("wrong error returned %v", err)
		}
	})
}

func TestListNoTmux(t *testing.T) {
	ctx, _ := log.New(context.Background(), nil)
	for name, tcf := range getAllListTestCases() {
		t.Run(name, func(t *testing.T) {
			tc := tcf(t)
			projects, err := List(ctx, nil, tc.config)
			if err != nil {
				t.Error(err)
			}
			if len(projects) != len(tc.projects)-1 {
				t.Errorf("Project list does not match expected len: %d != %d - 1", len(projects), len(tc.projects))
			}
			matched := make(map[string]bool)
			for _, p := range projects {
				matched[p.Name] = true
			}
			for _, p := range tc.projects {
				if !matched[p] && p != testutil.DEFAULT_TEST_SESSION {
					t.Errorf("Project %s not listed", p)
				}
			}
		})
	}
}

// List should log errors, and results only in debug mode
func TestListLogs(t *testing.T) {
	dir := t.TempDir()
	n := 10
	for i := range n {
		os.Mkdir(filepath.Join(dir, strconv.Itoa(i)), 0o700)
	}
	config := &config.Config{
		Projects: []*config.ProjectConfig{
			{
				Name:      "projects--5",
				Directory: "/home/test/bin",
			},
			{
				Name:           "projects",
				Directory:      dir,
				SubDirectories: true,
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		_, logger := log.New(ctx, nil)
		out := bytes.Buffer{}
		logger.AddSink(&out)
		_, err := List(ctx, srv, config)
		if !errors.Is(err, ErrAmbiguousName) {
			t.Errorf("wrong error returned %v", err)
		}
		logger.Info(ctx)
		res := make(map[string]any)
		json.Unmarshal(out.Bytes(), &res)
		errs := res["projectListEvent"].(map[string]any)["errors"].([]any)
		if len(errs) != 1 {
			t.Error("Should have one error")
		}
		if !strings.Contains(errs[0].(string), ErrAmbiguousName.Error()) {
			t.Error("Should contain a ErrAmbiguousName")
		}
		if _, ok := res["projectListResult"]; ok {
			t.Error("Should not log results at default log level")
		}
	})
}

func TestListLogsProjectConfigs(t *testing.T) {
	t.Parallel()

	validDir := t.TempDir()
	invalidDir := t.TempDir()
	config := &config.Config{
		Projects: []*config.ProjectConfig{
			{
				Name:      "valid-project",
				Directory: validDir,
			},
			{
				Name:      "invalid-project",
				Directory: invalidDir,
			},
		},
	}

	validContent := `windows:
  - name: overridden-terminal
    command: ["overridden-bash"]
switchCommands:
  - ["overridden-pwd"]`
	if err := os.WriteFile(filepath.Join(validDir, ".tmixer.yml"), []byte(validContent), 0o644); err != nil {
		t.Fatal(err)
	}

	invalidContent := `windows:
  - name: editor
    invalid_yaml_syntax: [unclosed bracket`
	if err := os.WriteFile(filepath.Join(invalidDir, ".tmixer.yml"), []byte(invalidContent), 0o644); err != nil {
		t.Fatal(err)
	}

	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_INFO)
		_, err := List(ctx, srv, config)
		if err != nil {
			t.Error(err)
		}

		res := testutil.GetLogEvent(ctx, logger, out)
		event := res["loadProjectConfigsEvent"].(map[string]any)

		loadedProjects := event["loadedProjects"].([]any)
		if len(loadedProjects) != 1 || loadedProjects[0].(string) != "valid-project" {
			t.Errorf("Expected valid-project in loadedProjects, got %v", loadedProjects)
		}

		errs := event["errors"].([]any)
		if len(errs) != 1 {
			t.Errorf("Expected 1 error, got %d", len(errs))
		}
		if !strings.Contains(errs[0].(string), filepath.Join(invalidDir, ".tmixer.yml")) {
			t.Errorf("Expected error to include invalid config path, got %v", errs[0])
		}
	})
}

func TestListLogsResult(t *testing.T) {
	runAllListTestCases(t, func(ctx context.Context, srv *tmux.Server, tc listTestCase) {
		ctx, logger := log.New(ctx, &log.LoggerOptions{Level: log.LEVEL_DEBUG})
		out := bytes.Buffer{}
		logger.AddSink(&out)
		_, err := List(ctx, srv, tc.config)
		if err != nil {
			t.Error(err)
		}
		logger.Info(ctx)
		res := make(map[string]any)
		json.Unmarshal(out.Bytes(), &res)
		event := res["projectListResult"].(map[string]any)
		list := event["result"].([]any)
		seen := make(map[string]bool)
		for _, p := range list {
			seen[p.(map[string]any)["Name"].(string)] = true
		}
		for _, p := range tc.projects {
			if !seen[p] {
				t.Errorf("Project %s was not listed", p)
			}
		}
	})
}

func BenchmarkList(b *testing.B) {
	dir := b.TempDir()
	n := 100
	for i := range n {
		os.Mkdir(filepath.Join(dir, strconv.Itoa(i)), 0o700)
	}
	config := &config.Config{
		Projects: []*config.ProjectConfig{
			{
				Name:      "bin",
				Directory: "/home/test/bin",
			},
			{
				Name:           "projects",
				Directory:      dir,
				SubDirectories: true,
			},
		},
	}
	ctx, tmux := testutil.SetupTestServer(b)
	defer testutil.TeardownTestServer(tmux)
	for i := range 10 {
		tmux.New("test-" + strconv.Itoa(i))
	}
	for b.Loop() {
		projects, err := List(ctx, tmux, config)
		if err != nil {
			b.Fatal(err)
		}
		if len(projects) < 110 {
			b.Fatal("Missing projects")
		}
	}
}

func TestProjectSpecificConfigs(t *testing.T) {
	runAllListTestCases(t, func(ctx context.Context, srv *tmux.Server, tc listTestCase) {
		initialProjects, err := List(ctx, srv, tc.config)
		if err != nil {
			t.Error(err)
		}
		if len(initialProjects) == 0 {
			t.Error("No projects returned")
		}

		// Create .tmixer.yml files for every second project with a real directory
		overriddenProjects := make(map[string]bool)
		content := `windows:
  - name: overridden-terminal
    command: ["overridden-bash"]
switchCommands:
  - ["overridden-pwd"]`
		index := 0
		for _, project := range initialProjects {
			if project.Config == nil {
				continue
			}
			if _, err := os.Stat(project.Config.Directory); err != nil {
				continue
			}
			if index%2 == 0 {
				err := os.WriteFile(filepath.Join(project.Config.Directory, ".tmixer.yml"), []byte(content), 0o644)
				if err != nil {
					t.Fatal(err)
				}
				overriddenProjects[project.Name] = true
			}
			index++
		}

		projects, err := List(ctx, srv, tc.config)
		if err != nil {
			t.Error(err)
		}

		// Check that projects with .tmixer.yml files were overridden
		for _, project := range projects {
			if project.Config == nil {
				continue // Skip orphaned sessions
			}

			_, wasOverridden := overriddenProjects[project.Name]
			if wasOverridden {
				// Should have the overridden config
				if len(project.Config.Windows) != 1 || project.Config.Windows[0].Name != "overridden-terminal" {
					t.Errorf("Project %s should have been overridden but has windows: %v", project.Name, project.Config.Windows)
				}
				if len(project.Config.SwitchCommands) != 1 || project.Config.SwitchCommands[0][0] != "overridden-pwd" {
					t.Errorf("Project %s should have been overridden but has switch commands: %v", project.Name, project.Config.SwitchCommands)
				}
			}
			// Non-overridden projects can have any config (including empty), we just verify overridden ones
		}
	})
}

func TestProjectSpecificConfigsSkipsHomeDirectory(t *testing.T) {
	tmpHome := t.TempDir()
	validDir := t.TempDir()
	t.Setenv("HOME", tmpHome)

	config := &config.Config{
		Projects: []*config.ProjectConfig{
			{
				Name:      "home-project",
				Directory: tmpHome,
			},
			{
				Name:      "valid-project",
				Directory: validDir,
			},
		},
	}

	content := `windows:
  - name: overridden-terminal
    command: ["overridden-bash"]
switchCommands:
  - ["overridden-pwd"]`
	if err := os.WriteFile(filepath.Join(tmpHome, ".tmixer.yml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(validDir, ".tmixer.yml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		projects, err := List(ctx, srv, config)
		if err != nil {
			t.Error(err)
		}
		var homeProject *Project
		var validProject *Project
		for _, project := range projects {
			switch project.Name {
			case "home-project":
				homeProject = project
			case "valid-project":
				validProject = project
			}
		}
		if homeProject == nil {
			t.Fatal("home-project not listed")
		}
		if validProject == nil {
			t.Fatal("valid-project not listed")
		}
		if homeProject.Config == nil || validProject.Config == nil {
			t.Fatal("expected project configs to be present")
		}
		if len(homeProject.Config.Windows) == 1 {
			t.Error("home-project should not be overridden by home .tmixer.yml")
		}
		if len(homeProject.Config.SwitchCommands) == 1 {
			t.Error("home-project should not be overridden by home .tmixer.yml")
		}
		if len(validProject.Config.Windows) != 1 || validProject.Config.Windows[0].Name != "overridden-terminal" {
			t.Errorf("valid-project should have overridden windows, got %v", validProject.Config.Windows)
		}
		if len(validProject.Config.SwitchCommands) != 1 || validProject.Config.SwitchCommands[0][0] != "overridden-pwd" {
			t.Errorf("valid-project should have overridden switch commands, got %v", validProject.Config.SwitchCommands)
		}
	})
}

func BenchmarkListControlMode(b *testing.B) {
	dir := b.TempDir()
	n := 100
	for i := range n {
		os.Mkdir(filepath.Join(dir, strconv.Itoa(i)), 0o700)
	}
	config := &config.Config{
		Projects: []*config.ProjectConfig{
			{
				Name:      "bin",
				Directory: "/home/test/bin",
			},
			{
				Name:           "projects",
				Directory:      dir,
				SubDirectories: true,
			},
		},
	}
	ctx, tmux := testutil.SetupTestServer(b)
	defer testutil.TeardownTestServer(tmux)
	tmux.StartControlMode()
	defer tmux.StopControlMode()
	for i := range 10 {
		tmux.New("test-" + strconv.Itoa(i))
	}
	for b.Loop() {
		projects, err := List(ctx, tmux, config)
		if err != nil {
			b.Fatal(err)
		}
		if len(projects) < 110 {
			b.Fatal("Missing projects")
		}
	}
}
