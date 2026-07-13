package inspektorgadget

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Azure/aks-mcp/internal/config"
)

// mockGadgetManager implements GadgetManager interface for testing
type mockGadgetManager struct {
	isDeployed      bool
	version         string
	deployedMessage string
	deployError     error
	runResult       string
	runError        error
	startResult     string
	startError      error
	stopError       error
	getResultsData  string
	getResultsError error
	gadgetInstances []*GadgetInstance
	listError       error
}

func (m *mockGadgetManager) RunGadget(ctx context.Context, image string, params map[string]string, duration time.Duration) (string, error) {
	if m.runError != nil {
		return "", m.runError
	}
	return m.runResult, nil
}

func (m *mockGadgetManager) StartGadget(ctx context.Context, image string, params map[string]string, tags []string) (string, error) {
	if m.startError != nil {
		return "", m.startError
	}
	return m.startResult, nil
}

func (m *mockGadgetManager) StopGadget(ctx context.Context, id string) error {
	return m.stopError
}

func (m *mockGadgetManager) GetResults(ctx context.Context, id string) (string, error) {
	if m.getResultsError != nil {
		return "", m.getResultsError
	}
	return m.getResultsData, nil
}

func (m *mockGadgetManager) ListGadgets(ctx context.Context) ([]*GadgetInstance, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	return m.gadgetInstances, nil
}

func (m *mockGadgetManager) IsDeployed(ctx context.Context) (bool, string, error) {
	return m.isDeployed, m.deployedMessage, m.deployError
}

func (m *mockGadgetManager) GetVersion() (string, error) {
	return m.version, nil
}

func (m *mockGadgetManager) SetRuntimeNamespace(namespace string) {}

func TestNotDeployedMessage(t *testing.T) {
	cfg := &config.ConfigData{}

	t.Run("with cluster identity includes az command and offer", func(t *testing.T) {
		params := map[string]interface{}{
			"subscription_id": "sub-123",
			"resource_group":  "rg-test",
			"cluster_name":    "cluster-test",
		}
		msg := notDeployedMessage(params, cfg)
		for _, want := range []string{
			"not deployed",
			"az k8s-extension create",
			"--cluster-name cluster-test",
			"action 'deploy'",
			"I can deploy it for you",
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("message missing %q, got %q", want, msg)
			}
		}
	})

	t.Run("without cluster identity still offers to deploy", func(t *testing.T) {
		msg := notDeployedMessage(map[string]interface{}{}, cfg)
		if !strings.Contains(msg, "action 'deploy'") {
			t.Errorf("expected deploy offer, got %q", msg)
		}
		if !strings.Contains(msg, "not deployed") {
			t.Errorf("expected not-deployed text, got %q", msg)
		}
	})
}

func TestHandleLifecycleActionReadonlyConfirmation(t *testing.T) {
	mgr := &mockGadgetManager{version: "0.53.0"}
	cfg := &config.ConfigData{AccessLevel: "readonly"}
	params := map[string]interface{}{
		"subscription_id": "sub-123",
		"resource_group":  "rg-test",
		"cluster_name":    "cluster-test",
	}

	tests := []struct {
		name     string
		action   string
		deployed bool
		wantCmd  string
	}{
		{
			name:     "deploy surfaces az create command",
			action:   deployAction,
			deployed: false,
			wantCmd:  "az k8s-extension create",
		},
		{
			name:     "undeploy surfaces az delete command",
			action:   undeployAction,
			deployed: true,
			wantCmd:  "az k8s-extension delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handleLifecycleAction(mgr, tt.deployed, tt.action, params, cfg)
			if err == nil {
				t.Fatalf("expected confirmation error, got nil")
			}
			msg := err.Error()
			if !strings.Contains(msg, "confirmation required") {
				t.Errorf("expected confirmation message, got %q", msg)
			}
			if !strings.Contains(msg, tt.wantCmd) {
				t.Errorf("expected message to include %q, got %q", tt.wantCmd, msg)
			}
			for _, want := range []string{"--cluster-name cluster-test", "--resource-group rg-test", "--name inspektor-gadget"} {
				if !strings.Contains(msg, want) {
					t.Errorf("expected message to include %q, got %q", want, msg)
				}
			}
		})
	}
}

