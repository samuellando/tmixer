package testutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/creack/pty"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/tmux"
)

var DEFAULT_TEST_SESSION = "default_test_session"

func SetupTestClient(tmux *tmux.Server, session *tmux.Session) *os.File {
	cmd := exec.Command("tmux", "-S", tmux.SocketPath, "-u", "attach", "-t", string(session.Id))
	f, err := pty.Start(cmd)
	if err != nil {
		panic(err)
	}
	// Read a single byte, waiting for the proccess to actually start
	time.Sleep(time.Second)
	buff := make([]byte, 100)
	_, err = f.Read(buff)
	if err != nil {
		panic(err)
	}
	return f
}

func SetupTestServer(t testing.TB) (context.Context, *tmux.Server) {
	ctx, _ := log.New(context.Background(), nil)
	dir, err := os.MkdirTemp(os.TempDir(), "tmixer")
	if err != nil {
		t.Fatal(err)
	}
	tmux := tmux.Tmux(ctx, fmt.Sprintf("%s/test.sock", dir))
	// Start one extra session so the server starts
	_, err = tmux.New(DEFAULT_TEST_SESSION)
	if err != nil {
		t.Fatal(fmt.Errorf("Error while starting server %w", err))
	}
	return ctx, tmux
}

func TeardownTestServer(s *tmux.Server) {
	path := s.SocketPath
	dir := filepath.Dir(path)
	err := s.Kill()
	if err != nil {
		fmt.Println("Problem killing test server")
		fmt.Println(err)
	}
	err = os.RemoveAll(dir)
	if err != nil {
		fmt.Println("Problem clearing test server path")
		fmt.Println(err)
	}
}

func RunWithAndWithoutControlMode(t *testing.T, f func(ctx context.Context, tmux *tmux.Server)) {
	ctx, tmux := SetupTestServer(t)
	f(ctx, tmux)
	TeardownTestServer(tmux)
	// And with control mode
	ctx, tmux = SetupTestServer(t)
	err := tmux.StartControlMode()
	if err != nil {
		t.Fatal(err)
	}
	f(ctx, tmux)
	err = tmux.StopControlMode()
	if err != nil {
		t.Fatal(err)
	}
	TeardownTestServer(tmux)
}
