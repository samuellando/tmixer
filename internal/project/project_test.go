package project

import (
	"os"
	"strings"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

func TestProjectStatus(t *testing.T) {
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		// Initially inactive
		status, err := project.Status()
		if err != nil {
			t.Fatal(err)
		}
		if status != PROJECT_STATUS_INACTIVE {
			t.Fatal("Project should have inactive status")
		}
		// After a start, should be active
		s, err := project.Start()
		if err != nil {
			t.Fatal(err)
		}
		status, err = project.Status()
		if err != nil {
			t.Fatal(err)
		}
		if status != PROJECT_STATUS_ACTIVE {
			t.Fatal("Project should have active status")
		}
		// Setup a cleint
		f := testutil.SetupTestClient(srv, s)
		defer f.Close()
		status, err = project.Status()
		if err != nil {
			t.Fatal(err)
		}
		if status != PROJECT_STATUS_ATTACHED {
			t.Fatal("Project should have attached status")
		}
		// After we kill should be inactive again
		err = project.Kill()
		if err != nil {
			t.Fatal(err)
		}
		status, err = project.Status()
		if err != nil {
			t.Fatal(err)
		}
		if status != PROJECT_STATUS_INACTIVE {
			t.Fatal("Project should have inactive status after kill")
		}
	})
}

func TestProjectSession(t *testing.T) {
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		s, err := project.Start()
		if err != nil {
			t.Fatal(err)
		}
		name, err := s.Name()
		if err != nil {
			t.Fatal(err)
		}
		if name != "bin" {
			t.Fatal("Should get a session")
		}
	})
}

func TestProjectLastActivity(t *testing.T) {
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
			"bin2": {
				Directory: "/home/test/bin",
			},
			"bin3": {
				Directory: "/home/test/bin",
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var bin *Project
		var bin2 *Project
		var bin3 *Project
		for _, p := range projects {
			if p.Name == "bin" {
				bin = p
			}
			if p.Name == "bin2" {
				bin2 = p
			}
			if p.Name == "bin3" {
				bin3 = p
			}
		}
		bin.Start()
		bin2.Start()
		a1, err := bin.LastActivity()
		if err != nil {
			t.Fatal(err)
		}
		a2, err := bin2.LastActivity()
		if err != nil {
			t.Fatal(err)
		}
		if a1.After(*a2) {
			t.Fatal("Should be before")
		}
		_, err = bin3.LastActivity()
		if err != ErrSessionNotFound {
			t.Fatal(err)
		}
	})
}

func TestProjectSessionNotFound(t *testing.T) {
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		s, err := project.Session()
		if err != ErrSessionNotFound {
			t.Fatal("Should get a session not found error")
		}
		if s != nil {
			t.Fatal("Should get a nil session")
		}
	})
}

func TestProjectStart(t *testing.T) {
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		s, err := project.Start()
		if err != nil {
			t.Fatal(err)
		}
		name, err := s.Name()
		if err != nil {
			t.Fatal(err)
		}
		if name != "bin" {
			t.Fatal("Should get a session")
		}
		status, err := project.Status()
		if err != nil {
			t.Fatal(err)
		}
		if status != PROJECT_STATUS_ACTIVE {
			t.Fatal("Project should be active")
		}
	})
}

func TestProjectAlreadyStarted(t *testing.T) {
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		s1, err := project.Start()
		if err != nil {
			t.Fatal(err)
		}
		s2, err := project.Start()
		if err != nil {
			t.Fatal(err)
		}
		if s1.Id != s2.Id {
			t.Fatal("Should return the already started project")
		}
	})
}

func TestProjectStartWindows(t *testing.T) {
	c1 := "expr 5 \\* 5"
	c2 := "expr 7 \\* 7"
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
				Windows: []config.WindowConfig{
					{
						Name:    "test1",
						Command: &c1,
					},
					{
						Name:    "test2",
						Command: &c2,
					},
				},
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		s, err := project.Start()
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Second)
		windows, _ := s.Windows()
		if len(windows) != 2 {
			t.Fatal("Should have 2 windows")
		}
		if name, _ := windows[0].Name(); name != "test1" {
			t.Fatal("Should have test1 as window name")
		}
		if name, _ := windows[1].Name(); name != "test2" {
			t.Fatal("Should have test2 as window name")
		}
		panes1, _ := windows[0].Panes()
		if len(panes1) != 1 {
			t.Fatal("should have 1 pane")
		}
		panes2, _ := windows[1].Panes()
		if len(panes2) != 1 {
			t.Fatal("should have 1 pane")
		}
		out1, _ := panes1[0].Capture()
		if !strings.Contains(strings.Join(out1, ""), "25") {
			t.Fatal("Command1 did not run!")
		}
		out2, _ := panes2[0].Capture()
		if !strings.Contains(strings.Join(out2, ""), "49") {
			t.Fatal("Command2 did not run!")
		}
	})
}

