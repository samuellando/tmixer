package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/creack/pty"
	"github.com/google/uuid"
	"samuellando.com/tmixer/internal/tmuxv2"
)

const TEST_SOCKET_DIR = "/tmp/tmixer-test"

func TestMain(m *testing.M) {
	// Run tests
	setupTestServersDir()
	code := m.Run()
	cleanUpTestServers()
	os.Exit(code)
}

func SetupTestClient(tmux *tmuxv2.Server, session *tmuxv2.Session) *os.File {
	cmd := exec.Command("tmux", "-S", tmux.SocketPath, "-u", "attach", "-t", session.Id)
	f, err := pty.Start(cmd)
	if err != nil {
		panic(err)
	}
	// Read a single byte, waiting for the proccess to actually start
	f.Read([]byte{0})
	return f
}

func SetupTestServer(t testing.TB) *tmuxv2.Server {
	uuid := uuid.NewString()
	tmux := tmuxv2.Tmux(fmt.Sprintf("%s/%s.sock", TEST_SOCKET_DIR, uuid))
	// Start one extra session so the server starts
	_, err := tmux.New("default_test_session")
	if err != nil {
		t.Fatal(fmt.Errorf("Error while starting server %w", err))
	}
	return tmux
}

func TeardownTestServer(s *tmuxv2.Server) {
	err := s.Kill()
	if err != nil {
		fmt.Println("Problem killing test server")
		fmt.Println(err)
	}
}

func setupTestServersDir() {
	err := os.MkdirAll(TEST_SOCKET_DIR, 0o700)
	if err != nil {
		panic(err)
	}
}

func cleanUpTestServers() {
	err := os.RemoveAll(TEST_SOCKET_DIR)
	if err != nil {
		fmt.Println("Problem clearing test server path")
		fmt.Println(err)
	}
}
