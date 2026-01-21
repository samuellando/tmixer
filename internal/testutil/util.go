package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/creack/pty"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/tmux"
)

func SetupLogging(ctx context.Context, level int) (context.Context, *log.Logger, *bytes.Buffer) {
	ctx, logger := log.New(ctx, &log.LoggerOptions{Level: level})
	out := &bytes.Buffer{}
	logger.AddSink(out)
	return ctx, logger, out
}

func GetLogEvent(ctx context.Context, logger *log.Logger, out *bytes.Buffer) map[string]any {
	logger.Info(ctx)
	res := make(map[string]any)
	json.Unmarshal(out.Bytes(), &res)
	return res
}

var DEFAULT_TEST_SESSION = "default_test_session"

func SetupTestClient(tmux *tmux.Server, session *tmux.Session) *os.File {
	if session == nil {
		session, _ = tmux.New("test_session")
	}
	cmd := exec.Command("tmux", "-S", tmux.SocketPath, "-u", "attach", "-t", string(session.Id))
	f, err := pty.Start(cmd)
	if err != nil {
		panic(err)
	}
	// Wait for the tmux session to become responsive or timeout
	buff := make([]byte, 256)
	readErrChan := make(chan error, 1)
	go func() {
		_, err := f.Read(buff)
		readErrChan <- err
	}()

	select {
	case err := <-readErrChan:
		if err != nil {
			f.Close()
			panic(fmt.Errorf("failed to read from tmux client: %w", err))
		}
	case <-time.After(2 * time.Second):
		f.Close()
		panic(fmt.Errorf("timeout waiting for tmux client responsiveness"))
	}

	return f
}

func SetupTestServer(t testing.TB) (context.Context, *tmux.Server) {
	ctx, _ := log.New(context.Background(), nil)
	dir := t.TempDir()
	tmux := tmux.Tmux(ctx, fmt.Sprintf("%s/test.sock", dir))
	// Start one extra session so the server starts
	_, err := tmux.New(DEFAULT_TEST_SESSION)
	if err != nil {
		t.Fatal(fmt.Errorf("Error while starting server %w", err))
	}
	return ctx, tmux
}

func TeardownTestServer(s *tmux.Server) {
	err := s.Kill()
	if err != nil {
		fmt.Println("Problem killing test server")
		fmt.Println(err)
	}
}

// Deprecated: Should use the TestRun version to enable parallelism
func RunWithAndWithoutControlMode(t *testing.T, f func(ctx context.Context, tmux *tmux.Server)) {
	ctx, tmux := SetupTestServer(t)
	f(ctx, tmux)
	TeardownTestServer(tmux)
	// Give shell processes and tmux server time to fully exit
	time.Sleep(500 * time.Millisecond)
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
	// Give shell processes and tmux server time to fully exit
	time.Sleep(500 * time.Millisecond)
}

func RunWithAndWithoutControlModeTestRun(t *testing.T, f func(t *testing.T, ctx context.Context, tmux *tmux.Server)) {
	t.Run("noControlMode", func(t *testing.T) {
		t.Parallel()
		ctx, tmux := SetupTestServer(t)
		f(t, ctx, tmux)
		TeardownTestServer(tmux)
		// Give shell processes and tmux server time to fully exit
		time.Sleep(500 * time.Millisecond)
	})
	t.Run("ControlMode", func(t *testing.T) {
		ctx, tmux := SetupTestServer(t)
		err := tmux.StartControlMode()
		if err != nil {
			t.Fatal(err)
		}
		f(t, ctx, tmux)
		tmux.StopControlMode()
		TeardownTestServer(tmux)
		// Give shell processes and tmux server time to fully exit
		time.Sleep(500 * time.Millisecond)
	})
}
