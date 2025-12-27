package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/creack/pty"
	"samuellando.com/tmixer/internal/tmux"
)

func SetupTestClient(tmux *tmux.Server, session *tmux.Session) *os.File {
	cmd := exec.Command("tmux", "-S", tmux.SocketPath, "-u", "attach", "-t", string(session.Id))
	f, err := pty.Start(cmd)
	if err != nil {
		panic(err)
	}
	// Read a single byte, waiting for the proccess to actually start
	f.Read([]byte{0})
	return f
}

func SetupTestServer(t testing.TB) *tmux.Server {
	dir, err := os.MkdirTemp(os.TempDir(), "tmixer")
	if err != nil {
		t.Fatal(err)
	}
	tmux := tmux.Tmux(fmt.Sprintf("%s/test.sock", dir))
	// Start one extra session so the server starts
	_, err = tmux.New("default_test_session")
	if err != nil {
		t.Fatal(fmt.Errorf("Error while starting server %w", err))
	}
	return tmux
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

func RunWithAndWithoutControlMode(f func (tmux *tmux.Server), t *testing.T) {
	tmux := SetupTestServer(t)
	f(tmux)
	TeardownTestServer(tmux)
	// And with control mode
	tmux = SetupTestServer(t)
	err := tmux.StartControlMode()
	if err != nil {
		t.Fatal(err)
	}
	f(tmux)
	err = tmux.StopControlMode() 
	if err != nil {
		t.Fatal(err)
	}
	TeardownTestServer(tmux)
}
