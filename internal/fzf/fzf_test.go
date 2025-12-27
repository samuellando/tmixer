package fzf

import (
	"strings"
	"testing"

	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/project"
	"samuellando.com/tmixer/internal/testutil"
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
				Directory: "/home/test/bin",
			},
			"bin": {
				Directory: "/home/test/bin",
			},
			"cats": {
				Directory: "/home/test/bin",
			},
			"dog": {
				Directory: "/home/test/bin",
			},
			"zzz": {
				Directory: "/home/test/bin",
			},
			"cat": {
				Directory: "/home/test/bin",
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

	projects, _ := project.List(tmux, config)
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
	err := displayProjects(projects, w)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(w.b.String()) != expected {
		t.Fatalf("\n%s\n!=\n%s", w.b.String(), expected)
	}
}
