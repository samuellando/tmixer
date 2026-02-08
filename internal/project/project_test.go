package project

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

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

func TestTtlPassed_NoTtlConfigured(t *testing.T) {
	t.Parallel()
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		p := &Project{
			Name:       "ttl-none",
			Config:     &config.ProjectConfig{},
			fullConfig: &config.Config{},
			server:     srv,
		}
		passed, err := p.TtlPassed()
		if err != nil {
			t.Error(err)
		}
		if passed {
			t.Error("Expected ttl not to pass when not configured")
		}
	})
}

func TestTtlPassed_InvalidTtlFormat(t *testing.T) {
	t.Parallel()
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		p := &Project{
			Name:   "ttl-invalid",
			Config: &config.ProjectConfig{},
			fullConfig: &config.Config{
				Ttl: stringPointer("not-a-duration"),
			},
			server: srv,
		}
		passed, err := p.TtlPassed()
		if err == nil {
			t.Error("Expected ttl parse error")
		}
		if passed {
			t.Error("Expected ttl to be false on parse error")
		}
	})
}

func TestTtlPassed_OneHourNotExpired(t *testing.T) {
	t.Parallel()
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		p := &Project{
			Name:   "ttl-one-hour",
			Config: &config.ProjectConfig{Directory: t.TempDir()},
			fullConfig: &config.Config{
				Ttl: stringPointer("1h"),
			},
			server: srv,
		}
		_, err := p.Start(ctx)
		if err != nil {
			t.Fatal(err)
		}
		passed, err := p.TtlPassed()
		if err != nil {
			t.Error(err)
		}
		if passed {
			t.Error("Expected ttl not to pass immediately")
		}
	})
}

func TestTtlPassed_FiveSecondsExpired(t *testing.T) {
	t.Parallel()
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		p := &Project{
			Name:   "ttl-five-seconds",
			Config: &config.ProjectConfig{Directory: t.TempDir()},
			fullConfig: &config.Config{
				Ttl: stringPointer("5s"),
			},
			server: srv,
		}
		_, err := p.Start(ctx)
		if err != nil {
			t.Fatal(err)
		}
		passed, err := p.TtlPassed()
		if err != nil {
			t.Error(err)
		}
		if passed {
			t.Error("Expected ttl not to pass immediately")
		}
		time.Sleep(6 * time.Second)
		passed, err = p.TtlPassed()
		if err != nil {
			t.Error(err)
		}
		if !passed {
			t.Error("Expected ttl to pass after sleep")
		}
	})
}

func TestTtlPassed_InactiveProject(t *testing.T) {
	t.Parallel()
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		p := &Project{
			Name:   "ttl-inactive",
			Config: &config.ProjectConfig{},
			fullConfig: &config.Config{
				Ttl: stringPointer("5s"),
			},
			server: srv,
		}
		passed, err := p.TtlPassed()
		if err == nil {
			t.Error("Expected session not found error")
		}
		if passed {
			t.Error("Expected ttl to be false when inactive")
		}
		if !errors.Is(err, ErrSessionNotFound) {
			t.Errorf("Expected session not found error, got %v", err)
		}
	})
}
