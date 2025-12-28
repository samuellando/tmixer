package fzf

import (
	"sort"
	"strings"
	"testing"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/testutil"

	"github.com/google/go-cmp/cmp"
)

type testWriter struct {
	b strings.Builder
}

func (w *testWriter) Write(p []byte) (int, error) {
	return w.b.Write(p)
}

func (w *testWriter) Close() error {
	return nil
}

func TestDisplayProjects(t *testing.T) {
	// TODO: Sub dirs
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
	s, _ := tmux.New("test-session")
	f := testutil.SetupTestClient(tmux, s)
	defer f.Close()
	config := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"dogs": {
				Directory: "/home/test/bin1",
			},
			"bin": {
				Directory: "/home/test/bin2",
			},
			"cats": {
				Directory: "/home/test/bin3",
			},
			"dog": {
				Directory: "/home/test/bin4",
			},
			"zzz": {
				Directory: "/home/test/bin5",
			},
			"cat": {
				Directory: "/home/test/bin6",
			},
		},
	}
	expected := "\033[31m\033[0m zzz"
	expected = expected + `
 dogs
 cats
 default_test_session
 test-session
 bin
 cat
 dog`

	projects, err := project.List(tmux, config)
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Config != nil && projects[j].Config != nil {
			return projects[i].Config.Directory < projects[j].Config.Directory
		}
		return false
	})
	for _, p := range projects {
		switch p.Name {
		case "cats":
			p.Start()
		case "dogs":
			p.Start()
		case "zzz":
			p.Start()
			p.Switch()
		}
	}
	w := &testWriter{}
	err = displayProjects(projects, w)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(strings.TrimSpace(w.b.String()), expected); diff != "" {
		t.Fatal(diff)
	}
}
