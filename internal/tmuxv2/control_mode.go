package tmuxv2

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

var (
	controlModeCmd       *exec.Cmd
	controlModeCmdStdOut io.ReadCloser
	controlModeStdIn     io.WriteCloser
	controlModeStdOut    *bufio.Reader
)

const CONTROL_SESSION_NAME = "__tmixer_control__"

func StartControlMode() error {
	if controlModeCmd != nil {
		return fmt.Errorf("Call close on the previous client first!")
	}
	cmd := command("new-session")
	cmd = cmd.withTmuxFlag("-C").withSession(CONTROL_SESSION_NAME).withFlag("-A")
	controlModeCmd = cmd.getExecCmd()
	var err error
	controlModeStdIn, err = controlModeCmd.StdinPipe()
	controlModeCmdStdOut, err = controlModeCmd.StdoutPipe()
	if err != nil {
		return errors.Join(err, CloseControlMode())
	}
	controlModeStdOut = bufio.NewReader(controlModeCmdStdOut)
	err = controlModeCmd.Start()
	if err != nil {
		return errors.Join(err, CloseControlMode())
	}
	_, err = readMessage()
	if err != nil {
		return errors.Join(err, CloseControlMode())
	}
	return err
}

func CloseControlMode() error {
	if controlModeCmd == nil {
		return fmt.Errorf("controlMode already closed")
	}
	err := controlModeStdIn.Close()
	if err != nil {
		return fmt.Errorf("error when closing stdin %w", err)
	}
	err = controlModeCmd.Wait()
	if err != nil {
		return fmt.Errorf("error when waiting for command to exit %w", err)
	}
	controlModeCmd = nil
	controlModeStdIn = nil
	controlModeStdOut = nil
	controlModeCmdStdOut = nil
	cmd := command("kill-session").withTargetSessionName(CONTROL_SESSION_NAME)
	_, err = cmd.run()
	if err != nil {
		return err
	}
	return nil
}

func sendControlModeCommand(c cmd) ([]string, error) {
	_, err := controlModeStdIn.Write([]byte(c.String() + "\n"))
	if err != nil {
		return nil, err
	}
	return readMessage()
}

func readMessage() ([]string, error) {
	readState := "before"
	out := make([]string, 0)
	for {
		outLine, err := controlModeStdOut.ReadString('\n')
		if err != nil {
			panic(err)
		}
		if strings.HasPrefix(outLine, "%begin") {
			readState = "inside"
			continue
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
			out = append(out, strings.TrimSpace(outLine))
		}
	}
	if readState == "done" {
		return out, nil
	} else {
		return out, fmt.Errorf("Command returned an error")
	}
}

func runCommandInControlModeIfStarted(c cmd) ([]string, error) {
	if controlModeCmd != nil {
		return sendControlModeCommand(c)
	}
	cmd := c.getExecCmd()
	out, err := cmd.CombinedOutput()
	lines := make([]string, 0)
	for line := range bytes.SplitSeq(out, []byte{'\n'}) {
		lines = append(lines, strings.TrimSpace(string(line)))
	}
	return lines, err
}

