package tmuxv2

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"fmt"
)

func TestMain(m *testing.M) {
	// Setup
	c := command("list-sessions")
	out, err := c.Run()
	var client *exec.Cmd
	if err != nil {
		fmt.Println("Starting tmux server")
		c := command("new").withTmuxFlag("-C")
		_, err := c.Run()
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
		_, err := c.Run()
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
		err := startControlMode()
		if err != nil {
			t.Fatal(err)
		}
		err = closeControlMode()
		if err != nil {
			t.Fatal(err)
		}
	}
	c := command("list-sessions")
	out, err := c.Run()
	if err != nil {
		t.Fatal(err)
	} 
	if strings.Contains(strings.Join(out, "\n"), CONTROL_SESSION_NAME) {
		t.Fatal("CONTROL_SESSION_NAME detected")
	}
}

func TestSendCommand(t *testing.T) {
	startControlMode()
	defer closeControlMode()
	c := command("list-sessions")
	lines, err := sendControlModeCommand(c)
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

func BenchmarkRun(b *testing.B) {
	startControlMode()
	defer closeControlMode()
	c := command("list-sessions")
    for b.Loop() {
		out, err := c.Run()
		if err != nil {
			b.Fatal(err)
		}
		if !strings.Contains(strings.Join(out, " "), CONTROL_SESSION_NAME) {
			b.Fail()
		}
    }
}

func BenchmarkControlModeRun(b *testing.B) {
	startControlMode()
	defer closeControlMode()
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
