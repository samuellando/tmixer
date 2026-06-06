package display_test

import (
	"sort"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/display"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/testutil"

	"github.com/google/go-cmp/cmp"
)

func TestDisplayProjects(t *testing.T) {
	ctx, tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
	s, _ := tmux.New("test-session")
	f := testutil.SetupTestClient(tmux, s)
	defer func() {
		err := f.Close()
		if err != nil {
			t.Error(err)
		}
	}()
	config := &config.Config{
		Projects: []*config.ProjectConfig{
			{
				Name:      "dogs",
				Directory: "/home/test/bin1",
			},
			{
				Name:      "bin",
				Directory: "/home/test/bin2",
			},
			{
				Name:      "cats",
				Directory: "/home/test/bin3",
			},
			{
				Name:      "dog",
				Directory: "/home/test/bin4",
			},
			{
				Name:      "zzz",
				Directory: "/home/test/bin5",
			},
			{
				Name:      "cat",
				Directory: "/home/test/bin6",
			},
		},
	}
	expected := []string{
		"\033[31m\033[0m zzz",
		" dogs",
		" cats",
		" default_test_session",
		" test-session",
		" bin",
		" cat",
		" dog",
	}

	projects, err := project.List(ctx, tmux, config)
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Config != nil && projects[j].Config != nil {
			return projects[i].Config.Directory > projects[j].Config.Directory
		}
		return false
	})
	for _, p := range projects {
		switch p.Name {
		case "cats":
			_, err = p.Start(ctx)
			if err != nil {
				t.Fatal(err)
			}
		case "dogs":
			_, err = p.Start(ctx)
			if err != nil {
				t.Fatal(err)
			}
		case "zzz":
			_, err = p.Start(ctx)
			if err != nil {
				t.Fatal(err)
			}
			_, err = p.Switch(ctx)
			if err != nil {
				t.Fatal(err)
			}
		}
		time.Sleep(time.Second)
	}
	out, err := display.Projects(ctx, projects)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(expected, out); diff != "" {
		t.Fatal(diff)
	}
}

func TestGetSelectedProject(t *testing.T) {
	if got, err := display.GetProjectNameFromOutput("i beta"); got != "beta" && err != nil {
		t.Fatalf("unexpected project: %#v", got)
	}
	if got, err := display.GetProjectNameFromOutput("i missing"); got != "missing" && err != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
	if got, err := display.GetProjectNameFromOutput("badFormat"); got != "" && err == nil {
		t.Fatal("expected an error for a bad format")
	}
}
