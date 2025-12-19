package tmuxv2

import (
	"fmt"
	"os/exec"
	"strings"
)

type cmd struct {
	tmuxFlags [][]string
	command   string
	flags     [][]string
	arguments []string
	server *Server
}


func (srv *Server) command(c string) cmd {
	cmd := cmd{
		tmuxFlags: make([][]string, 0),
		command:   c,
		flags:     make([][]string, 0),
		arguments: make([]string, 0),
		server: srv,
	}
	if srv.SocketPath != "" {
		return cmd.withTmuxFlag("-S", srv.SocketPath)
	} else {
		return cmd
	}
}

func (c cmd) run() ([]string, error) {
	return c.server.runCommandInControlModeIfStarted(c)
}

func (c cmd) withFlag(flagValues ...string) cmd {
	c.flags = append(c.flags, flagValues)
	return c
}

func (c cmd) withTmuxFlag(flagValues ...string) cmd {
	c.tmuxFlags = append(c.tmuxFlags, flagValues)
	return c
}

func (c cmd) withArgument(arg string) cmd {
	c.arguments = append(c.arguments, arg)
	return c
}

func (c cmd) withTargetSession(t *Session) cmd {
	return c.withFlag("-t", string(t.Id))
}

func (c cmd) withTargetWindow(t *Window) cmd {
	return c.withFlag("-t", string(t.Id))
}

func (c cmd) withTargetPane(t *Pane) cmd {
	return c.withFlag("-t", string(t.Id))
}

func (c cmd) withTargetSessionName(name string) cmd {
	return c.withFlag("-t", "="+name+":")
}

func (c cmd) withSession(name string) cmd {
	return c.withFlag("-s", name)
}

func (c cmd) withWindow(t *Window) cmd {
	return c.withFlag("-s", string(t.Id))
}

func (c cmd) withName(n string) cmd {
	return c.withFlag("-n", n)
}

func (c cmd) withWorkingDirectory(path string) cmd {
	return c.withFlag("-c", path)
}

func (c cmd) withFormat(format string) cmd {
	return c.withFlag("-F", fmt.Sprintf("%s", format))
}

func (c cmd) withFilter(filter string) cmd {
	return c.withFlag("-f", filter)
}

func (c cmd) detached() cmd {
	return c.withFlag("-d")
}

func (c cmd) print() cmd {
	return c.withFlag("-P")
}

func (c cmd) String() string {
	return strings.Join(c.internalArguments(), " ")
}

func (c cmd) getExecCmd() *exec.Cmd {
	return exec.Command("tmux", c.tmuxArguments()...)
}

func (c cmd) tmuxArguments() []string {
	args := make([]string, 0)
	for _, flag := range c.tmuxFlags {
		args = append(args, flag...)
	}
	args = append(args, c.command)
	for _, flag := range c.flags {
		args = append(args, flag...)
	}
	args = append(args, c.arguments...)
	return args
}

func (c cmd) internalArguments() []string {
	args := make([]string, 0)
	args = append(args, c.command)
	for _, flag := range c.flags {
		args = append(args, flag[0])
		if len(flag) > 1 {
			args = append(args, fmt.Sprintf("\"%s\"", flag[1]))
		}
	}
	args = append(args, c.arguments...)
	return args
}
