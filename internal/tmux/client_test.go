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
		s, err := srv.New("client_session")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := srv.New("client_session2"); err != nil {
			t.Fatal(err)
		}
		s3, err := srv.New("client_session3")
		if err != nil {
			t.Fatal(err)
		}
		f := testutil.SetupTestClient(srv, s)
		defer func() {
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
		}()
		client, err := srv.ActiveClient()
		if err != nil {
			t.Fatal(err)
		}
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
		s, err := srv.New("client_session")
		if err != nil {
			t.Fatal(err)
		}
		f := testutil.SetupTestClient(srv, s)
		err = srv.DisplayMessage(message)
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			time.Sleep(2 * time.Second)
			if err := f.Close(); err != nil {
				t.Logf("error closing client in goroutine: %v", err)
			}
		}()
		out, err := io.ReadAll(f)
		if err == nil {
			// Since we we closing the file without allowing the stream to finish
			t.Fatal("Expected an error")
		}
		if !strings.Contains(string(out), message) {
			t.Error("Message missing")
		}
	})
}
