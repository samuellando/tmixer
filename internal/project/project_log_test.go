package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"samuellando.com/tmixer/internal/log"
	"samuellando.com/tmixer/internal/testutil"
	"samuellando.com/tmixer/internal/tmux"
)

func runAllLogTestCases(t *testing.T, f func(ctx context.Context, srv *tmux.Server, out *bytes.Buffer, logger *log.Logger, tc *projectTestCase)) {
	t.Parallel()
	// Need to reset in between for each test case to avoid switches
	for _, tc := range getAllTestCases() {
		t.Run(tc.project.Name, func(t *testing.T) {
			t.Parallel()
			ctx, srv := testutil.SetupTestServer(t)
			defer testutil.TeardownTestServer(srv)
			err := srv.StartControlMode()
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := srv.StopControlMode(); err != nil {
					t.Error(err)
				}
			}()

			client := setupTestCase(t, ctx, tc, srv)
			defer teardownTestCase(t, client)

			ctx = context.Background()
			ctx, logger := log.New(ctx, nil)
			out := bytes.Buffer{}
			logger.AddSink(&out)

			tc.project.server = srv

			f(ctx, srv, &out, logger, tc)
		})
	}
}

func TestProjectStartLogs(t *testing.T) {
	runAllLogTestCases(t, func(ctx context.Context, srv *tmux.Server, out *bytes.Buffer, logger *log.Logger, tc *projectTestCase) {
		// Exec
		res, _ := tc.project.Start(ctx)
		// Grab the log event
		logger.Info(ctx)
		wideEvent := make(map[string]any)
		if err := json.Unmarshal(out.Bytes(), &wideEvent); err != nil {
			t.Error(err)
		}
		event := wideEvent["projectStartEvent"].(map[string]any)
		// Check each field
		// field: Name
		name := event["name"].(string)
		if name != tc.project.Name {
			t.Error("Event name does not match")
		}
		// field: InitialStatus
		eventStatus := event["initialStatus"].(float64)
		if eventStatus != float64(tc.initialStatus()) {
			t.Error("Event initialStatus does not match")
		}
		// field: SessionId
		sessionId := event["sessionId"].(string)
		if sessionId != string(res.Id) {
			t.Error("Event sessionId does not match")
		}
		// field: Errors
		if _, ok := event["errors"]; ok {
			t.Error("Should not have any errors")
		}
	})
}

func TestProjectKillLogs(t *testing.T) {
	runAllLogTestCases(t, func(ctx context.Context, srv *tmux.Server, out *bytes.Buffer, logger *log.Logger, tc *projectTestCase) {
		initialSession, _ := tc.project.Session()
		// Exec
		cleanup, err := tc.project.Kill(ctx)
		if err != nil && !errors.Is(err, ErrSessionNotFound) {
			t.Error(err)
		}
		if err := cleanup(); err != nil {
			t.Error(err)
		}
		// Grab the log event
		logger.Info(ctx)
		wideEvent := make(map[string]any)
		if err := json.Unmarshal(out.Bytes(), &wideEvent); err != nil {
			t.Error(err)
		}
		event := wideEvent["projectKillEvent"].(map[string]any)
		// Check all fields
		// field: Name
		name := event["name"].(string)
		if name != tc.project.Name {
			t.Error("Event name does not match")
		}
		// field: InitialStatus
		eventStatus := event["initialStatus"].(float64)
		if eventStatus != float64(tc.initialStatus()) {
			t.Errorf("Event initialStatus does not match for %s", tc.project.Name)
		}
		// field: Session Id
		if tc.initialStatus() >= PROJECT_STATUS_ACTIVE {
			sessionId := event["sessionId"].(string)
			if sessionId != string(initialSession.Id) {
				t.Error("Event sessionId does not match")
			}
		} else {
			if _, ok := event["sessionId"]; ok {
				t.Error("should not have session id")
			}
		}
		// field: TempSessionName
		if tc.initialStatus() == PROJECT_STATUS_ATTACHED {
			sessionId := event["tempSessionName"].(string)
			if _, err := uuid.Parse(sessionId); err != nil {
				t.Error("temp session if is invalid")
			}
		} else {
			if _, ok := event["tempSessionName"]; ok {
				t.Error("should not have a temp session for non attached project")
			}
		}
		// field: Errors
		if tc.initialStatus() >= PROJECT_STATUS_ACTIVE {
			if _, ok := event["errors"]; ok {
				t.Error("should not have errors")
			}
		} else {
			errs := event["errors"].([]any)
			if len(errs) != 1 {
				t.Error("should have one error")
			}
		}
	})
}

func TestProjectKillLogsSwitchDecision(t *testing.T) {
	runAllLogTestCases(t, func(ctx context.Context, srv *tmux.Server, out *bytes.Buffer, logger *log.Logger, tc *projectTestCase) {
		// Exec
		cleanup, err := tc.project.Kill(ctx)
		if err != nil && !errors.Is(err, ErrSessionNotFound) {
			t.Error(err)
		}
		if err := cleanup(); err != nil {
			t.Error(err)
		}
		// Grab the log event
		logger.Info(ctx)
		wideEvent := make(map[string]any)
		if err := json.Unmarshal(out.Bytes(), &wideEvent); err != nil {
			t.Error(err)
		}
		// Check each field
		if tc.initialStatus() == PROJECT_STATUS_ATTACHED {
			event := wideEvent["switchToBestProject"].(map[string]any)
			// field: Name
			name := event["name"].(string)
			if name != tc.project.Name {
				t.Error("Event name does not match")
			}
			// field: SortResult
			sortResult := event["sortResult"].([]any)
			if len(sortResult) != len(testConfig.Projects)+1 {
				t.Error("Event sort result not correct length")
			}
			// field: Selected
			selected := event["selected"].(string)
			if selected == "" {
				t.Error("Event selected empty")
			}
			// field: errors
			if _, ok := event["errors"]; ok {
				t.Error("Should have no errors")
			}
		} else {
			if _, ok := wideEvent["switchToBestProject"].(map[string]any); ok {
				t.Error("Should not log a switch event")
			}
		}
	})
}