func TestProjectStartPanes(t *testing.T) {
	c1 := "expr 5 \\* 5"
	c2 := "expr 7 \\* 7"
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
				Windows: []config.WindowConfig{
					{
						Name: "test1",
						Panes: []config.PaneConfig{
							{Command: &c1},
							{Command: &c2},
						},
					},
				},
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		s, err := project.Start()
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Second)
		windows, _ := s.Windows()
		if len(windows) != 1 {
			t.Fatal("Should have 1 window")
		}
		if name, _ := windows[0].Name(); name != "test1" {
			t.Fatal("Should have test1 as window name")
		}
		panes1, _ := windows[0].Panes()
		if len(panes1) != 2 {
			t.Fatal("should have 2 pane2")
		}
		out1, _ := panes1[0].Capture()
		if !strings.Contains(strings.Join(out1, ""), "25") {
			t.Fatal("Command1 did not run!")
		}
		out2, _ := panes1[1].Capture()
		if !strings.Contains(strings.Join(out2, ""), "49") {
			t.Fatal("Command2 did not run!")
		}
	})
}

func TestProjectStartPanesWithWindowCommand(t *testing.T) {
	c1 := "expr 5 \\* 5"
	c2 := "expr 7 \\* 7"
	cw := "expr 9 \\* 9"
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
				Windows: []config.WindowConfig{
					{
						Name:    "test1",
						Command: &cw,
						Panes: []config.PaneConfig{
							{Command: &c1},
							{Command: &c2},
						},
					},
				},
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		s, err := project.Start()
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Second)
		windows, _ := s.Windows()
		if len(windows) != 1 {
			t.Fatal("Should have 1 window")
		}
		if name, _ := windows[0].Name(); name != "test1" {
			t.Fatal("Should have test1 as window name")
		}
		panes1, _ := windows[0].Panes()
		if len(panes1) != 3 {
			t.Fatal("should have 3 pane2")
		}
		out1, _ := panes1[0].Capture()
		if !strings.Contains(strings.Join(out1, ""), "81") {
			t.Fatal("Command1 did not run!")
		}
		out2, _ := panes1[1].Capture()
		if !strings.Contains(strings.Join(out2, ""), "25") {
			t.Fatal("Command2 did not run!")
		}
		out3, _ := panes1[2].Capture()
		if !strings.Contains(strings.Join(out3, ""), "49") {
			t.Fatal("Command2 did not run!")
		}
	})
}

func TestProjectKill(t *testing.T) {
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		_, err = project.Start()
		status, err := project.Status()
		if err != nil {
			t.Fatal(err)
		}
		if status != PROJECT_STATUS_ACTIVE {
			t.Fatal("Project should be active")
		}
		err = project.Kill()
		status, err = project.Status()
		if err != nil {
			t.Fatal(err)
		}
		if status != PROJECT_STATUS_INACTIVE {
			t.Fatal("Project should be inactive")
		}
	})
}

func TestProjectSwitch(t *testing.T) {
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		s, _ := srv.New("ses-1")
		f := testutil.SetupTestClient(srv, s)
		defer f.Close()
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		err = project.Switch()
		if err != nil {
			t.Fatal(err)
		}
		status, _ := project.Status()
		if status != PROJECT_STATUS_ATTACHED {
			t.Fatal("Project should be attached")
		}
	})
}

func TestProjectSwitchCommands(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "tmixer-test-projects")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory:      dir,
				SwitchCommands: []string{"touch file1", "touch file2"},
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		s, _ := srv.New("ses-1")
		f := testutil.SetupTestClient(srv, s)
		defer f.Close()
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		err = project.Switch()
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Second)
		entries, _ := os.ReadDir(dir)
		if len(entries) != 2 {
			t.Fatal("Should have created 2 files")
		}
	})
}

func TestProjectRunSwitchCommands(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "tmixer-test-projects")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory:      dir,
				SwitchCommands: []string{"touch file1", "touch file2"},
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		s, _ := srv.New("ses-1")
		f := testutil.SetupTestClient(srv, s)
		defer f.Close()
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		project.Start()
		err = project.RunSwitchCommands()
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Second)
		entries, _ := os.ReadDir(dir)
		if len(entries) != 2 {
			t.Fatal("Should have created 2 files")
		}
	})
}

func TestProjectReset(t *testing.T) {
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		s1, _ := project.Start()
		s2, err := project.Reset()
		if err != nil {
			t.Fatal(err)
		}
		if s1.Id == s2.Id {
			t.Fatal("Should be new id")
		}
		res, _ := project.Session()
		if res.Id != s2.Id {
			t.Fatal("Should match new session id")
		}
	})
}

