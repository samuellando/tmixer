package project

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/tmux"
)

func TestProjectSwitch(t *testing.T) {
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		session, err := tc.project.Switch(ctx)
		if err != nil {
			t.Error(err)
		}
		if session == nil {
			t.Error("Should return a session")
		}
		status, err := tc.project.Status()
		if err != nil {
			t.Error(err)
		}
		if status != PROJECT_STATUS_ATTACHED {
			t.Error("Should attache to project")
		}
	})
}

func TestProjectSwitchCreatesAttachesSession(t *testing.T) {
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		session, err := tc.project.Switch(ctx)
		if err != nil {
			t.Error(err)
		}
		if session == nil {
			t.Error("Should have a session")
		}
		if tc.initialStatus() != PROJECT_STATUS_INACTIVE {
			if tc.session.Id != session.Id {
				t.Error("Should attach to existing session")
			}
		}
	})
}

func TestProjectSwitchWindowsAndPanes(t *testing.T) {
	maxAttempts := 60
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		res, err := tc.project.Switch(ctx)
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
