package tmuxv2

import (
	"strings"
	"testing"
)

func assertString(a, b string, t *testing.T) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}

// Test general

func TestBase(t *testing.T) {
	c := command("test")
	assertString(c.String(), "test", t)
}

func TestWithFlag(t *testing.T) {
	c := command("test").withFlag("-s", "Hello")
	assertString(c.String(), "test -s Hello", t)
	c = command("test").withFlag("-A")
	assertString(c.String(), "test -A", t)
}

func TestWithTmuxFlag(t *testing.T) {
	c := command("test").withTmuxFlag("-s", "Hello")
	assertString(c.String(), "-s Hello test", t)
	c = command("test").withTmuxFlag("-B")
	assertString(c.String(), "-B test", t)
}

func TestWithArgument(t *testing.T) {
	c := command("test").withArgument("Hello")
	assertString(c.String(), "test Hello", t)
}

func TestCombination(t *testing.T) {
	c := command("test").withArgument("hello").withTmuxFlag("-T").withFlag("-s")
	assertString(c.String(), "-T test -s hello", t)

}

// Test each helper

func TestWithTargetClient(t *testing.T) {
	client := Client{Id: "/pty/25"}
	c := command("test").withTargetClient(&client)
	assertString(c.String(), "test -t /pty/25", t)
}

func TestWithTargetSession(t *testing.T) {
	session := Session{Id: "$19"}
	c := command("test").withTargetSession(&session)
	assertString(c.String(), "test -t $19", t)
}

func TestWithTargetSessionName(t *testing.T) {
	c := command("test").withTargetSessionName("__hello__")
	assertString(c.String(), "test -t =__hello__", t)
}

func TestWithSession(t *testing.T) {
	c := command("test").withSession("__hello__")
	assertString(c.String(), "test -s __hello__", t)
}

func TestWithWorkingDirectory(t *testing.T) {
	c := command("test").withWorkingDirectory("/home/sam")
	assertString(c.String(), "test -c /home/sam", t)
}

func TestWithFormat(t *testing.T) {
	c := command("test").withFormat("#{session_name}")
	assertString(c.String(), "test -F #{session_name}", t)
}

func TestDetached(t *testing.T) {
	c := command("test").detached()
	assertString(c.String(), "test -d", t)
}

func TestPrint(t *testing.T) {
	c := command("test").print()
	assertString(c.String(), "test -P", t)
}

// Test output comands

func TestRun(t *testing.T) {
	c := command("test").withTmuxFlag("-V")
	out, err := c.Run()
	if err != nil {
		t.Fatal("Should not return an error")
	}
	if !strings.HasPrefix(out[0], "tmux") {
		t.Fatal("Should return the output")
	}
}

func TestRunBadCommand(t *testing.T) {
	c := command("test")
	out, err := c.Run()
	if err == nil {
		t.Fatal("Should return an error")
	}
	if !strings.HasPrefix(out[0], "unknown") {
		t.Fatal("Should return the output")
	}
}

func TestStart(t *testing.T) {
	c := command("test").withTmuxFlag("-V")
	cmd := c.getExecCmd()
	stdout, _ := cmd.StdoutPipe()
	buff := make([]byte, 100)
	cmd.Start()
	stdout.Read(buff)
	cmd.Wait()
	if !strings.HasPrefix(string(buff), "tmux") {
		t.Fatal("Should return the output")
	}
}
