package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

func TestLogs(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(home, ".local/state/tmixer/logs")
	testutil.CaptureStdout(func() {
		f, err := log.RotateLogFile(logPath, 24*time.Hour)
		logFile := f.Name()
		stat, err := f.Stat()
		origSize := stat.Size()
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
		sockefile := filepath.Join(t.TempDir(), "tmux.sock")
		err = run("-S", sockefile, "--help")
		if err != nil {
			t.Fatal(err)
		}
		out, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatal(err)
		}
		data := make(map[string]any)
		if len(out) <= int(origSize) {
			t.Error("The log file should contain more data")
		}
		parts := bytes.Split(out, []byte{'\n'})
		out = parts[len(parts)-2]
		err = json.Unmarshal(out, &data)
		if err != nil {
			t.Error(err)
		}
		if data["level"].(string) != "INFO" {
			t.Error("The log file should contain a valid event")
		}
	})
}

func TestLogFile(t *testing.T) {
	testutil.CaptureStdout(func() {
		logFile := filepath.Join(t.TempDir(), "log.jsonl")
		sockefile := filepath.Join(t.TempDir(), "tmux.sock")
		err := run("-S", sockefile, "--help", "--logFile", logFile)
		if err != nil {
			t.Error(err)
		}
		out, err := os.ReadFile(logFile)
		if err != nil {
			t.Error(err)
		}
		data := make(map[string]any)
		err = json.Unmarshal(out, &data)
		if err != nil {
			t.Error(err)
		}
		if data["level"].(string) != "INFO" {
			t.Error("The log file should contain a valid event")
		}
	})
}

func TestDisplayHelp(t *testing.T) {
	testutil.GoldenTest(t, testutil.CaptureStdout(func() {
		err := run("--help")
		if err != nil {
			t.Error(err)
		}
	}))
}

func TestLoadConfig(t *testing.T) {
	config := `
projects:
  test-load-config:
    directory: "/tmp"
`
	configFile := filepath.Join(t.TempDir(), "config.yml")
	err := os.WriteFile(configFile, []byte(config), 0755)
	if err != nil {
		t.Fatal(err)
	}
	out := testutil.CaptureStdout(func() {
		sockefile := filepath.Join(t.TempDir(), "tmux.sock")
		err = run("-S", sockefile, "-c", configFile, "list")
		if err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "test-load-config") {
		t.Error("Config did not load")
	}
}

func TestList(t *testing.T) {
	config := `
projects:
  test-list1:
    directory: "/tmp"
  test-list2:
    directory: "/tmp"
`
	configFile := filepath.Join(t.TempDir(), "config.yml")
	err := os.WriteFile(configFile, []byte(config), 0755)
	if err != nil {
		t.Fatal(err)
	}
	testutil.GoldenTest(t, testutil.CaptureStdout(func() {
		sockefile := filepath.Join(t.TempDir(), "tmux.sock")
		err = run("-S", sockefile, "-c", configFile, "--combineProjects", "false", "list")
		if err != nil {
			t.Fatal(err)
		}
	}))
}

func TestSwitch(t *testing.T) {
	config := `
projects:
  test-switch:
    directory: "/tmp"
`
	configFile := filepath.Join(t.TempDir(), "config.yml")
	err := os.WriteFile(configFile, []byte(config), 0755)
	if err != nil {
		t.Fatal(err)
	}
	out := testutil.CaptureStdout(func() {
		_, srv := testutil.SetupTestServer(t)
		defer testutil.TeardownTestServer(srv)
		f := testutil.SetupTestClient(srv, nil)
		defer f.Close()

		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "switch", "test-switch")
		if err != nil {
			t.Fatal(err)
		}
		clt, _ := srv.ActiveClient()
		session, _ := clt.Session()
		name, _ := session.Name()
		if name != "test-switch" {
			t.Error("Wrong session")
		}
	})
	if len(out) != 0 {
		t.Error("Should produce no output")
	}
}

func TestReset(t *testing.T) {
	config := `
projects:
  test-reset:
    directory: "/tmp"
`
	configFile := filepath.Join(t.TempDir(), "config.yml")
	err := os.WriteFile(configFile, []byte(config), 0755)
	if err != nil {
		t.Fatal(err)
	}
	out := testutil.CaptureStdout(func() {
		_, srv := testutil.SetupTestServer(t)
		defer testutil.TeardownTestServer(srv)
		f := testutil.SetupTestClient(srv, nil)
		defer f.Close()

		clt, _ := srv.ActiveClient()

		origSessions, _ := srv.ListSessions()

		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "switch", "test-reset")
		if err != nil {
			t.Fatal(err)
		}

		sessions, _ := srv.ListSessions()
		if len(origSessions)+2 != len(sessions) {
			t.Error("Should have one new session + control")
		}

		initial, _ := clt.Session()

		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "reset", "test-reset")
		if err != nil {
			t.Fatal(err)
		}

		sessions, _ = srv.ListSessions()
		if len(origSessions)+2 != len(sessions) {
			t.Error("Should one nes session + control")
		}
		after, _ := clt.Session()

		if initial.Id == after.Id {
			t.Error("Shouldve reset the session")
		}

	})
	if len(out) != 0 {
		t.Error("Should produce no output")
	}
}

