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
			testutil.RunWithAndWithoutControlModeTestRun(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
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
					Projects: map[string]*config.ProjectConfig{
						"bin": {
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
					Projects: map[string]*config.ProjectConfig{
						"projects": {
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
					Projects: map[string]*config.ProjectConfig{
						"bin": {
							Directory: "/home/test/bin",
						},
						"projects": {
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
		Projects: map[string]*config.ProjectConfig{
			"projects--5": {
				Directory: "/home/test/bin",
			},
			"projects": {
				Directory:      dir,
				SubDirectories: true,
			},
		},
	}
	testutil.RunWithAndWithoutControlModeTestRun(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
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
		Projects: map[string]*config.ProjectConfig{
			"projects--5": {
				Directory: "/home/test/bin",
			},
			"projects": {
				Directory:      dir,
				SubDirectories: true,
			},
		},
	}
	testutil.RunWithAndWithoutControlModeTestRun(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
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
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
			"projects": {
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

func BenchmarkListControlMode(b *testing.B) {
	dir := b.TempDir()
	n := 100
	for i := range n {
		os.Mkdir(filepath.Join(dir, strconv.Itoa(i)), 0o700)
	}
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
			"projects": {
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