func TestHandleLifecycleActionIsDeployed(t *testing.T) {
	cfg := &config.ConfigData{AccessLevel: "readonly"}
	// is_deployed must work without cluster identity or confirmation.
	mgr := &mockGadgetManager{version: "0.53.0"}
	got, err := handleLifecycleAction(mgr, true, isDeployedAction, map[string]interface{}{}, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "0.53.0") {
		t.Errorf("expected version in output, got %q", got)
	}
}

func TestInspektorGadgetHandler(t *testing.T) {
	cfg := &config.ConfigData{}

	t.Run("invalid action", func(t *testing.T) {
		mockMgr := &mockGadgetManager{
			isDeployed: true,
		}

		handler := InspektorGadgetHandler(mockMgr, cfg)
		params := map[string]interface{}{
			"action": "invalid_action",
		}

		_, err := handler.Handle(context.Background(), params, cfg)
		if err == nil {
			t.Error("expected error for invalid action, got nil")
		} else {
			expectedErr := fmt.Sprintf("invalid action: invalid_action, expected one of %v", getActions())
			if err.Error() != expectedErr {
				t.Errorf("expected error %q, got %q", expectedErr, err.Error())
			}
		}
	})

	t.Run("start and stop gadget", func(t *testing.T) {
		mockMgr := &mockGadgetManager{
			isDeployed:     true,
			startResult:    "gadget-1",
			getResultsData: "gadget results",
		}

		handler := InspektorGadgetHandler(mockMgr, cfg)
		params := map[string]interface{}{
			"action":      "start",
			"gadget_name": observeDNS,
		}

		result, err := handler.Handle(context.Background(), params, cfg)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if result != "Gadget started with ID: gadget-1" {
			t.Errorf("expected 'gadget started', got %q", result)
		}

		params = map[string]interface{}{
			"action":    "get_results",
			"gadget_id": "gadget-1",
		}

		result, err = handler.Handle(context.Background(), params, cfg)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if result != "gadget results" {
			t.Errorf("expected 'gadget results', got %q", result)
		}

		params = map[string]interface{}{
			"action":    "stop",
			"gadget_id": "gadget-1",
		}

		result, err = handler.Handle(context.Background(), params, cfg)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if result != "Gadget with ID gadget-1 stopped successfully" {
			t.Errorf("expected 'Gadget with ID gadget-1 stopped successfully', got %q", result)
		}
	})

	t.Run("run action", func(t *testing.T) {
		mockMgr := &mockGadgetManager{
			isDeployed: true,
			runResult:  "DNS trace complete",
		}

		handler := InspektorGadgetHandler(mockMgr, cfg)
		params := map[string]interface{}{
			"action":           "run",
			"gadget_name":      observeDNS,
			"duration":         5.0,
			"namespace":        "kube-system",
			"observe_dns.name": "kubernetes",
		}

		result, err := handler.Handle(context.Background(), params, cfg)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if result != "DNS trace complete" {
			t.Errorf("expected 'DNS trace complete', got %q", result)
		}
	})

	t.Run("list gadgets", func(t *testing.T) {
		gadgetInstances := []*GadgetInstance{
			{
				ID:        "gadget-1",
				StartedAt: "2023-01-01T00:00:00Z",
			},
			{
				ID:        "gadget-2",
				StartedAt: "2023-01-01T01:00:00Z",
			},
		}

		mockMgr := &mockGadgetManager{
			isDeployed:      true,
			gadgetInstances: gadgetInstances,
		}

		handler := InspektorGadgetHandler(mockMgr, cfg)
		params := map[string]interface{}{
			"action": "list_gadgets",
		}

		result, err := handler.Handle(context.Background(), params, cfg)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		var parsedResult []*GadgetInstance
		if err := json.Unmarshal([]byte(result), &parsedResult); err != nil {
			t.Errorf("failed to parse result JSON: %v", err)
		}

		if len(parsedResult) != 2 {
			t.Errorf("expected 2 gadget instances, got %d", len(parsedResult))
		}
	})
}
