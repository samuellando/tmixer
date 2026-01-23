package fzf_test

import (
	"strings"
	"testing"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/fzf"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/testutil"
)

func TestDisplayProjectsEventFields(t *testing.T) {
	ctx, tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)

	// Set up logging
	ctx, logger, out := testutil.SetupLogging(ctx, log.LEVEL_DEBUG)

	// Create a simple config with one project
	cfg := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"test-project": {
				Directory: "/tmp/test",
			},
		},
	}

	// Get the projects list
	projects, err := project.List(ctx, tmux, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Call DisplayProjects
	w := &strings.Builder{}
	err = fzf.DisplayProjects(ctx, projects, w)
	if err != nil {
		t.Fatal(err)
	}

	res := testutil.GetLogEvent(ctx, logger, out)

	// Check displayProjectsEvent exists
	event, ok := res["displayProjectsEvent"].(map[string]any)
	if !ok {
		t.Fatal("displayProjectsEvent not found in log output")
	}

	// errors field should be omitted when there are no errors
	if _, ok := event["errors"]; ok {
		t.Error("'errors' field should be omitted when there are no errors")
	}
}
