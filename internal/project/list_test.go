package project

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

func TestListAddsFields(t *testing.T) {
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
		},
	}
	f := func(tmux *tmux.Server) {
		tmux.New("test-session")
		projects, err := List(tmux, config)
		if err != nil {
			t.Fatal(err)
		}
		matched_session := false
		matched_bin := false
		for _, p := range projects {
			if p.server != tmux {
				t.Fatal("Should always attach server")
			}
			switch p.Name {
			case "test-session":
				if p.Config != nil {
					t.Fatal("Should hgave nil config")
				}
				matched_session = true
			case "bin":
				if p.Config == nil {
					t.Fatal("Should have config")
				}
				if p.Config.Directory != "/home/test/bin" {
					t.Fatal("Should have correct directory")
				}
				matched_bin = true
			}
		}
		if !matched_bin || !matched_session {
			t.Fatal("Missing projects")
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}

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
	f := func(tmux *tmux.Server) {
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
				StartupWindows: []config.WindowConfig{{}, {}},
				SwitchCommands: []string{"", ""},
			},
		},
	}
	f := func(tmux *tmux.Server) {
		projects, err := List(tmux, config)
		if err != nil {
			t.Fatal(err)
		}
		projectNames := map[string]*Project{}
		for _, p := range projects {
			if p.server != tmux {
				t.Fatal("Should always attach server")
			}
			projectNames[p.Name] = p
		}
		if _, ok := projectNames["bin"]; !ok {
			t.Fatal("bin should be there")
		}
		for i := range n {
			if p, ok := projectNames["projects--"+strconv.Itoa(i)]; !ok {
				t.Fatalf("missing ub project %d", i)
			} else {
				// For all others change that the values are correct in the config
				if p.Config.SubDirectories {
					t.Fatal("Should set sub dirs of sub dir to false")
				}
				if p.Config.Directory != filepath.Join(dir, strconv.Itoa(i)) {
					t.Fatal("Directory should be the project dir itself")
				}
				if len(p.Config.StartupWindows) != 2 {
					t.Fatal("Should share startup windows")
				}
				if len(p.Config.SwitchCommands) != 2 {
					t.Fatal("Should share switch commands")
				}
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
	f := func(tmux *tmux.Server) {
		for i := range n {
			tmux.New("test-" + strconv.Itoa(i))
		}
		projects, err := List(tmux, config)
		if err != nil {
			t.Fatal(err)
		}
		projectNames := map[string]bool{}
		for _, p := range projects {
			if p.server != tmux {
				t.Fatal("Should always attach server")
			}
			if projectNames[p.Name] {
				t.Fatalf("duplicate project name %s", p.Name)
			}
			projectNames[p.Name] = true
		}
		if !projectNames["bin"] {
			t.Fatal("bin should be there")
		}
		if projectNames[tmux.CONTROL_SESSION_NAME] {
			t.Fatal("Should not list the control session")
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
	f := func(tmux *tmux.Server) {
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
	f := func(tmux *tmux.Server) {
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
	tmux := testutil.SetupTestServer(b)
	defer testutil.TeardownTestServer(tmux)
	for i := range 10 {
		tmux.New("test-" + strconv.Itoa(i))
	}
	for b.Loop() {
		projects, err := List(tmux, config)
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
	tmux := testutil.SetupTestServer(b)
	defer testutil.TeardownTestServer(tmux)
	tmux.StartControlMode()
	defer tmux.StopControlMode()
	for i := range 10 {
		tmux.New("test-" + strconv.Itoa(i))
	}
	for b.Loop() {
		projects, err := List(tmux, config)
		if err != nil {
			b.Fatal(err)
		}
		if len(projects) < 110 {
			b.Fatal("Missing projects")
		}
	}
}
