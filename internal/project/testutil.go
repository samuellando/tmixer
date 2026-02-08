package project

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

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
	Projects: []*config.ProjectConfig{
		{
			Name: "inactive",
		},
		{
			Name:    "inactive-windows",
			Windows: testWindowConfigs,
		},
		{
			Name:           "inactive-switch",
			SwitchCommands: switchCommands,
		},
		{
			Name:           "inactive-windows-switch",
			Windows:        testWindowConfigs,
			SwitchCommands: switchCommands,
		},
		{
			Name: "active",
		},
		{
			Name:    "active-windows",
			Windows: testWindowConfigs,
		},
		{
			Name:           "active-switch",
			SwitchCommands: switchCommands,
		},
		{
			Name:           "active-windows-switch",
			Windows:        testWindowConfigs,
			SwitchCommands: switchCommands,
		},
		{
			Name:           "attached-switch",
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
	for _, pc := range testConfig.Projects {
		tc := projectTestCase{}
		// Make a copy of the config to avoid race conditions in parallel tests
		configCopy := *pc
		p := Project{
			Name:       pc.Name,
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
			testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
				client := setupTestCase(t, ctx, tc, srv)
				defer teardownTestCase(t, client)
				f(t, ctx, srv, tc)
			})
		})
	}
}
