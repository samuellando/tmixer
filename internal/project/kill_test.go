package project

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

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
				t.Error("Inactive project give session not found error")
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
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
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
				time.Sleep(5 * time.Second)
			}
		}
		cleanup, err := attached.Kill(ctx)
		if err != nil {
			t.Error(err)
		}
		err = cleanup()
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
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
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
				if err != nil {
					t.Error(err)
				}
				err = cleanup()
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
		if err != nil {
			t.Error(err)
		}
		err = cleanup()
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
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
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
		if err != nil {
			t.Error(err)
		}
		err = cleanup()
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
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
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
			Projects: []*config.ProjectConfig{
				attached.Config,
			},
		}
		cleanup, err := attached.Kill(ctx)
		if err != nil {
			t.Error(err)
		}
		err = cleanup()
		if err != nil {
			t.Error(err)
		}
		_, err = srv.ActiveClient()
		if err == nil {
			t.Error(err)
		}
	})
}
