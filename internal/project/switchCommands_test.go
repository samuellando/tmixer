package project

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"samuellando.com/tmixer/internal/tmux"
)

func TestProjectSwitchCommands(t *testing.T) {
	maxAttempts := 60
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		_, err := tc.project.Switch(ctx)
		if err != nil {
			t.Error(err)
		}

		ticker := time.Tick(time.Second)

	tickLoop:
		for i := 0; ; i++ {
			<-ticker
			ls, err := os.ReadDir(tc.project.Config.Directory)
			if err != nil {
				if i == maxAttempts {
					t.Error(err)
					break tickLoop
				}
				continue tickLoop
			}
			if strings.Contains(tc.project.Name, "switch") {
				if len(ls) == len(switchCommands) {
					break tickLoop
				} else if i == maxAttempts {
					t.Errorf("Expected %d files from switch commands got %d", len(switchCommands), len(ls))
					break tickLoop
				}
			} else {
				if len(ls) == 0 {
					break tickLoop
				} else if i == maxAttempts {
					t.Errorf("Expected 0 files from switch commands got %d", len(ls))
					break tickLoop
				}
			}
		}
	})
}

func TestProjectRunSwitchCommands(t *testing.T) {
	maxAttempts := 60
	runAllTestCases(t, func(t *testing.T, ctx context.Context, srv *tmux.Server, tc *projectTestCase) {
		err := tc.project.RunSwitchCommands(ctx)
		if tc.initialStatus() == PROJECT_STATUS_INACTIVE {
			if err != ErrSessionNotFound {
				t.Error("Should retrun sesison not found error for inactive projects")
			}
		} else {
			if err != nil {
				t.Error(err)
			}
			ticker := time.Tick(time.Second)
		tickLoop:
			for i := 0; ; i++ {
				<-ticker
				ls, err := os.ReadDir(tc.project.Config.Directory)
				if err != nil {
					if i == maxAttempts {
						t.Error(err)
						break tickLoop
					}
					continue tickLoop
				}
				if strings.Contains(tc.project.Name, "switch") {
					if len(ls) == len(switchCommands) {
						break tickLoop
					} else if i == maxAttempts {
						t.Errorf("Expected %d files from switch commands got %d", len(switchCommands), len(ls))
						break tickLoop
					}
				} else {
					if len(ls) == 0 {
						break tickLoop
					} else if i == maxAttempts {
						t.Errorf("Expected 0 files from switch commands got %d", len(ls))
						break tickLoop
					}
				}

			}
		}
	})
}
