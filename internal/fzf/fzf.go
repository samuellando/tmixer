package fzf

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/project"
)

func PickProject(ctx context.Context, config *config.Config, projects []*project.Project) (*project.Project, error) {
	type pickProjectEvent struct {
		Args         []string `json:"args"`
		Output       string   `json:"output"`
		ParsedOutput string   `json:"parsedOutput"`
		Errors       []string `json:"errors,omitempty"`
	}
	event := &pickProjectEvent{}
	finish := log.Track(ctx, "pickProjectEvent", event)
	defer finish()
	input := projects
	projects = make([]*project.Project, len(input))
	copy(projects, input)

	cmd := exec.Command("fzf", config.FzfFlags...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: false,
	}
	event.Args = cmd.Args

	ptmx, tty, err := pty.Open()
	if err != nil {
		err := fmt.Errorf("while opening pty for fzf: %w", err)
		event.Errors = append(event.Errors, err.Error())
		fmt.Println(err)
		return nil, err
	}
	defer ptmx.Close()
	defer tty.Close()

	inPipe, _ := cmd.StdinPipe()
	outPipe, _ := cmd.StdoutPipe()
	cmd.Stderr = tty

	err = cmd.Start()
	if err != nil {
		fmt.Print(">>>", err)
		return nil, err
	}

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	defer term.Restore(fd, oldState)
	go io.Copy(ptmx, os.Stdin)
	go io.Copy(os.Stdout, ptmx)

	result := make(chan error)
	go func() {
		err := DisplayProjects(ctx, projects, inPipe)
		inPipe.Close()
		if err != nil {
			event.Errors = append(event.Errors, err.Error())
		}
		result <- err
	}()

	resize := func() {
		ws, err := pty.GetsizeFull(os.Stdin)
		if err == nil {
			_ = pty.Setsize(ptmx, ws)
		}
	}
	resize()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			resize()
		}
	}()

	out, err := io.ReadAll(outPipe)

	err = cmd.Wait()
	if err != nil {
		err := fmt.Errorf("fzf command error: %w %s", err)
		event.Errors = append(event.Errors, err.Error())
		return nil, err
	}
	if err = <-result; err != nil {
		event.Errors = append(event.Errors, err.Error())
		return nil, err
	}

	event.Output = string(out)
	selected, err := parseOutput(string(out))
	if err != nil {
		event.Errors = append(event.Errors, err.Error())
		return nil, err
	}
	event.ParsedOutput = string(selected)
	fmt.Println(selected)
	for _, project := range projects {
		if project.Name == selected {
			return project, nil
		}
	}

	return nil, fmt.Errorf("No project selected")
}

func DisplayProjects(ctx context.Context, projects []*project.Project, w io.Writer) error {
	type displayProjectsEvent struct {
		Errors []string `json:"errors,omitempty"`
	}
	event := &displayProjectsEvent{}
	finish := log.Track(ctx, "displayProjectsEvent", event)
	defer finish()
	var sortError error
	sort.Slice(projects, func(i, j int) bool {
		if sortError == nil {
			res, err := compare(projects, i, j)
			if err == nil {
				return res
			} else {
				event.Errors = append(event.Errors, err.Error())
				sortError = err
			}
		}
		return projects[i].Name < projects[j].Name
	})
	if sortError != nil {
		err := fmt.Errorf("while sorting projects: %w", sortError)
		event.Errors = append(event.Errors, err.Error())
		return err
	}
	for _, project := range projects {
		info, err := display(project)
		if err != nil {
			err := fmt.Errorf("while displaying project: %w", err)
			event.Errors = append(event.Errors, err.Error())
			return err
		}
		io.WriteString(w, info+"\n")
	}
	return nil
}

func display(p *project.Project) (string, error) {
	icon := "\uf114"
	status, err := p.Status()
	if err != nil {
		return p.Name, err
	}
	if status >= project.PROJECT_STATUS_ACTIVE {
		icon = "\uf07b"
	}
	if status == project.PROJECT_STATUS_ATTACHED {
		icon = "\033[31m" + icon + "\033[0m"
	}
	return icon + " " + p.Name, nil
}

func parseOutput(out string) (string, error) {
	parts := strings.Split(out, " ")
	if len(parts) != 2 {
		return "", fmt.Errorf("output should have 2 parts")
	}
	return strings.TrimSpace(parts[1]), nil
}

func compare(projects []*project.Project, i, j int) (bool, error) {
	istatus, err := projects[i].Status()
	if err != nil {
		return false, err
	}
	jstatus, err := projects[j].Status()
	if err != nil {
		return false, err
	}
	if istatus == project.PROJECT_STATUS_ATTACHED {
		return true, nil
	}
	if istatus == jstatus {
		switch istatus {
		case project.PROJECT_STATUS_INACTIVE:
			return projects[i].Name < projects[j].Name, nil
		case project.PROJECT_STATUS_ACTIVE:
			ila, err := projects[i].LastActivity()
			if err != nil {
				return false, err
			}
			jla, err := projects[j].LastActivity()
			if err != nil {
				return false, err
			}
			return ila.After(*jla), nil
		}
	}
	return istatus > jstatus, nil
}
