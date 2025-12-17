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
	tmux := Tmux()
	c := tmux.command("test")
	assertString(c.String(), "test", t)
}

func TestWithFlag(t *testing.T) {
	tmux := Tmux()
	c := tmux.command("test").withFlag("-s", "Hello")
	assertString(c.String(), "test -s Hello", t)
	c = tmux.command("test").withFlag("-A")
	assertString(c.String(), "test -A", t)
}

func TestWithTmuxFlag(t *testing.T) {
	tmux := Tmux()
	c := tmux.command("test").withTmuxFlag("-s", "Hello")
	assertString(strings.Join(c.tmuxArguments(), " "), "-s Hello test", t)
	c = tmux.command("test").withTmuxFlag("-B")
	assertString(strings.Join(c.tmuxArguments(), " "), "-B test", t)
}

func TestWithArgument(t *testing.T) {
	tmux := Tmux()
	c := tmux.command("test").withArgument("Hello")
	assertString(c.String(), "test Hello", t)
}

func TestCombination(t *testing.T) {
	tmux := Tmux()
	c := tmux.command("test").withArgument("hello").withTmuxFlag("-T").withFlag("-s")
	assertString(strings.Join(c.tmuxArguments(), " "), "-T test -s hello", t)

}

// Test each helper

func TestWithTargetSession(t *testing.T) {
	tmux := Tmux()
	session := Session{Id: "$19"}
	c := tmux.command("test").withTargetSession(&session)
	assertString(c.String(), "test -t $19", t)
}

func TestWithTargetSessionName(t *testing.T) {
	tmux := Tmux()
	c := tmux.command("test").withTargetSessionName("__hello__")
	assertString(c.String(), "test -t =__hello__:", t)
}

func TestWithSession(t *testing.T) {
	tmux := Tmux()
	c := tmux.command("test").withSession("__hello__")
	assertString(c.String(), "test -s __hello__", t)
}

func TestWithWorkingDirectory(t *testing.T) {
	tmux := Tmux()
	c := tmux.command("test").withWorkingDirectory("/home/sam")
	assertString(c.String(), "test -c /home/sam", t)
}

func TestWithFormat(t *testing.T) {
	tmux := Tmux()
	c := tmux.command("test").withFormat("#{session_name}")
	assertString(c.String(), "test -F #{session_name}", t)
}

func TestWithFilter(t *testing.T) {
	tmux := Tmux()
	c := tmux.command("test").withFilter("#{session_name}")
	assertString(c.String(), "test -f #{session_name}", t)
}

func TestDetached(t *testing.T) {
	tmux := Tmux()
	c := tmux.command("test").detached()
	assertString(c.String(), "test -d", t)
}

func TestPrint(t *testing.T) {
	tmux := Tmux()
	c := tmux.command("test").print()
	assertString(c.String(), "test -P", t)
}

// Test output comands

func TestRun(t *testing.T) {
	tmux := setupTestServer(t)
	defer teardownTestServer(tmux)
	c := tmux.command("test").withTmuxFlag("-V")
	out, err := c.run()
	if err != nil {
		t.Fatal("Should not return an error")
	}
	if !strings.HasPrefix(out[0], "tmux") {
		t.Fatalf("Should return the output %s", out)
	}
}

func TestRunBadCommand(t *testing.T) {
	tmux := setupTestServer(t)
	defer teardownTestServer(tmux)
	tmux.command("new").detached().run()
	c := tmux.command("test")
	out, err := c.run()
	if err == nil {
		t.Fatal("Should return an error")
	}
	if !strings.HasPrefix(out[0], "unknown") {
		t.Fatalf("Should return the output %s", out)
	}
}

func TestStart(t *testing.T) {
	tmux := setupTestServer(t)
	defer teardownTestServer(tmux)
	c := tmux.command("test").withTmuxFlag("-V")
	cmd := c.getExecCmd()
	stdout, _ := cmd.StdoutPipe()
	buff := make([]byte, 100)
	cmd.Start()
	stdout.Read(buff)
	cmd.Wait()
	if !strings.HasPrefix(string(buff), "tmux") {
		t.Fatalf("Should return the output %s", buff)
	}
}
