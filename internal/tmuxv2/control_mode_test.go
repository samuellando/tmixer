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
	out, err := c.run()
	var client *exec.Cmd
	if err != nil {
		fmt.Println("Starting tmux server")
		c := command("new").withTmuxFlag("-C")
		client = c.getExecCmd()
		err := client.Start()
		if err != nil {
			panic(err)
		}
	} else if strings.Contains(strings.Join(out, "\n"), CONTROL_SESSION_NAME) {
		panic("CONTROL_SESSION_NAME detected")
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
	for range 10 {
		err := StartControlMode()
		if err != nil {
			t.Fatal(err)
		}
		err = CloseControlMode()
		if err != nil {
			t.Fatal(err)
		}
	}
	c := command("list-sessions")
	out, err := c.run()
	if err != nil {
		t.Fatal(err)
	} 
	if strings.Contains(strings.Join(out, "\n"), CONTROL_SESSION_NAME) {
		t.Fatal("CONTROL_SESSION_NAME detected")
	}
}

func TestRunCommand(t *testing.T) {
	StartControlMode()
	defer CloseControlMode()
	c := command("list-sessions")
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
	StartControlMode()
	defer CloseControlMode()
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
		if strings.Contains(strings.Join(out, " "), CONTROL_SESSION_NAME) {
			b.Fail()
		}
    }
}

func BenchmarkControlModeRun(b *testing.B) {
	StartControlMode()
	defer CloseControlMode()
	c := command("list-sessions")
    for b.Loop() {
		out, err := sendControlModeCommand(c)
		if err != nil {
			b.Fatal(err)
		}
		if !strings.Contains(strings.Join(out, " "), CONTROL_SESSION_NAME) {
			b.Fail()
		}
    }
}
