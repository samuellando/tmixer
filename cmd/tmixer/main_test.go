package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/log/rotation"
	"samuellando.com/tmixer/internal/testutil"
)

func TestLogs(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(home, ".local/state/tmixer/logs")
	testutil.CaptureStdout(func() {
		f, err := rotation.RotateLogFile(logPath, 24*time.Hour)
		if err != nil {
			t.Error(err)
		}
		logFile := f.Name()
		stat, err := f.Stat()
		origSize := stat.Size()
		if err != nil {
			t.Fatal(err)
		}
		err = f.Close()
		if err != nil {
			t.Error(err)
		}
		socketFile := filepath.Join(t.TempDir(), "tmux.sock")
		err = run("-S", socketFile, "--help")
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
		socketFile := filepath.Join(t.TempDir(), "tmux.sock")
		err := run("-S", socketFile, "--help", "--logFile", logFile)
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
  - name: test-load-config
    directory: "/tmp"
`
	configFile := filepath.Join(t.TempDir(), "config.yml")
	err := os.WriteFile(configFile, []byte(config), 0755)
	if err != nil {
		t.Fatal(err)
	}
	out := testutil.CaptureStdout(func() {
		socketFile := filepath.Join(t.TempDir(), "tmux.sock")
		err = run("-S", socketFile, "-c", configFile, "list")
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
  - name: test-list1
    directory: "/tmp"
  - name: test-list2
    directory: "/tmp"
`
	configFile := filepath.Join(t.TempDir(), "config.yml")
	err := os.WriteFile(configFile, []byte(config), 0755)
	if err != nil {
		t.Fatal(err)
	}
	testutil.GoldenTest(t, testutil.CaptureStdout(func() {
		socketFile := filepath.Join(t.TempDir(), "tmux.sock")
		err = run("-S", socketFile, "-c", configFile, "--combineProjects", "false", "list")
		if err != nil {
			t.Fatal(err)
		}
	}))
}

func TestSwitch(t *testing.T) {
	config := `
projects:
  - name: test-switch
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
		defer func() {
			err = f.Close()
			if err != nil {
				t.Error(err)
			}
		}()

		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "switch", "test-switch")
		if err != nil {
			t.Fatal(err)
		}
		clt, _ := srv.ActiveClient()
		session, _ := clt.Session()
		name, _ := session.Name()
		if name != "test-switch" {
			t.Error("Expected session name to be test-switch")
		}
	})
	if len(out) != 0 {
		t.Error("Should produce no output")
	}
}

func TestReset(t *testing.T) {
	config := `
projects:
  - name: test-reset
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
		defer func() {
			err = f.Close()
			if err != nil {
				t.Error(err)
			}
		}()

		clt, _ := srv.ActiveClient()

		origSessions, _ := srv.ListSessions()

		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "switch", "test-reset")
		if err != nil {
			t.Fatal(err)
		}

		sessions, _ := srv.ListSessions()
		if len(sessions) != len(origSessions)+2 {
			t.Error("Should have one new session + control")
		}

		initial, _ := clt.Session()

		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "reset", "test-reset")
		if err != nil {
			t.Fatal(err)
		}

		sessions, err = srv.ListSessions()
		if err != nil {
			t.Error(err)
		}
		if len(sessions) != len(origSessions)+2 {
			t.Error("Should have one new session + control")
		}

		after, _ := clt.Session()
		if initial.Id != after.Id {
			t.Error("Should've kept the same session")
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
  - name: test-reset
    directory: "/tmp"
  - name: test-reset2
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
		defer func() {
			err = f.Close()
			if err != nil {
				t.Error(err)
			}
		}()

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

		sessions, _ := srv.ListSessions()
		if len(sessions) != len(origSessions)+3 {
			t.Error("Expected session count to increase by 3")
		}

		initial, _ := clt.Session()
		initialName, _ := initial.Name()
		if initialName != "test-reset2" {
			t.Error("Expected active session name to be test-reset2")
		}
		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "reset")
		if err != nil {
			t.Fatal(err)
		}

		sessions, err = srv.ListSessions()
		if err != nil {
			t.Error(err)
		}
		if len(sessions) != len(origSessions)+3 {
			t.Error("Expected session count to increase by 3")
		}
		after, _ := clt.Session()
		if initial.Id != after.Id {
			t.Error("Should've kept the same session")
		}
		afterName, _ := after.Name()
		if afterName != "test-reset2" {
			t.Error("Expected session name to remain test-reset2")
		}
	})
	if out != "" {
		t.Error("Should produce no output")
	}
}

func TestKill(t *testing.T) {
	config := `
projects:
  - name: test-kill
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
		defer func() {
			err = f.Close()
			if err != nil {
				t.Error(err)
			}
		}()

		origSessions, _ := srv.ListSessions()

		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "switch", "test-kill")
		if err != nil {
			t.Fatal(err)
		}

		sessions, _ := srv.ListSessions()

		if len(sessions) != len(origSessions)+2 {
			t.Error("Should have one new session")
		}

		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "kill", "test-kill")
		if err != nil {
			t.Fatal(err)
		}

		sessions, err = srv.ListSessions()
		if err != nil {
			t.Error(err)
		}
		if len(sessions) != len(origSessions)+1 {
			t.Error("Should have one new session")
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
  - name: test-ttl
    directory: "/tmp"
  - name: test-ttl2
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
		defer func() {
			err = f.Close()
			if err != nil {
				t.Error(err)
			}
		}()

		origSessions, _ := srv.ListSessions()

		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "switch", "test-ttl")
		if err != nil {
			t.Fatal(err)
		}
		err = run("-S", srv.SocketPath, "-c", configFile, "--combineProjects", "false", "switch", "test-ttl2")
		if err != nil {
			t.Fatal(err)
		}

		sessions, _ := srv.ListSessions()
		if len(sessions) != len(origSessions)+3 {
			t.Error("Expected session count to increase by 3")
		}

		time.Sleep(2 * time.Second)

		// Should kill the original session
		err = run(
			"-S", srv.SocketPath, "-c", configFile,
			"--combineProjects", "false",
			"--projectTtl", "1s",
			"list",
		)
		if err != nil {
			t.Fatal(err)
		}

		sessions, err = srv.ListSessions()
		if err != nil {
			t.Error(err)
		}
		if len(sessions) != len(origSessions) {
			t.Error("Should have killed the unattached session")
		}
	})
}
