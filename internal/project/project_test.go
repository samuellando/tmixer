package project

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

func stringPointer(s string) *string {
	return &s
}

var pane_counts = []int{1, 4, 3}
var primes = []int{471193, 842887, 5197, 316219}

func primeCommand(i, j int) *string {
	return stringPointer(fmt.Sprintf("expr %d \\* %d", primes[i], primes[j]))
}

var testWindowConfigs = []config.WindowConfig{
	{
		Name:    "one",
		Command: primeCommand(0, 0),
	},
	{
		Name:    "two",
		Command: primeCommand(1, 0),
		Panes: []config.PaneConfig{
			{
				Command: primeCommand(1, 1),
			},
			{
				Command: primeCommand(1, 2),
				Split:   stringPointer("Horizontal"),
			},
			{
				Command: primeCommand(1, 3),
				Split:   stringPointer("Vertical"),
			},
		},
	},
	{
		Name: "three",
		Panes: []config.PaneConfig{
			{
				Command: primeCommand(2, 0),
			},
			{
				Command: primeCommand(2, 1),
				Split:   stringPointer("Horizontal"),
			},
			{
				Command: primeCommand(2, 2),
				Split:   stringPointer("Vertical"),
			},
		},
	},
}

var switchCommands = []string{"touch one-$(date +%s%N)", "touch two-$(date +%s%N)"}

var testConfig = &config.Config{
	Projects: map[string]*config.ProjectConfig{
		"inactive": {},
		"inactive-windows": {
			Windows: testWindowConfigs,
		},
		"inactive-switch": {
			SwitchCommands: switchCommands,
		},
		"inactive-windows-switch": {
			Windows:        testWindowConfigs,
			SwitchCommands: switchCommands,
		},
		"active": {},
		"active-windows": {
			Windows: testWindowConfigs,
		},
		"active-switch": {
			SwitchCommands: switchCommands,
		},
		"active-windows-switch": {
			Windows:        testWindowConfigs,
			SwitchCommands: switchCommands,
		},
		"attached-switch": {
			SwitchCommands: switchCommands,
		},
	},
}

type projectTestCase struct {
	project *Project
	session *tmux.Session
}

func (tc *projectTestCase) initialStatus() ProjectStatus {
	if strings.HasPrefix(tc.project.Name, "inactive") {
		return PROJECT_STATUS_INACTIVE
	}
	if strings.HasPrefix(tc.project.Name, "active") {
		return PROJECT_STATUS_ACTIVE
	}
	if strings.HasPrefix(tc.project.Name, "attached") {
		return PROJECT_STATUS_ATTACHED
	}
	panic("Invalid initial status")
}

func setupTestProjects(t *testing.T, ctx context.Context, srv *tmux.Server) (string, *os.File, []projectTestCase) {
	dir, err := os.MkdirTemp(os.TempDir(), "tmixer-test-projects")
	if err != nil {
		t.Fatal(err)
	}
	testCases := make([]projectTestCase, 0)
	var client *os.File
	for name, pc := range testConfig.Projects {
		tc := projectTestCase{}
		pc.Directory = filepath.Join(dir, name)
		err := os.Mkdir(pc.Directory, 0o700)
		if err != nil {
			t.Error(err)
		}
		p := Project{
			Name:       name,
			Config:     pc,
			server:     srv,
			fullConfig: testConfig,
		}
		tc.project = &p
		if strings.HasPrefix(name, "active") || strings.HasPrefix(name, "attached") {
			s, err := p.Start(ctx)
			if err != nil {
				t.Error(err)
			}
			tc.session = s
		}
		if strings.HasPrefix(name, "attached") {
			client = testutil.SetupTestClient(srv, tc.session)
		}
		testCases = append(testCases, tc)
	}
	return dir, client, testCases
}

func teardownTestProjects(t *testing.T, dir string, client *os.File) {
	err := os.RemoveAll(dir)
	if err != nil {
		t.Error(err)
	}
	err = client.Close()
	if err != nil {
		t.Error(err)
	}
}

func TestProjectStatus(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		for _, tc := range testCases {
			status, err := tc.project.Status()
			if err != nil {
				t.Error(err)
			}
			if tc.initialStatus() != status {
				t.Errorf("Status does not match %d != %d", tc.initialStatus(), status)
			}
		}
	})
}

