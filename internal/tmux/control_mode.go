package tmux

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
)

type outputEvent struct {
	Lines []string
	Err   error
}

type ClientSessionChangeEvent struct {
	Session     SessionId
	SessionName string
	Client      ClientId
}

type controlModeClient struct {
	commandMu         sync.Mutex
	subscribersMu     sync.Mutex
	controlModeCmd    *exec.Cmd
	controlModeStdIn  io.WriteCloser
	controlModeStdOut *bufio.Reader
	outputEvents      chan *outputEvent
	subscribers       map[chan *ClientSessionChangeEvent]bool
}

func (s *Server) Sub() chan *ClientSessionChangeEvent {
	client := s.controlModeClient
	if s.controlModeClient == nil {
		log.Fatal("Need's control mode to subscribe")
	}
	client.subscribersMu.Lock()
	defer client.subscribersMu.Unlock()
	ch := make(chan *ClientSessionChangeEvent)
	client.subscribers[ch] = true
	return ch
}

func (s *Server) UnSub(ch chan *ClientSessionChangeEvent) {
	client := s.controlModeClient
	if s.controlModeClient == nil {
		log.Fatal("Need's control mode to unsubscribe")
	}
	client.subscribersMu.Lock()
	defer client.subscribersMu.Unlock()
	delete(client.subscribers, ch)
	close(ch)
}

func (client *controlModeClient) transmit(e *ClientSessionChangeEvent) {
	client.subscribersMu.Lock()
	defer client.subscribersMu.Unlock()
	for s := range client.subscribers {
		s <- e
	}
}

const CONTROL_SESSION_NAME = "__tmixer_control__"

func (srv *Server) StartControlMode() error {
	// Create the control mode session object
	client := controlModeClient{
		outputEvents: make(chan *outputEvent),
		subscribers:  make(map[chan *ClientSessionChangeEvent]bool),
	}
	// Setup the actual command to run
	cmd := srv.command("new-session")
	cmd = cmd.withTmuxFlag("-C").withFlag("-A").withSession(CONTROL_SESSION_NAME)
	// Escape and setup stdin and stdout
	client.controlModeCmd = cmd.getExecCmd()

	// Set all th fields in the model object
	srv.controlModeClient = &client
	rollback := func(err error) error {
		srv.controlModeClient = nil
		return err
	}

	var err error
	client.controlModeStdIn, err = client.controlModeCmd.StdinPipe()
	if err != nil {
		return rollback(err)
	}
	stdout, err := client.controlModeCmd.StdoutPipe()
	if err != nil {
		return rollback(err)
	}
	client.controlModeStdOut = bufio.NewReader(stdout)

	// Run the command in background
	err = client.controlModeCmd.Start()
	if err != nil {
		return rollback(err)
	}
	go client.processOutput()
	_, err = client.readMessage()
	if err != nil {
		return rollback(err)
	}
	return nil
}

func (srv *Server) StopControlMode() error {
	if srv.controlModeClient == nil {
		return fmt.Errorf("CONTROL MODE ALREADY CLOSED")
	}
	defer func() {
		srv.controlModeClient = nil
	}()

	err := srv.controlModeClient.controlModeStdIn.Close()
	if err != nil {
		err = fmt.Errorf("when closing stdin %w", err)
		return err
	}
	_, _ = srv.controlModeClient.readMessage()
	err = srv.controlModeClient.controlModeCmd.Wait()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		err = fmt.Errorf("when waiting for exit %w", err)
		return err
	}
	// Finally clean up the session if we can
	return nil
}

func (srv *Server) runCommandInControlModeIfStarted(c cmd) ([]string, error) {
	if srv.controlModeClient != nil {
		return srv.controlModeClient.sendCommand(c)
	}
	return srv.runCommandOutsideControlMode(c)
}

func (srv *Server) runCommandOutsideControlMode(c cmd) ([]string, error) {
	cmd := c.getExecCmd()
	out, err := cmd.CombinedOutput()

	lines := make([]string, 0)
	for line := range bytes.SplitSeq(out, []byte{'\n'}) {
		l := string(line)
		if len(l) > 0 {
			lines = append(lines, l)
		}
	}
	if err != nil {
		err := fmt.Errorf("command returned error: %w with output \"%s\"", err, strings.Join(lines, "\n"))
		return lines, err
	}
	return lines, nil
}

func (client *controlModeClient) sendCommand(c cmd) ([]string, error) {
	client.commandMu.Lock()
	defer client.commandMu.Unlock()
	_, err := client.controlModeStdIn.Write([]byte(c.String() + "\n"))
	if err != nil {
		err = fmt.Errorf("failed to write to stdin: %w", err)
		return nil, err
	}
	out, err := client.readMessage()
	return out, err
}

func (client *controlModeClient) readMessage() ([]string, error) {
	e := <-client.outputEvents
	return e.Lines, e.Err
}

func (client *controlModeClient) processOutput() {
	for {
		readState := "before"
		out := make([]string, 0)
		for {
			outLine, err := client.controlModeStdOut.ReadString('\n')
			if err == io.EOF {
				return
			}
			if err != nil {
				client.outputEvents <- &outputEvent{Err: err}
				return
			}
			if strings.HasPrefix(outLine, "%client-session-changed") {
				parts := strings.Split(strings.TrimSpace(outLine), " ")
				clientId, _ := parseClientId(parts[1])
				sessionId, _ := parseSessionId(parts[2])
				sessionName := parts[3]
				if sessionName != CONTROL_SESSION_NAME {
					client.transmit(&ClientSessionChangeEvent{
						Client:      clientId,
						Session:     sessionId,
						SessionName: sessionName,
					})
				}
			}
			if strings.HasPrefix(outLine, "%begin") {
				readState = "inside"
				continue
			}
			if strings.HasPrefix(outLine, "%end") {
				readState = "done"
				break
			}
			if strings.HasPrefix(outLine, "%exit") {
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
		switch readState {
		case "done":
			client.outputEvents <- &outputEvent{Lines: out, Err: nil}
		case "error":
			err := fmt.Errorf("command returned error output \"%s\"", strings.Join(out, "\n"))
			client.outputEvents <- &outputEvent{Lines: out, Err: err}
		default:
			err := fmt.Errorf("CRITICAL READ ERROR")
			client.outputEvents <- &outputEvent{Lines: nil, Err: err}
		}
	}
}
