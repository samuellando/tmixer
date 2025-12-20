package projectv2

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"samuellando.com/tmixer/internal/configv2"
	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmuxv2"
)

func TestListIncludesAllProjects(t *testing.T) {
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
			"projects": {
				Directory: "/home/test/Projects",
			},
		},
	}
	f := func(tmux *tmuxv2.Server) {
		projects, err := List(tmux, config)
		if err != nil {
			t.Fatal(err)
		}
		projectNames := map[string]bool{}
		for _, p := range projects {
			projectNames[p.Name] = true
		}
		for cfgName := range config.Projects {
			if !projectNames[cfgName] {
				t.Errorf("Project %q missing from list", cfgName)
			}
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestListIncludesAllSubDirProjects(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "tmixer-test-projects")
	defer os.RemoveAll(dir)
	n := 100
	if err != nil {
		t.Fatal(err)
	}
	for i := range 100 {
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
	f := func(tmux *tmuxv2.Server) {
		projects, err := List(tmux, config)
		if err != nil {
			t.Fatal(err)
		}
		projectNames := map[string]bool{}
		for _, p := range projects {
			projectNames[p.Name] = true
		}
		if !projectNames["bin"] {
			t.Fatal("bin should be there")
		}
		for i := range n {
			if !projectNames["projects--"+strconv.Itoa(i)] {
				t.Fatalf("missing ub project %d", i)
			}
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestListIncludesAllSessions(t *testing.T) {
	n := 10
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
		},
	}
	f := func(tmux *tmuxv2.Server) {
		for i := range n {
			tmux.New("test-" + strconv.Itoa(i))
		}
		projects, err := List(tmux, config)
		if err != nil {
			t.Fatal(err)
		}
		projectNames := map[string]bool{}
		for _, p := range projects {
			if projectNames[p.Name] {
				t.Fatalf("duplicate project name %s", p.Name)
			}
			projectNames[p.Name] = true
		}
		if !projectNames["bin"] {
			t.Fatal("bin should be there")
		}
		for i := range n {
			if !projectNames["test-"+strconv.Itoa(i)] {
				t.Fatalf("missing session project %d", i)
			}
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestListMatchesExistingSessions(t *testing.T) {
	n := 10
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
			"test-3": {
				Directory: "/home/test/bin",
			},
		},
	}
	f := func(tmux *tmuxv2.Server) {
		for i := range n {
			tmux.New("test-" + strconv.Itoa(i))
		}
		projects, err := List(tmux, config)
		if err != nil {
			t.Fatal(err)
		}
		projectNames := map[string]bool{}
		for _, p := range projects {
			if projectNames[p.Name] {
				t.Fatalf("duplicate project name %s", p.Name)
			}
			projectNames[p.Name] = true
		}
		if !projectNames["bin"] {
			t.Fatal("bin should be there")
		}
		for i := range n {
			if !projectNames["test-"+strconv.Itoa(i)] {
				t.Fatalf("missing session project %d", i)
			}
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

func TestListNoTmux(t *testing.T) {
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
		},
	}
	projects, err := List(nil, config)
	if err != nil {
		t.Fatal(err)
	}
	projectNames := map[string]bool{}
	for _, p := range projects {
		if projectNames[p.Name] {
			t.Fatalf("duplicate project name %s", p.Name)
		}
		projectNames[p.Name] = true
	}
	if !projectNames["bin"] {
		t.Fatal("bin should be there")
	}
}

func TestListNoProjects(t *testing.T) {
	config := &config.Config{
		Projects: nil,
	}
	f := func(tmux *tmuxv2.Server) {
		projects, err := List(tmux, config)
		if err != nil {
			t.Fatal(err)
		}
		if len(projects) == 0 {
			t.Fatal("should still return session projects")
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)

}