func TestProjectSession(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		for _, tc := range testCases {
			res, err := tc.project.Session()
			switch tc.initialStatus() {
			case PROJECT_STATUS_INACTIVE:
				if !errors.Is(err, ErrSessionNotFound) {
					t.Error("Inactive project should give session not found")
				}
			default:
				if err != nil {
					t.Error(err)
				}
				if res.Id != tc.session.Id {
					t.Errorf("Project session did not match %s != %s", res.Id, tc.session.Id)
				}
			}
		}
	})
}

func FuzzTmuxSessionName(f *testing.F) {
	testcases := []string{"Hello World", "Hello.World", "!12345"}
	for _, tc := range testcases {
		f.Add(tc)
	}
	_, srv := testutil.SetupTestServer(f)
	defer testutil.TeardownTestServer(srv)
	f.Fuzz(func(t *testing.T, a string) {
		if a == "" || strings.Contains(a, "\x00") {
			t.Skip()
		}
		p := Project{
			Name: a,
			Config: &config.ProjectConfig{
				Directory: "/tmp",
			},
			server: srv,
		}
		session, err := srv.New(a)
		if err != nil {
			t.Fatal(err)
		}
		sessionName, err := session.Name()
		if err != nil {
			t.Fatal(err)
		}
		if sessionName != p.TmuxSessionName() {
			t.Errorf("%s != %s", sessionName, p.TmuxSessionName())
		}
		err = session.Kill()
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestProjectLastActivity(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		for _, tc := range testCases {
			res, err := tc.project.LastActivity()
			switch tc.initialStatus() {
			case PROJECT_STATUS_INACTIVE:
				if !errors.Is(err, ErrSessionNotFound) {
					t.Error("Inactive project should give session not found")
				}
			default:
				if err != nil {
					t.Error(err)
				}
				if time.Since(*res) > 30*time.Second {
					t.Errorf("Project last activity time should be recent %s", time.Since(*res))
				}
			}
		}
	})
}

func TestProjectStart(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		for _, tc := range testCases {
			res, err := tc.project.Start(ctx)
			if err != nil {
				t.Error(err)
			}
			if res == nil {
				t.Error("Should not return a nil session")
			}
			if tc.session != nil && tc.session.Id != res.Id {
				t.Error("Session Id should match")
			}
			status, err := tc.project.Status()
			if err != nil {
				t.Error(err)
			}
			switch tc.initialStatus() {
			case PROJECT_STATUS_INACTIVE:
				if status != PROJECT_STATUS_ACTIVE {
					t.Error("Inactive project should be active now")
				}
			case PROJECT_STATUS_ACTIVE:
				if status != PROJECT_STATUS_ACTIVE {
					t.Error("Active project should still be active")
				}
			case PROJECT_STATUS_ATTACHED:
				if status != PROJECT_STATUS_ATTACHED {
					t.Error("Attached project should still be attached")
				}
			}
		}
	})
}

func TestProjectStartWindowsAndPanes(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		for _, tc := range testCases {
			res, err := tc.project.Start(ctx)
			time.Sleep(time.Second)
			if err != nil {
				t.Error(err)
			}
			if strings.Contains(tc.project.Name, "windows") {
				windows, err := res.Windows()
				if err != nil {
					t.Error(err)
				}
				if len(windows) != 3 {
					t.Error("Should have 3 windows")
				}
				for i := range 3 {
					panes, err := windows[i].Panes()
					if err != nil {
						t.Error(err)
					}
					for j := range pane_counts[i] {
						expected := fmt.Sprintf("%d", primes[i]*primes[j])
						out, err := panes[j].Capture()
						if err != nil {
							t.Error(err)
						}
						allOut := strings.Join(out, "")
						if !strings.Contains(allOut, expected) {
							t.Errorf("Missing output %s in %s", expected, allOut)
						}
					}
				}
			} else {
				windows, err := res.Windows()
				if err != nil {
					t.Error(err)
				}
				if len(windows) != 1 {
					t.Errorf("Should have 1 (default) window got %d", len(windows))
				}
			}
		}
	})
}

