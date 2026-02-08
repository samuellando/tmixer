package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/tmux"
)

func TestProjectReset(t *testing.T) {
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		initialSessions, err := srv.ListSessions()
		if err != nil {
			t.Error(err)
		}
		initialSession, _ := tc.project.Session()
		res, cleanup, reserr := tc.project.Reset(ctx)
		status, err := tc.project.Status()
		if err != nil {
			t.Error(err)
		}
		switch tc.initialStatus() {
		case PROJECT_STATUS_INACTIVE:
			if !errors.Is(reserr, ErrSessionNotFound) {
				t.Error("Inactive project should return not found error")
			}
			if res != nil {
				t.Error("Should not return a sesison for inactive projects")
			}
			err = cleanup()
			if err != nil {
				t.Error(err)
			}
		case PROJECT_STATUS_ATTACHED:
			if reserr != nil {
				t.Error(reserr)
			}
			if status != tc.initialStatus() {
				t.Error("Status should match original")
			}
			if res.Id == tc.session.Id {
				t.Error("Status should not match original")
			}
			if !srv.HasSession(initialSession) {
				t.Error("Original session should still be active")
			}
			err = cleanup()
			if err != nil {
				t.Error(err)
			}
			if srv.HasSession(initialSession) {
				t.Error("Should kill the original session")
			}
		case PROJECT_STATUS_ACTIVE:
			if reserr != nil {
				t.Error(reserr)
			}
			if status != tc.initialStatus() {
				t.Error("Status should match original")
			}
			if res.Id == tc.session.Id {
				t.Error("Status should not match original")
			}
			err = cleanup()
			if err != nil {
				t.Error(err)
			}
		default:
			t.Error("Not implemented")
		}
		finalSessions, err := srv.ListSessions()
		if err != nil {
			t.Error(err)
		}
		if len(initialSessions) != len(finalSessions) {
			t.Error("Number of sessions before and after should match")
		}
	})
}

func TestProjectResetWindowsAndPanes(t *testing.T) {
	maxAttempts := 60
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		if tc.initialStatus() == PROJECT_STATUS_INACTIVE {
			return
		}
		res, cleanup, err := tc.project.Reset(ctx)
		cleanup()
		if err != nil {
			t.Error(err)
		}

		ticker := time.Tick(time.Second)

	tickLoop:
		for i := 0; ; i++ {
			<-ticker
			if strings.Contains(tc.project.Name, "windows") {
				windows, err := res.Windows()
				if err != nil {
					t.Error(err)
				}
				if len(windows) != 3 {
					if i == maxAttempts {
						t.Error("Should have 3 windows")
						break tickLoop
					} else {
						continue tickLoop
					}
				}
				for i := range 3 {
					panes, err := windows[i].Panes()
					if err != nil {
						t.Error(err)
					}
					for j := range pane_counts[i] {
						expected := fmt.Sprintf("%d", primes[i]*primes[j])
						out, err := panes[j].Capture()
						if err != nil {
							t.Error(err)
						}
						allOut := strings.Join(out, "")
						if !strings.Contains(allOut, expected) {
							if i == maxAttempts {
								t.Errorf("Missing output %s in %s", expected, allOut)
								break tickLoop
							} else {
								continue tickLoop
							}
						}
					}
				}
			} else {
				windows, err := res.Windows()
				if err != nil {
					t.Error(err)
				}
				if len(windows) != 1 {
					if i == maxAttempts {
						t.Errorf("Should have 1 (default) window got %d", len(windows))
						break tickLoop
					} else {
						continue tickLoop
					}
				}
			}
			break tickLoop
		}
	})
}

func TestProjectResetCommands(t *testing.T) {
	maxAttempts := 60
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		if tc.initialStatus() == PROJECT_STATUS_INACTIVE {
			return
		}
		_, cleanup, err := tc.project.Reset(ctx)
		cleanup()
		if err != nil {
			t.Error(err)
		}
		ticker := time.Tick(time.Second)
	tickloop:
		for i := 1; ; i++ {
			<-ticker
			ls, err := os.ReadDir(tc.project.Config.Directory)
			if err != nil {
				if i == maxAttempts {
					t.Error(err)
					break tickloop
				}
				continue tickloop
			}
			if strings.Contains(tc.project.Name, "switch") && tc.initialStatus() == PROJECT_STATUS_ATTACHED {
				if len(ls) == len(switchCommands) {
					break tickloop
				} else if i == maxAttempts {
					t.Errorf("Expected %d files from switch commands got %d", len(switchCommands), len(ls))
					break tickloop
				}
			} else {
				if len(ls) == 0 {
					break tickloop
				} else if i == maxAttempts {
					t.Errorf("Expected 0 files from switch commands got %d in %s", len(ls), tc.project.Config.Directory)
					break tickloop
				}
			}
		}
	})
}
