package project

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/tmux"
)

func TestProjectStart(t *testing.T) {
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		res, err := tc.project.Start(ctx)
		if err != nil {
			t.Error(err)
		}
		if res == nil {
			t.Error("Should not return a nil session")
		}
		if tc.session != nil && tc.session.Id != res.Id {
			t.Error("Session Id should match")
		}
		status, err := tc.project.Status()
		if err != nil {
			t.Error(err)
		}
		switch tc.initialStatus() {
		case PROJECT_STATUS_INACTIVE:
			if status != PROJECT_STATUS_ACTIVE {
				t.Error("Inactive project should be active now")
			}
		case PROJECT_STATUS_ACTIVE:
			if status != PROJECT_STATUS_ACTIVE {
				t.Error("Active project should still be active")
			}
		case PROJECT_STATUS_ATTACHED:
			if status != PROJECT_STATUS_ATTACHED {
				t.Error("Attached project should still be attached")
			}
		}
	})
}

func TestProjectStartWindowsAndPanes(t *testing.T) {
	maxAttempts := 60
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		res, err := tc.project.Start(ctx)
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
