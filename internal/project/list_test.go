package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
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

func setupListTest(t *testing.T) (string, []listTestCase) {
	dir, err := os.MkdirTemp(os.TempDir(), "tmixer-test-projects")
	if err != nil {
		t.Fatal(err)
	}
	n := 10
	for i := range n {
		os.Mkdir(filepath.Join(dir, strconv.Itoa(i)), 0o700)
		f, err := os.OpenFile(filepath.Join(dir, "file"+strconv.Itoa(i)), os.O_RDONLY|os.O_CREATE, 0o644)
		if err != nil {
			t.Error(err)
		}
		f.Close()
	}

	var testCases = []listTestCase{
		{
			config:   &config.Config{},
			projects: []string{testutil.DEFAULT_TEST_SESSION},
		},
		{
			config: &config.Config{
				Projects: map[string]*config.ProjectConfig{
					"bin": {
						Directory: "/home/test/bin",
					},
				},
			},
			projects: []string{testutil.DEFAULT_TEST_SESSION, "bin"},
		},
		{
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
		},
		{
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
		},
		{
			config: &config.Config{
				Projects: map[string]*config.ProjectConfig{
					"hello...world": {
						Directory: "/home/test/bin",
					},
				},
			},
			projects: []string{
				testutil.DEFAULT_TEST_SESSION,
				"hello...world",
			},
		},
	}
	return dir, testCases
}

func teardownListTest(t *testing.T, dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestListsAll(t *testing.T) {
	dir, testCases := setupListTest(t)
	defer teardownListTest(t, dir)
	for _, tc := range testCases {
		testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
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
}

func TestSetsFields(t *testing.T) {
	dir, testCases := setupListTest(t)
	defer teardownListTest(t, dir)
	for _, tc := range testCases {
		testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
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
}

func TestListIncludesAllSessions(t *testing.T) {
	n := 10
	dir, testCases := setupListTest(t)
	defer teardownListTest(t, dir)
	for _, tc := range testCases {
		testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
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
}

func TestListMatchesExistingSessions(t *testing.T) {
	dir, testCases := setupListTest(t)
	defer teardownListTest(t, dir)
	for _, tc := range testCases {
		testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
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
}

func TestAmbiguousNames(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "tmixer-test-projects")
	if err != nil {
		t.Fatal(err)
	}
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
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		_, err := List(ctx, srv, config)
		if !errors.Is(err, ErrAmbiguousName) {
			t.Errorf("wrong error returned %v", err)
		}
	})
}

func TestListNoTmux(t *testing.T) {
	ctx, _ := log.New(context.Background(), nil)
	dir, testCases := setupListTest(t)
	defer teardownListTest(t, dir)
	for _, tc := range testCases {
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
	}
}

func BenchmarkList(b *testing.B) {
	dir, err := os.MkdirTemp(os.TempDir(), "tmixer-test-projects")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)
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
	dir, err := os.MkdirTemp(os.TempDir(), "tmixer-test-projects")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)
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
