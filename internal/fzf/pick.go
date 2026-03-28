package fzf

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
	"samuellando.com/tmixer/internal/config"
	"samuellando.com/tmixer/internal/log"
)

func Pick(ctx context.Context, config *config.Config, options []string) (*string, error) {
	logEvent := log.Track(ctx, "fzf")
	defer logEvent.Done()
	logEvent.Log("options", options)

	// We need to use the raw stdin for the fzf command
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			err := fmt.Errorf("while getting raw stdin: %w", err)
			logEvent.Error(err)
			return nil, err
		}
		defer func() {
			err = errors.Join(err, term.Restore(fd, oldState))
		}()
	}

	// Run the command in a ptx for consistency across envs
	cmd := exec.Command("fzf", config.FzfFlags...)
	logEvent.Log("args", cmd.Args)
	ptx, err := startPty(cmd, os.Stdin, os.Stdout)
	if err != nil {
		err := fmt.Errorf("while opening pty for fzf: %w", err)
		logEvent.Error(err)
		return nil, err
	}
	defer func() {
		err = errors.Join(err, ptx.Close())
	}()

	// Now pipe in the projects to fzf
	displayErrs := make(chan error)
	go func() {
		for _, o := range options {
			_, err := io.WriteString(ptx, o+"\n")
			if err != nil {
				displayErrs <- err
				return
			}
		}
		err = ptx.CloseInPipe()
		if err != nil {
			displayErrs <- err
			return
		}
		displayErrs <- nil
	}()

	// Wait for the command to exit
	out, err := io.ReadAll(ptx)
	err = cmd.Wait()
	if err != nil {
		if cmd.ProcessState.ExitCode() != 130 {
			err := fmt.Errorf("fzf command error: %w", err)
			logEvent.Error(err)
			return nil, err
		} else {
			return nil, nil
		}
	}
	if err = <-displayErrs; err != nil {
		logEvent.Error(err)
		return nil, err
	}

	// Return the selected project
	s := strings.TrimSpace(string(out))
	logEvent.Log("output", s)
	return &s, nil
}