func TestProjectResetWindows(t *testing.T) {
	c1 := "expr 5 \\* 5"
	c2 := "expr 7 \\* 7"
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
				Windows: []config.WindowConfig{
					{
						Name:    "test1",
						Command: &c1,
					},
					{
						Name:    "test2",
						Command: &c2,
					},
				},
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		project.Start()
		_, err = project.Reset()
		if err != nil {
			t.Fatal(err)
		}
		res, _ := project.Session()
		time.Sleep(time.Second)
		windows, _ := res.Windows()
		if len(windows) != 2 {
			t.Fatal("Should have 2 windows")
		}
		if name, _ := windows[0].Name(); name != "test1" {
			t.Fatal("Should have test1 as window name")
		}
		if name, _ := windows[1].Name(); name != "test2" {
			t.Fatal("Should have test2 as window name")
		}
		panes1, _ := windows[0].Panes()
		if len(panes1) != 1 {
			t.Fatal("should have 1 pane")
		}
		panes2, _ := windows[1].Panes()
		if len(panes2) != 1 {
			t.Fatal("should have 1 pane")
		}
		out1, _ := panes1[0].Capture()
		if !strings.Contains(strings.Join(out1, ""), "25") {
			t.Fatal("Command1 did not run!")
		}
		out2, _ := panes2[0].Capture()
		if !strings.Contains(strings.Join(out2, ""), "49") {
			t.Fatal("Command2 did not run!")
		}
	})
}

func TestProjectResetAttached(t *testing.T) {
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		s1, _ := project.Start()
		f := testutil.SetupTestClient(srv, s1)
		defer f.Close()
		s2, err := project.Reset()
		if err != nil {
			t.Fatal(err)
		}
		if s1.Id == s2.Id {
			t.Fatal("Should be new id")
		}
		res, _ := project.Session()
		if res.Id != s2.Id {
			t.Fatal("Should match new session id")
		}
		if status, _ := project.Status(); status != PROJECT_STATUS_ATTACHED {
			t.Fatal("Should attach to new session")
		}
	})
}

func TestProjectKillAttachedLastActive(t *testing.T) {
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
			"dog": {
				Directory: "/home/test/bin",
			},
			"cat": {
				Directory: "/home/test/bin",
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		var s1 *tmux.Session
		var s2 *tmux.Session
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
				s1, _ = p.Start()
			}
			if p.Name == "cat" {
				s2, _ = p.Start()
			}
		}
		f := testutil.SetupTestClient(srv, s1)
		defer f.Close()
		project.Kill()
		client, _ := srv.ActiveClient()
		cs, _ := client.Session()
		if cs.Id != s2.Id {
			t.Fatal("Should switch to the last active session")
		}
	})
}

func TestProjectKillAttachedDefault(t *testing.T) {
	def := "cat"
	config := &config.Config{
		DefaultProject: &def,
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory: "/home/test/bin",
			},
			"dog": {
				Directory: "/home/test/bin",
			},
			"cat": {
				Directory: "/home/test/bin",
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		var s1 *tmux.Session
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
				s1, _ = p.Start()
			}
		}
		// Kill the default test session
		sessions, _ := srv.ListSessions()
		for _, s := range sessions {
			if s.Id != s1.Id {
				name, _ := s.Name()
				if name != tmux.CONTROL_SESSION_NAME {
					s.Kill()
				}
			}

		}
		f := testutil.SetupTestClient(srv, s1)
		defer f.Close()
		project.Kill()
		client, err := srv.ActiveClient()
		if err != nil {
			t.Fatal(err)
		}
		cs, err := client.Session()
		if err != nil {
			t.Fatal(err)
		}
		name, _ := cs.Name()
		if name != def {
			t.Fatalf("Should switch to the default project, got %s", name)
		}
	})
}

func TestProjectResetAttachedSwitchCommands(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "tmixer-test-projects")
	if err != nil {
		t.Fatal(err)
	}
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"bin": {
				Directory:      dir,
				SwitchCommands: []string{"touch test"},
			},
		},
	}
	testutil.RunWithAndWithoutControlMode(t, func(srv *tmux.Server) {
		projects, err := List(srv, config)
		if err != nil {
			t.Fatal(err)
		}
		var project *Project
		for _, p := range projects {
			if p.Name == "bin" {
				project = p
			}
		}
		if project == nil {
			t.Fatal("bin project not listed")
		}
		s1, _ := project.Start()
		f := testutil.SetupTestClient(srv, s1)
		defer f.Close()
		s2, err := project.Reset()
		time.Sleep(time.Second)
		if err != nil {
			t.Fatal(err)
		}
		if s1.Id == s2.Id {
			t.Fatal("Should be new id")
		}
		res, _ := project.Session()
		if res.Id != s2.Id {
			t.Fatal("Should match new session id")
		}
		if status, _ := project.Status(); status != PROJECT_STATUS_ATTACHED {
			t.Fatal("Should attach to new session")
		}
		entries, _ := os.ReadDir(dir)
		if len(entries) != 1 {
			t.Fatal("Should have created 1 files")
		}

	})
}
