package project

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
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

func setupTestCase(t *testing.T, ctx context.Context, tc *projectTestCase, srv *tmux.Server) *os.File {
	tc.session = setupTestProject(t, ctx, tc.project, srv)
	if strings.HasPrefix(tc.project.Name, "attached") {
		return testutil.SetupTestClient(srv, tc.session)
	} else {
		return testutil.SetupTestClient(srv, nil)
	}
}

func setupTestProject(t *testing.T, ctx context.Context, project *Project, srv *tmux.Server) *tmux.Session {
	dir := t.TempDir()
	project.Config.Directory = dir
	project.server = srv
	if strings.HasPrefix(project.Name, "active") || strings.HasPrefix(project.Name, "attached") {
		s, err := project.Start(ctx)
		if err != nil {
			t.Error(err)
		}
		return s
	}
	return nil
}

func getAllTestCases() []*projectTestCase {
	testCases := make([]*projectTestCase, 0)
	for name, pc := range testConfig.Projects {
		tc := projectTestCase{}
		// Make a copy of the config to avoid race conditions in parallel tests
		configCopy := *pc
		p := Project{
			Name:       name,
			Config:     &configCopy,
			fullConfig: testConfig,
		}
		tc.project = &p
		testCases = append(testCases, &tc)
	}
	return testCases
}

func teardownTestCase(t *testing.T, client *os.File) {
	err := client.Close()
	if err != nil {
		t.Error(err)
	}
}

func runAllTestCases(t *testing.T, f func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase)) {
	t.Parallel()
	for _, tc := range getAllTestCases() {
		t.Run(tc.project.Name, func(t *testing.T) {
			t.Parallel()
			testutil.RunWithAndWithoutControlModeTestRun(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
				client := setupTestCase(t, ctx, tc, srv)
				defer teardownTestCase(t, client)
				f(t, ctx, srv, tc)
			})
		})
	}
}

func TestProjectStatus(t *testing.T) {
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		status, err := tc.project.Status()
		if err != nil {
			t.Error(err)
		}
		if tc.initialStatus() != status {
			t.Errorf("Status does not match %d != %d", tc.initialStatus(), status)
		}
	})
}

func TestProjectSession(t *testing.T) {
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
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
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
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
	})
}

func TestProjectStart(t *testing.T) {
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
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
	})
}

func TestProjectStartWindowsAndPanes(t *testing.T) {
	maxAttempts := 60
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		res, err := tc.project.Start(ctx)
		if err != nil {
			t.Error(err)
		}
		ticker := time.Tick(time.Second)
	tickloop:
		for i := 0; ; i++ {
			<-ticker
			if strings.Contains(tc.project.Name, "windows") {
				windows, err := res.Windows()
				if err != nil {
					t.Error(err)
				}
				if len(windows) != 3 {
					if i == maxAttempts {
						t.Error("Should have 3 windows")
						break tickloop
					} else {
						continue tickloop
					}
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
							if i == maxAttempts {
								t.Errorf("Missing output %s in %s", expected, allOut)
								break tickloop
							} else {
								continue tickloop
							}
						}
					}
				}
			} else {
				windows, err := res.Windows()
				if err != nil {
					t.Error(err)
				}
				if len(windows) != 1 {
					if i == maxAttempts {
						t.Errorf("Should have 1 (default) window got %d", len(windows))
						break tickloop
					} else {
						continue tickloop
					}
				}
			}
			break tickloop
		}
	})
}

func TestProjectKill(t *testing.T) {
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		initialSession, _ := tc.project.Session()
		cleanup, res := tc.project.Kill(ctx)
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
			err = cleanup()
			if err != nil {
				t.Error(err)
			}
		case PROJECT_STATUS_ATTACHED:
			if res != nil {
				t.Error(res)
			}
			// old session is still active
			if !srv.HasSession(initialSession) {
				t.Error("Original session should still be active")
			}
			err = cleanup()
			if err != nil {
				t.Error(err)
			}
			if srv.HasSession(initialSession) {
				t.Error("Should kill the original session")
			}
		case PROJECT_STATUS_ACTIVE:
			if res != nil {
				t.Error(res)
			}
			err = cleanup()
			if err != nil {
				t.Error(err)
			}
		default:
			t.Error("Not implemented")
		}
	})
}