func TestResetCurrent(t *testing.T) {
	t.Parallel()
	config := `
projects:
  test-reset:
    directory: "/tmp"
  test-reset2:
    directory: "/tmp"
`
	configFile := filepath.Join(t.TempDir(), "config.yml")
	err := os.WriteFile(configFile, []byte(config), 0755)
	if err != nil {
		t.Fatal(err)
	}
	out := testutil.CaptureStdout(func() {
		_, srv := testutil.SetupTestServer(t)
		defer testutil.TeardownTestServer(srv)
		f := testutil.SetupTestClient(srv, nil)
		defer f.Close()

		clt, _ := srv.ActiveClient()

		origSessions, _ := srv.ListSessions()

		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "switch", "test-reset")
		if err != nil {
			t.Fatal(err)
		}
		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "switch", "test-reset2")
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(7 * time.Second)

		sessions, _ := srv.ListSessions()
		sessions, _ = srv.ListSessions()
		if len(sessions) != len(origSessions)+3 {
			t.Error(testutil.ErrorNotEqual[int]{Actual: len(sessions), Expected: len(origSessions) + 3})
		}

		initial, _ := clt.Session()
		initialName, _ := initial.Name()
		if initialName != "test-reset2" {
			t.Error(testutil.ErrorNotEqual[string]{Actual: initialName, Expected: "test-reset2"})
		}
		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "reset")
		if err != nil {
			t.Fatal(err)
		}

		sessions, _ = srv.ListSessions()
		if len(sessions) != len(origSessions)+3 {
			t.Error(testutil.ErrorNotEqual[int]{Actual: len(sessions), Expected: len(origSessions) + 3})
		}
		after, _ := clt.Session()
		if after.Id == initial.Id {
			t.Error(testutil.ErrorEqual[tmux.SessionId]{Actual: after.Id, Expected: initial.Id})
		}
		afterName, _ := after.Name()
		if afterName != "test-reset2" {
			t.Error(testutil.ErrorNotEqual[string]{Actual: afterName, Expected: "test-reset2"})
		}
	})
	if len(out) != 0 {
		t.Error("Should produce no output")
	}
}

func TestKill(t *testing.T) {
	config := `
projects:
  test-kill:
    directory: "/tmp"
`
	configFile := filepath.Join(t.TempDir(), "config.yml")
	err := os.WriteFile(configFile, []byte(config), 0755)
	if err != nil {
		t.Fatal(err)
	}
	out := testutil.CaptureStdout(func() {
		_, srv := testutil.SetupTestServer(t)
		defer testutil.TeardownTestServer(srv)
		f := testutil.SetupTestClient(srv, nil)
		defer f.Close()

		origSessions, _ := srv.ListSessions()

		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "switch", "test-kill")
		if err != nil {
			t.Fatal(err)
		}

		sessions, _ := srv.ListSessions()

		if len(origSessions)+2 != len(sessions) {
			t.Error("Should have one new session")
		}

		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "kill", "test-kill")
		if err != nil {
			t.Fatal(err)
		}

		sessions, _ = srv.ListSessions()
		if len(origSessions)+1 != len(sessions) {
			t.Error("Should have original sessions")
		}

	})
	if len(out) != 0 {
		t.Error("Should produce no output")
	}
}

func TestProjectTtl(t *testing.T) {
	t.Parallel()
	config := `
projects:
  test-ttl:
    directory: "/tmp"
  test-ttl2:
    directory: "/tmp"
`
	configFile := filepath.Join(t.TempDir(), "config.yml")
	err := os.WriteFile(configFile, []byte(config), 0755)
	if err != nil {
		t.Fatal(err)
	}
	testutil.CaptureStdout(func() {
		_, srv := testutil.SetupTestServer(t)
		defer testutil.TeardownTestServer(srv)
		f := testutil.SetupTestClient(srv, nil)
		defer f.Close()

		origSessions, _ := srv.ListSessions()

		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "switch", "test-ttl")
		if err != nil {
			t.Fatal(err)
		}
		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "switch", "test-ttl2")
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(7 * time.Second)
		sessions, _ := srv.ListSessions()
		if len(origSessions)+3 != len(sessions) {
			t.Error("Should have two new session + control")
		}

		// Should kill the original sessiopn
		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "--projectTtl", "1s", "list")
		if err != nil {
			t.Fatal(err)
		}

		sessions, _ = srv.ListSessions()
		if len(origSessions) != len(sessions) {
			t.Error("Should have killed the unattached sessions")
		}
	})
}