func TestProjectKill(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		for _, tc := range testCases {
			f, res := tc.project.Kill(ctx)
			if f != nil {
				f()
			}
			status, err := tc.project.Status()
			if err != nil {
				t.Error(err)
			}
			if status != PROJECT_STATUS_INACTIVE {
				t.Error("Project should be inactive")
			}
			switch tc.initialStatus() {
			case PROJECT_STATUS_INACTIVE:
				if !errors.Is(res, ErrSessionNotFound) {
					t.Error("Inactive project give session not found errror")
				}
			default:
				if res != nil {
					t.Error(res)
				}
			}
		}
	})
}

func TestProjectKillAttachedLastActive(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		time.Sleep(time.Second)
		var attached *Project
		var lastActive *Project
		// Randomize the order of the projects
		rand.Shuffle(len(testCases), func(i, j int) {
			tmp := testCases[i]
			testCases[i] = testCases[j]
			testCases[j] = tmp
		})
		// Find the currently attached project
		for _, tc := range testCases {
			if tc.initialStatus() == PROJECT_STATUS_ATTACHED {
				attached = tc.project
			}
		}
		// Attach and switch back for all active projects to force a lastActivity update
		for _, tc := range testCases {
			if tc.initialStatus() == PROJECT_STATUS_ACTIVE {
				err := tc.project.Switch(ctx)
				if err != nil {
					t.Error(err)
				}
				err = attached.Switch(ctx)
				if err != nil {
					t.Error(err)
				}
				lastActive = tc.project
				time.Sleep(time.Second)
			}
		}
		f, err := attached.Kill(ctx)
		if f != nil {
			f()
		}
		if err != nil {
			t.Error(err)
		}
		status, err := lastActive.Status()
		if err != nil {
			t.Error(err)
		}
		if status != PROJECT_STATUS_ATTACHED {
			t.Error("The last active project should be attached now")
		}
	})
}

func TestProjectKillAttachedDefault(t *testing.T) {
	def := "inactive-windows-switch"
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		var attached *Project
		var defProj *Project
		for _, tc := range testCases {
			if tc.initialStatus() == PROJECT_STATUS_ATTACHED {
				attached = tc.project
			}
			if tc.initialStatus() == PROJECT_STATUS_ACTIVE {
				f, err := tc.project.Kill(ctx)
				if f != nil {
					f()
				}
				if err != nil {
					t.Error(err)
				}
			}
			if tc.project.Name == def {
				defProj = tc.project
			}
		}
		// Kill the default test session
		ts, err := srv.GetSessionWithName(testutil.DEFAULT_TEST_SESSION)
		if err != nil {
			t.Error(err)
		}
		err = ts.Kill()
		if err != nil {
			t.Error(err)
		}
		// Set the default project
		attached.fullConfig.DefaultProject = &def
		f, err := attached.Kill(ctx)
		if f != nil {
			f()
		}
		if err != nil {
			t.Error(err)
		}
		status, err := defProj.Status()
		if err != nil {
			t.Error(err)
		}
		if status != PROJECT_STATUS_ATTACHED {
			t.Error("The default project should be attached now")
		}
	})
}

func TestProjectKillAttachedNoDefault(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		var attached *Project
		for _, tc := range testCases {
			if tc.initialStatus() == PROJECT_STATUS_ATTACHED {
				attached = tc.project
			}
			if tc.initialStatus() == PROJECT_STATUS_ACTIVE {
				_, err := tc.project.Kill(ctx)
				if err != nil {
					t.Error(err)
				}
			}
		}
		// Kill the default test session
		ts, err := srv.GetSessionWithName(testutil.DEFAULT_TEST_SESSION)
		if err != nil {
			t.Error(err)
		}
		err = ts.Kill()
		if err != nil {
			t.Error(err)
		}
		// Set the default project
		attached.fullConfig.DefaultProject = nil
		f, err := attached.Kill(ctx)
		if f != nil {
			f()
		}
		if err != nil {
			t.Error(err)
		}
		someAttached := false
		for _, tc := range testCases {
			status, err := tc.project.Status()
			if err != nil {
				t.Error(err)
			}
			if status == PROJECT_STATUS_ATTACHED {
				someAttached = true
			}
		}
		if !someAttached {
			t.Error("Some project should be attached now")
		}
	})
}