func TestProjectKillAttachedLastActive(t *testing.T) {
	testutil.RunWithAndWithoutControlModeTestRun(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		testCases := getAllTestCases()
		for _, tc := range testCases {
			if tc.initialStatus() == PROJECT_STATUS_ATTACHED {
				client := setupTestCase(t, ctx, tc, srv)
				defer teardownTestCase(t, client)
			} else {
				setupTestProject(t, ctx, tc.project, srv)
			}
		}
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
				_, err := tc.project.Switch(ctx)
				if err != nil {
					t.Error(err)
				}
				_, err = attached.Switch(ctx)
				if err != nil {
					t.Error(err)
				}
				lastActive = tc.project
				time.Sleep(time.Second)
			}
		}
		cleanup, err := attached.Kill(ctx)
		cleanup()
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
	testutil.RunWithAndWithoutControlModeTestRun(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		testCases := getAllTestCases()
		for _, tc := range testCases {
			if tc.initialStatus() == PROJECT_STATUS_ATTACHED {
				client := setupTestCase(t, ctx, tc, srv)
				defer teardownTestCase(t, client)
			} else {
				setupTestProject(t, ctx, tc.project, srv)
			}
		}
		var attached *Project
		var defProj *Project
		for _, tc := range testCases {
			if tc.initialStatus() == PROJECT_STATUS_ATTACHED {
				attached = tc.project
			}
			if tc.initialStatus() == PROJECT_STATUS_ACTIVE {
				cleanup, err := tc.project.Kill(ctx)
				cleanup()
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
		cleanup, err := attached.Kill(ctx)
		cleanup()
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
	testutil.RunWithAndWithoutControlModeTestRun(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		testCases := getAllTestCases()
		for _, tc := range testCases {
			if tc.initialStatus() == PROJECT_STATUS_ATTACHED {
				client := setupTestCase(t, ctx, tc, srv)
				defer teardownTestCase(t, client)
			} else {
				setupTestProject(t, ctx, tc.project, srv)
			}
		}
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
		cleanup, err := attached.Kill(ctx)
		cleanup()
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
	testutil.RunWithAndWithoutControlModeTestRun(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		testCases := getAllTestCases()
		for _, tc := range testCases {
			if tc.initialStatus() == PROJECT_STATUS_ATTACHED {
				client := setupTestCase(t, ctx, tc, srv)
				defer teardownTestCase(t, client)
			} else {
				setupTestProject(t, ctx, tc.project, srv)
			}
		}
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
		cleanup, err := attached.Kill(ctx)
		cleanup()
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
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		session, err := tc.project.Switch(ctx)
		if err != nil {
			t.Error(err)
		}
		if session == nil {
			t.Error("Should return a session")
		}
		status, err := tc.project.Status()
		if err != nil {
			t.Error(err)
		}
		if status != PROJECT_STATUS_ATTACHED {
			t.Error("Should attache to project")
		}
	})
}

func TestProjectSwitchCreatesAttachesSession(t *testing.T) {
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		session, err := tc.project.Switch(ctx)
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
	})
}

func TestProjectSwitchWindowsAndPanes(t *testing.T) {
	maxAttempts := 60
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		res, err := tc.project.Switch(ctx)
		if err != nil {
			t.Error(err)
		}
		ticker := time.Tick(time.Second)
	tickLoop:
		for i := 0; ; i++ {
			<-ticker
			if strings.Contains(tc.project.Name, "windows") {
				windows, err := res.Windows()
				if err != nil {
					t.Error(err)
				}
				if len(windows) != 3 {
					if i == maxAttempts {
						t.Error("Should have 3 windows")
						break tickLoop
					} else {
						continue tickLoop
					}
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
							if i == maxAttempts {
								t.Errorf("Missing output %s in %s", expected, allOut)
								break tickLoop
							} else {
								continue tickLoop
							}
						}
					}
				}
			} else {
				windows, err := res.Windows()
				if err != nil {
					t.Error(err)
				}
				if len(windows) != 1 {
					if i == maxAttempts {
						t.Errorf("Should have 1 (default) window got %d", len(windows))
						break tickLoop
					} else {
						continue tickLoop
					}
				}
			}
			break tickLoop
		}
	})
}

func TestProjectSwitchCommands(t *testing.T) {
	maxAttempts := 60
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		_, err := tc.project.Switch(ctx)
		if err != nil {
			t.Error(err)
		}

		ticker := time.Tick(time.Second)

	tickLoop:
		for i := 0; ; i++ {
			<-ticker
			ls, err := os.ReadDir(tc.project.Config.Directory)
			if err != nil {
				if i == maxAttempts {
					t.Error(err)
					break tickLoop
				}
				continue tickLoop
			}
			if strings.Contains(tc.project.Name, "switch") {
				if len(ls) == len(switchCommands) {
					break tickLoop
				} else if i == maxAttempts {
					t.Errorf("Expected %d files from switch commands got %d", len(switchCommands), len(ls))
					break tickLoop
				}
			} else {
				if len(ls) == 0 {
					break tickLoop
				} else if i == maxAttempts {
					t.Errorf("Expected 0 files from switch commands got %d", len(ls))
					break tickLoop
				}
			}
		}
	})
}

