package tmuxv2

import (
	"os"
	"time"
	"os/exec"
	"strings"
	"testing"
	"fmt"
)

func TestMain(m *testing.M) {
	// Setup
	c := command("list-sessions")
	_, err := c.run()
	var client *exec.Cmd
	if err != nil {
		fmt.Println("Starting tmux server")
		c := command("new").withTmuxFlag("-C")
		client = c.getExecCmd()
		err := client.Start()
		if err != nil {
			panic(err)
		}
	} 
	// Run tests
	code := m.Run()
	// cleanup server
	if client != nil {
		fmt.Println("Stoping tmux server")
		c := command("kill-server")
		_, err := c.run()
		if err != nil {
			panic(err)
		}
		client.Wait()
	}
	// Exit with the proper code
	os.Exit(code)
}

func TestStartStopControlMode(t *testing.T) {
	session_name := "__test_tmixer_control_mode_st__"
	for range 10 {
		client, err := StartControlMode(session_name)
		if err != nil {
			t.Fatal(err)
		}
		err = client.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	c := command("list-sessions")
	out, err := c.run()
	if err != nil {
		t.Fatal(err)
	} 
	if strings.Contains(strings.Join(out, "\n"), session_name) {
		t.Fatal("CONTROL_SESSION_NAME detected")
	}
}

func TestStartStopControlModeSimul(t *testing.T) {
	session_name := "__test_tmixer_control_mode_simul__"
	n := 10
	clients := make([]*controlModeClient, n)
	var err error
	for i := range n {
		clients[i], err = StartControlMode(session_name)
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := range n {
		err = clients[i].Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	c := command("list-sessions")
	out, err := c.run()
	if err != nil {
		t.Fatal(err)
	} 
	if strings.Contains(strings.Join(out, "\n"), session_name) {
		t.Fatal("CONTROL_SESSION_NAME detected")
	}
}

func TestRunCommand(t *testing.T) {
	session_name := "__test_tmixer_control_mode_run__"
	client, _ := StartControlMode(session_name)
	SetDefaultClient(client)
	defer client.Close()
	c := command("list-sessions")
	lines, err := c.run()
	if err != nil {
		t.Fatal(err)
	} 
	found := false
	for _, line := range lines {
		if strings.Contains(line, session_name) {
			found = true
		}
	}
	if !found {
		t.Fatal("Control session should be listed")
	}
}

func TestUsesControlMode(t *testing.T) {
	// Since control mode is so much fater (~7000x) this is a reliable test
	c := command("list-sessions")
	n := 100
	start := time.Now()
	for range n {
		_, err := c.run()
		if err != nil {
			t.Fatal(err)
		}
	}
	psTime := time.Since(start)
	client, _ := StartControlMode()
	SetDefaultClient(client)
	defer client.Close()
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
	c := command("list-sessions")
    for b.Loop() {
		out, err := c.run()
		if err != nil {
			b.Fatal(err)
		}
		if strings.Contains(strings.Join(out, " "), DEFAULT_CONTROL_SESSION_NAME) {
			b.Fail()
		}
    }
}

func BenchmarkControlModeRun(b *testing.B) {
	client, _ := StartControlMode()
	SetDefaultClient(client)
	defer client.Close()
	c := command("list-sessions")
    for b.Loop() {
		out, err := c.run()
		if err != nil {
			b.Fatal(err)
		}
		if !strings.Contains(strings.Join(out, " "), DEFAULT_CONTROL_SESSION_NAME) {
			b.Fail()
		}
    }
}