func TestProjectKillAttachedNoProjects(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		var attached *Project
		for _, tc := range testCases {
			if tc.initialStatus() == PROJECT_STATUS_ATTACHED {
				attached = tc.project
			}
			if tc.initialStatus() == PROJECT_STATUS_ACTIVE {
				_, err := tc.project.Kill(ctx)
				if err != nil {
					t.Error(err)
				}
			}
		}
		// Kill the default test session
		ts, err := srv.GetSessionWithName(testutil.DEFAULT_TEST_SESSION)
		if err != nil {
			t.Error(err)
		}
		err = ts.Kill()
		if err != nil {
			t.Error(err)
		}
		// Set the default project
		attached.fullConfig = &config.Config{
			Projects: map[string]*config.ProjectConfig{
				attached.Name: attached.Config,
			},
		}
		f, err := attached.Kill(ctx)
		if f != nil {
			f()
		}
		if err != nil {
			t.Error(err)
		}
		_, err = srv.ActiveClient()
		if err == nil {
			t.Error(err)
		}
	})
}

func TestProjectSwitch(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		for _, tc := range testCases {
			err := tc.project.Switch(ctx)
			if err != nil {
				t.Error(err)
			}
			status, err := tc.project.Status()
			if err != nil {
				t.Error(err)
			}
			if status != PROJECT_STATUS_ATTACHED {
				t.Error("Should attache to project")
			}
		}
	})
}

func TestProjectSwitchCreatesSession(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		for _, tc := range testCases {
			err := tc.project.Switch(ctx)
			if err != nil {
				t.Error(err)
			}
			session, err := tc.project.Session()
			if err != nil {
				t.Error(err)
			}
			if session == nil {
				t.Error("Should have a session")
			}
			if tc.initialStatus() != PROJECT_STATUS_INACTIVE {
				if tc.session.Id != session.Id {
					t.Error("Should attach to existing sesison")
				}
			}
		}
	})
}

func TestProjectSwitchWindowsAndPanes(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		for _, tc := range testCases {
			err := tc.project.Switch(ctx)
			time.Sleep(time.Second)
			if err != nil {
				t.Error(err)
			}
			res, err := tc.project.Session()
			if err != nil {
				t.Error(err)
			}
			if strings.Contains(tc.project.Name, "windows") {
				windows, err := res.Windows()
				if err != nil {
					t.Error(err)
				}
				if len(windows) != 3 {
					t.Error("Should have 3 windows")
				}
				for i := range 3 {
					panes, err := windows[i].Panes()
					if err != nil {
						t.Error(err)
					}
					for j := range pane_counts[i] {
						expected := fmt.Sprintf("%d", primes[i]*primes[j])
						out, err := panes[j].Capture()
						if err != nil {
							t.Error(err)
						}
						allOut := strings.Join(out, "")
						if !strings.Contains(allOut, expected) {
							t.Errorf("Missing output %s in %s", expected, allOut)
						}
					}
				}
			} else {
				windows, err := res.Windows()
				if err != nil {
					t.Error(err)
				}
				if len(windows) != 1 {
					t.Errorf("Should have 1 (default) window got %d", len(windows))
				}
			}
		}
	})
}

func TestProjectSwitchCommands(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		for _, tc := range testCases {
			err := tc.project.Switch(ctx)
			if err != nil {
				t.Error(err)
			}
			time.Sleep(time.Second)
			ls, err := os.ReadDir(tc.project.Config.Directory)
			if err != nil {
				t.Error(err)
			}
			if strings.Contains(tc.project.Name, "switch") {
				if len(ls) != len(switchCommands) {
					t.Errorf("Expected %d files from switch commands got %d", len(switchCommands), len(ls))
				}
			} else {
				if len(ls) != 0 {
					t.Errorf("Expected 0 files from switch commands got %d", len(ls))
				}
			}
		}
	})
}