func TestProjectRunSwitchCommands(t *testing.T) {
	maxAttempts := 60
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		err := tc.project.RunSwitchCommands(ctx)
		if tc.initialStatus() == PROJECT_STATUS_INACTIVE {
			if err != ErrSessionNotFound {
				t.Error("Should retrun sesison not found error for inactive projects")
			}
		} else {
			if err != nil {
				t.Error(err)
			}
			ticker := time.Tick(time.Second)
		tickLoop:
			for i := 0; ; i++ {
				<-ticker
				ls, err := os.ReadDir(tc.project.Config.Directory)
				if err != nil {
					if i == maxAttempts {
						t.Error(err)
						break tickLoop
					}
					continue tickLoop
				}
				if strings.Contains(tc.project.Name, "switch") {
					if len(ls) == len(switchCommands) {
						break tickLoop
					} else if i == maxAttempts {
						t.Errorf("Expected %d files from switch commands got %d", len(switchCommands), len(ls))
						break tickLoop
					}
				} else {
					if len(ls) == 0 {
						break tickLoop
					} else if i == maxAttempts {
						t.Errorf("Expected 0 files from switch commands got %d", len(ls))
						break tickLoop
					}
				}

			}
		}
	})
}

func TestProjectReset(t *testing.T) {
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		initialSessions, err := srv.ListSessions()
		if err != nil {
			t.Error(err)
		}
		initialSession, _ := tc.project.Session()
		res, cleanup, reserr := tc.project.Reset(ctx)
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
			err = cleanup()
			if err != nil {
				t.Error(err)
			}
		case PROJECT_STATUS_ATTACHED:
			if reserr != nil {
				t.Error(reserr)
			}
			if status != tc.initialStatus() {
				t.Error("Status should match original")
			}
			if res.Id == tc.session.Id {
				t.Error("Status should not match original")
			}
			if !srv.HasSession(initialSession) {
				t.Error("Original session should still be active")
			}
			err = cleanup()
			if err != nil {
				t.Error(err)
			}
			if srv.HasSession(initialSession) {
				t.Error("Should kill the original session")
			}
		case PROJECT_STATUS_ACTIVE:
			if reserr != nil {
				t.Error(reserr)
			}
			if status != tc.initialStatus() {
				t.Error("Status should match original")
			}
			if res.Id == tc.session.Id {
				t.Error("Status should not match original")
			}
			err = cleanup()
			if err != nil {
				t.Error(err)
			}
		default:
			t.Error("Not implemented")
		}
		finalSessions, err := srv.ListSessions()
		if err != nil {
			t.Error(err)
		}
		if len(initialSessions) != len(finalSessions) {
			t.Error("Number of sessions before and after should match")
		}
	})
}

func TestProjectResetWindowsAndPanes(t *testing.T) {
	maxAttempts := 60
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		if tc.initialStatus() == PROJECT_STATUS_INACTIVE {
			return
		}
		res, cleanup, err := tc.project.Reset(ctx)
		cleanup()
		if err != nil {
			t.Error(err)
		}

		ticker := time.Tick(time.Second)

	tickLoop:
		for i := 0; ; i++ {
			<-ticker
			if strings.Contains(tc.project.Name, "windows") {
				windows, err := res.Windows()
				if err != nil {
					t.Error(err)
				}
				if len(windows) != 3 {
					if i == maxAttempts {
						t.Error("Should have 3 windows")
						break tickLoop
					} else {
						continue tickLoop
					}
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
							if i == maxAttempts {
								t.Errorf("Missing output %s in %s", expected, allOut)
								break tickLoop
							} else {
								continue tickLoop
							}
						}
					}
				}
			} else {
				windows, err := res.Windows()
				if err != nil {
					t.Error(err)
				}
				if len(windows) != 1 {
					if i == maxAttempts {
						t.Errorf("Should have 1 (default) window got %d", len(windows))
						break tickLoop
					} else {
						continue tickLoop
					}
				}
			}
			break tickLoop
		}
	})
}

func TestProjectResetCommands(t *testing.T) {
	maxAttempts := 60
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		if tc.initialStatus() == PROJECT_STATUS_INACTIVE {
			return
		}
		_, cleanup, err := tc.project.Reset(ctx)
		cleanup()
		if err != nil {
			t.Error(err)
		}
		ticker := time.Tick(time.Second)
	tickloop:
		for i := 1; ; i++ {
			<-ticker
			ls, err := os.ReadDir(tc.project.Config.Directory)
			if err != nil {
				if i == maxAttempts {
					t.Error(err)
					break tickloop
				}
				continue tickloop
			}
			if strings.Contains(tc.project.Name, "switch") && tc.initialStatus() == PROJECT_STATUS_ATTACHED {
				if len(ls) == len(switchCommands) {
					break tickloop
				} else if i == maxAttempts {
					t.Errorf("Expected %d files from switch commands got %d", len(switchCommands), len(ls))
					break tickloop
				}
			} else {
				if len(ls) == 0 {
					break tickloop
				} else if i == maxAttempts {
					t.Errorf("Expected 0 files from switch commands got %d in %s", len(ls), tc.project.Config.Directory)
					break tickloop
				}
			}
		}
	})
}
