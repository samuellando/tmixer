package display_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/display"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/testutil"
)

func TestDisplayProjectsEventFields(t *testing.T) {
	ctx, tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)

	// Set up logging
	ctx = testutil.SetupLoggingV2(ctx, log.LEVEL_DEBUG)

	// Create a simple config with one project
	cfg := &config.Config{
		Projects: []*config.ProjectConfig{
			{
				Name:      "test-project",
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
	result, err := display.Projects(ctx, projects)
	if err != nil {
		t.Fatal(err)
	}
	anyResult := make([]any, len(result))
	for i, r := range result {
		anyResult[i] = r
	}

	res := testutil.GetLogEventV2(t, ctx)

	// Check displayProjects exists
	event, ok := res["displayProjects"].(map[string]any)
	if !ok {
		t.Fatal("displayProjects not found in log output")
	}

	// errors field should be omitted when there are no errors
	if _, ok := event["errors"]; ok {
		t.Error("'errors' field should be omitted when there are no errors")
	}
	if diff := cmp.Diff(event["result"], anyResult); diff != "" {
		t.Errorf("The result in the log should match the actual result %s", diff)
	}
}
