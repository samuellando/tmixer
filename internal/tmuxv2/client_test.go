package tmuxv2_test

import (
	"testing"

	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmuxv2"
)

func TestSwitchAndClientSession(t *testing.T) {
	f := func(tmux *tmuxv2.Server) {
		s, _ := tmux.New("client_session")
		tmux.New("client_session2")
		s3, _ := tmux.New("client_session3")
		f := testutil.SetupTestClient(tmux, s)
		defer f.Close()
		client, err := tmux.ActiveClient()
		res, err := client.Session()
		if err != nil {
			t.Fatal(err)
		}
		if res.Id != s.Id {
			t.Fatalf("Expected first session %v [%v]", res, s)
		}
		err = client.Switch(s3)
		if err != nil {
			t.Fatal(err)
		}
		res, err = client.Session()
		if err != nil {
			t.Fatal(err)
		}
		if res.Id != s3.Id {
			t.Fatalf("Expected third session %v [%v]", res, s3)
		}
	}
	testutil.RunWithAndWithoutControlMode(f, t)
}
