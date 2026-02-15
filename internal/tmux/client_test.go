package tmux_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

func TestSwitchAndClientSession(t *testing.T) {
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		s, _ := srv.New("client_session")
		srv.New("client_session2")
		s3, _ := srv.New("client_session3")
		f := testutil.SetupTestClient(srv, s)
		defer f.Close()
		client, err := srv.ActiveClient()
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
	})
}

func TestDisplayMessage(t *testing.T) {
	message := "This is a test message"
	testutil.RunWithAndWithoutControlMode(t, func(t *testing.T, ctx context.Context, srv *tmux.Server) {
		s, _ := srv.New("client_session")
		f := testutil.SetupTestClient(srv, s)
		defer f.Close()
		err := srv.DisplayMessage(message)
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			time.Sleep(2 * time.Second)
			f.Close()
		}()
		out, _ := io.ReadAll(f)
		if !strings.Contains(string(out), message) {
			t.Error("Message missing")
		}
	})
}
