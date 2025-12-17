package tmuxv2

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
)

const TEST_SOCKET_DIR = "/tmp/tmixer-test"

func TestMain(m *testing.M) {
	// Run tests
	setupTestServersDir()
	code := m.Run()
	cleanUpTestServers()
	os.Exit(code)
}

func setupTestServer(t testing.TB) *Server {
	uuid := uuid.NewString()
	tmux := Tmux(fmt.Sprintf("%s/%s.sock", TEST_SOCKET_DIR, uuid))
	// Start one extra session so the server starts
	_, err := tmux.command("new").detached().withSession("default_test_session").run()
	if err != nil {
		t.Fatal(fmt.Errorf("Error while starting server %w", err))
	}
	return tmux
}

func teardownTestServer(s *Server) {
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