func TestProjectSwitchLogs(t *testing.T) {
	runAllLogTestCases(t, func(ctx context.Context, srv *tmux.Server, out *bytes.Buffer, logger *log.Logger, tc *projectTestCase) {
		// Exec
		session, _ := tc.project.Switch(ctx)
		// Grab the log event
		logger.Info(ctx)
		wideEvent := make(map[string]any)
		if err := json.Unmarshal(out.Bytes(), &wideEvent); err != nil {
			t.Error(err)
		}
		event := wideEvent["projectSwitchEvent"].(map[string]any)
		// Check each field
		// field: Name
		name := event["name"].(string)
		if name != tc.project.Name {
			t.Error("Event name does not match")
		}
		// field: SessionId
		sessionId := event["sessionId"].(string)
		if sessionId != string(session.Id) {
			t.Error("Event sessionId does not match")
		}
		// field: ClientId
		clientId := event["clientId"].(string)
		if clientId == "" {
			t.Error("Event should contain a clientId")
		}
		// field: Errors
		if _, ok := event["errors"]; ok {
			t.Error("Should have no errors")
		}
	})
}

func TestProjectRunSwitchCommandsLogs(t *testing.T) {
	runAllLogTestCases(t, func(ctx context.Context, srv *tmux.Server, out *bytes.Buffer, logger *log.Logger, tc *projectTestCase) {
		// Exec
		err := tc.project.RunSwitchCommands(ctx)
		if err != nil && !errors.Is(err, ErrSessionNotFound) {
			t.Error(err)
		}
		// Grab the log event
		logger.Info(ctx)
		wideEvent := make(map[string]any)
		if err := json.Unmarshal(out.Bytes(), &wideEvent); err != nil {
			t.Error(err)
		}
		event := wideEvent["projectRunSwitchCommandsEvent"].(map[string]any)
		// Check each field
		// field: Name
		name := event["name"].(string)
		if name != tc.project.Name {
			t.Error("Event name does not match")
		}
		// field: SessionId
		if tc.initialStatus() >= PROJECT_STATUS_ACTIVE {
			sessionId := event["sessionId"].(string)
			session, _ := tc.project.Session()
			if sessionId != string(session.Id) {
				t.Error("Event sessionId does not match")
			}
		}
		// field: Commands
		if tc.initialStatus() >= PROJECT_STATUS_ACTIVE {
			if len(tc.project.Config.SwitchCommands) > 0 {
				commands := event["commands"].([]any)
				if len(commands) != len(tc.project.Config.SwitchCommands) {
					t.Error("Should list all switch commands")
				}
			} else {
				if _, ok := event["commands"]; ok {
					t.Error("Should not have commands")
				}
			}
		}
		// field: errors
		if tc.initialStatus() >= PROJECT_STATUS_ACTIVE {
			if _, ok := event["errors"]; ok {
				t.Error("Should not have errors")
			}
		} else {
			if errs, ok := event["errors"]; !ok {
				t.Error("Should have errors")
			} else if len(errs.([]any)) != 1 {
				t.Error("Should have 1 error")
			}
		}
	})
}

func TestProjectResetLogs(t *testing.T) {
	runAllLogTestCases(t, func(ctx context.Context, srv *tmux.Server, out *bytes.Buffer, logger *log.Logger, tc *projectTestCase) {
		session, _ := tc.project.Session()
		// Exec
		cleanup, err := tc.project.Reset(ctx)
		if err := cleanup(); err != nil {
			t.Error(err)
		}
		if err != nil && !errors.Is(err, ErrSessionNotFound) {
			t.Error(err)
		}
		// Grab the event
		logger.Info(ctx)
		wideEvent := make(map[string]any)
		if err := json.Unmarshal(out.Bytes(), &wideEvent); err != nil {
			t.Error(err)
		}
		event := wideEvent["projectResetEvent"].(map[string]any)
		// Check all the fields
		// field: Name
		name := event["name"].(string)
		if name != tc.project.Name {
			t.Error("Event name does not match")
		}
		// field: InitialStatus
		status := event["initialStatus"].(float64)
		if status != float64(tc.initialStatus()) {
			t.Error("The initial status should match")
		}
		// field: InitialSessionId
		if tc.initialStatus() >= PROJECT_STATUS_ACTIVE {
			initialSessionId := event["sessionId"].(string)
			if initialSessionId != string(session.Id) {
				t.Error("Event initial sessionId does not match")
			}
		} else {
			if _, ok := event["sessionId"]; ok {
				t.Error("Should not have an initial session")
			}
		}
		// field: TempSessionName
		if tc.initialStatus() > PROJECT_STATUS_INACTIVE {
			tempSessionName := event["tempSessionName"].(string)
			if tempSessionName == "" {
				t.Error("Should return a temp session name if used")
			}
		} else {
			if _, ok := event["tempSessionName"]; ok {
				t.Error("Should not have a temp session")
			}
		}
		// field: Errors
		if tc.initialStatus() >= PROJECT_STATUS_ACTIVE {
			if _, ok := event["errors"]; ok {
				t.Error("Should not have errors")
			}
		} else {
			errs := event["errors"].([]any)
			if len(errs) != 1 {
				t.Error("Should have one error")
			}
		}
	})
}
