package tmuxv2_test

import (
	"testing"

	"samuellando.com/tmixer/internal/testutil"
)

func TestSwitchAndClientSession(t *testing.T) {
	tmux := testutil.SetupTestServer(t)
	defer testutil.TeardownTestServer(tmux)
	s, _ := tmux.New("client_session")
	tmux.New("client_session2")
	s3, _ := tmux.New("client_session3")
	f := testutil.SetupTestClient(tmux, s)
	defer f.Close()
	res, err := tmux.ClientSession()
	if err != nil {
		t.Fatal(err)
	}
	if res.Id != s.Id {
		t.Fatalf("Expected first session %v [%v]", res, s)
	}
	err = tmux.Switch(s3)
	if err != nil {
		t.Fatal(err)
	}
	res, err = tmux.ClientSession()
	if err != nil {
		t.Fatal(err)
	}
	if res.Id != s3.Id {
		t.Fatalf("Expected third session %v [%v]", res, s3)
	}
}
