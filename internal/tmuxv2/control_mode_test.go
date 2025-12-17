package tmuxv2

import (
	"strings"
	"testing"
	"time"
)

func TestStartStopControlMode(t *testing.T) {
	tmux := setupTestServer(t)
	defer teardownTestServer(tmux)
	for range 10 {
		err := tmux.StartControlMode()
		if err != nil {
			t.Fatal(err)
		}
		err = tmux.StopControlMode()
		if err != nil {
			t.Fatal(err)
		}
	}
	c := tmux.command("list-sessions")
	out, err := c.run()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(out, "\n"), CONTROL_SESSION_NAME) {
		t.Fatal("CONTROL_SESSION_NAME detected")
	}
}

func TestStartStopControlModeSimul(t *testing.T) {
	tmux := setupTestServer(t)
	defer teardownTestServer(tmux)
	n := 10
	servers := make([]*Server, n)
	var err error
	for i := range n {
		servers[i] = Tmux(tmux.socketPath)
		servers[i].StartControlMode()
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := range n {
		err = servers[i].StopControlMode()
		if err != nil {
			t.Fatal(err)
		}
	}
	c := tmux.command("list-sessions")
	out, err := c.run()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(out, "\n"), CONTROL_SESSION_NAME) {
		t.Fatal("CONTROL_SESSION_NAME detected")
	}
}

func TestRunCommand(t *testing.T) {
	tmux := setupTestServer(t)
	defer teardownTestServer(tmux)
	tmux.StartControlMode()
	defer tmux.StopControlMode()
	c := tmux.command("list-sessions")
	lines, err := c.run()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, line := range lines {
		if strings.Contains(line, CONTROL_SESSION_NAME) {
			found = true
		}
	}
	if !found {
		t.Fatal("Control session should be listed")
	}
}

func TestUsesControlMode(t *testing.T) {
	tmux := setupTestServer(t)
	defer teardownTestServer(tmux)
	// Since control mode is so much fater (~7000x) this is a reliable test
	c := tmux.command("list-sessions")
	n := 100
	start := time.Now()
	for range n {
		_, err := c.run()
		if err != nil {
			t.Fatal(err)
		}
	}
	psTime := time.Since(start)
	tmux.StartControlMode()
	defer tmux.StopControlMode()
	start = time.Now()
	for range n {
		_, err := c.run()
		if err != nil {
			t.Fatal(err)
		}
	}
	cmTime := time.Since(start)
	if !(20*cmTime <= psTime) {
		t.Fatal("Should be slower")
	}
}

func BenchmarkRun(b *testing.B) {
	tmux := setupTestServer(b)
	defer teardownTestServer(tmux)
	c := tmux.command("list-sessions")
	for b.Loop() {
		out, err := c.run()
		if err != nil {
			b.Fatal(err)
		}
		if strings.Contains(strings.Join(out, " "), CONTROL_SESSION_NAME) {
			b.Fail()
		}
	}
}

func BenchmarkControlModeRun(b *testing.B) {
	tmux := setupTestServer(b)
	defer teardownTestServer(tmux)
	tmux.StartControlMode()
	defer tmux.StopControlMode()
	c := tmux.command("list-sessions")
	for b.Loop() {
		out, err := c.run()
		if err != nil {
			b.Fatal(err)
		}
		if !strings.Contains(strings.Join(out, " "), CONTROL_SESSION_NAME) {
			b.Fail()
		}
	}
}