func TestProjectRunSwitchCommands(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		for _, tc := range testCases {
			err := tc.project.RunSwitchCommands(ctx)
			time.Sleep(time.Second)
			if tc.initialStatus() == PROJECT_STATUS_INACTIVE {
				if err != ErrSessionNotFound {
					t.Error("Should retrun sesison not found error for inactive projects")
				}
			} else {
				if err != nil {
					t.Error(err)
				}
				ls, err := os.ReadDir(tc.project.Config.Directory)
				if err != nil {
					t.Error(err)
				}
				if strings.Contains(tc.project.Name, "switch") {
					if len(ls) != len(switchCommands) {
						t.Errorf("Expected %d files from switch commands got %d", len(switchCommands), len(ls))
					}
				} else {
					if len(ls) != 0 {
						t.Errorf("Expected 0 files from switch commands got %d", len(ls))
					}
				}
			}
		}
	})
}

func TestProjectReset(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		for _, tc := range testCases {
			initialSessions, err := srv.ListSessions()
			if err != nil {
				t.Error(err)
			}
			res, f, reserr := tc.project.Reset(ctx)
			if f != nil {
				f()
			}
			status, err := tc.project.Status()
			if err != nil {
				t.Error(err)
			}
			switch tc.initialStatus() {
			case PROJECT_STATUS_INACTIVE:
				if !errors.Is(reserr, ErrSessionNotFound) {
					t.Error("Inactive project should return not found error")
				}
				if res != nil {
					t.Error("Should not return a sesison for inactive projects")
				}
			default:
				if reserr != nil {
					t.Error(reserr)
				}
				if status != tc.initialStatus() {
					t.Error("Status should match original")
				}
				if res.Id == tc.session.Id {
					t.Error("Status should not match original")
				}
			}
			finalSessions, err := srv.ListSessions()
			if err != nil {
				t.Error(err)
			}
			if len(initialSessions) != len(finalSessions) {
				t.Error("Number of sessions before and after should match")
			}
		}
	})
}

func TestProjectResetWindowsAndPanes(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		for _, tc := range testCases {
			if tc.initialStatus() == PROJECT_STATUS_INACTIVE {
				continue
			}
			res, f, err := tc.project.Reset(ctx)
			if f != nil {
				f()
			}
			if err != nil {
				t.Error(err)
			}
			time.Sleep(time.Second)
			if strings.Contains(tc.project.Name, "windows") {
				windows, err := res.Windows()
				if err != nil {
					t.Error(err)
				}
				if len(windows) != 3 {
					t.Error("Should have 3 windows")
				}
				for i := range 3 {
					panes, err := windows[i].Panes()
					if err != nil {
						t.Error(err)
					}
					for j := range pane_counts[i] {
						expected := fmt.Sprintf("%d", primes[i]*primes[j])
						out, err := panes[j].Capture()
						if err != nil {
							t.Error(err)
						}
						allOut := strings.Join(out, "")
						if !strings.Contains(allOut, expected) {
							t.Errorf("Missing output %s in %s", expected, allOut)
						}
					}
				}
			} else {
				windows, err := res.Windows()
				if err != nil {
					t.Error(err)
				}
				if len(windows) != 1 {
					t.Errorf("Should have 1 (default) window got %d", len(windows))
				}
			}
		}
	})
}

func TestProjectResetCommands(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(ctx context.Context, srv *tmux.Server) {
		dir, client, testCases := setupTestProjects(t, ctx, srv)
		defer teardownTestProjects(t, dir, client)
		for _, tc := range testCases {
			if tc.initialStatus() == PROJECT_STATUS_INACTIVE {
				continue
			}
			_, f, err := tc.project.Reset(ctx)
			if f != nil {
				f()
			}
			if err != nil {
				t.Error(err)
			}
			time.Sleep(time.Second)
			ls, err := os.ReadDir(tc.project.Config.Directory)
			if err != nil {
				t.Error(err)
			}
			if strings.Contains(tc.project.Name, "switch") && tc.initialStatus() == PROJECT_STATUS_ATTACHED {
				if len(ls) != len(switchCommands) {
					t.Errorf("Expected %d files from switch commands got %d", len(switchCommands), len(ls))
				}
			} else {
				if len(ls) != 0 {
					t.Errorf("Expected 0 files from switch commands got %d", len(ls))
				}
			}
		}
	})
}
