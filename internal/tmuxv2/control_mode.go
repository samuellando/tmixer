package tmuxv2

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

var (
	controlModeCmd       *exec.Cmd
	controlModeCmdStdIn  io.WriteCloser
	controlModeCmdStdOut io.ReadCloser
	controlModeStdIn     *bufio.Writer
	controlModeStdOut    *bufio.Reader
)

func startControlMode() error {
	if controlModeCmd != nil {
		return fmt.Errorf("Call close on the previous session first!")
	}
	controlModeCmd = exec.Command("tmux", "-C")
	err := controlModeCmd.Start()
	if err != nil {
		closeControlMode()
		return err
	}
	stdIn, err := controlModeCmd.StdinPipe()
	if err != nil {
		closeControlMode()
		return err
	}
	controlModeStdIn = bufio.NewWriter(stdIn)
	stdOut, err := controlModeCmd.StdoutPipe()
	if err != nil {
		closeControlMode()
		return err
	}
	controlModeStdOut = bufio.NewReader(stdOut)
	return nil
}

func closeControlMode() error {
	err := controlModeCmd.Process.Kill()
	if err != nil {
		return err
	}
	err = controlModeCmdStdIn.Close()
	if err != nil {
		return err
	}
	err = controlModeCmdStdIn.Close()
	if err != nil {
		return err
	}
	err = controlModeCmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

func runControlModeCommand(cmd string) ([]string, error) {
	_, err := controlModeStdIn.WriteString(cmd + "\n")
	if err != nil {
		return nil, err
	}
	readState := "before"
	out := make([]string, 0)
	for {
		outLine, err := controlModeStdOut.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(outLine, "%begin") {
			readState = "inside"
		}
		if strings.HasPrefix(outLine, "%end") {
			readState = "done"
			break
		}
		if strings.HasPrefix(outLine, "%error") {
			readState = "error"
			break
		}
		if readState == "inside" {
			out = append(out, outLine)
		}
	}
	if readState == "done" {
		return out, nil
	} else {
		return out, fmt.Errorf("Command returned an error")
	}
}
