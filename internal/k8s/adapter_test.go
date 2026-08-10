package k8s

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
	k8sconfig "github.com/Azure/mcp-kubernetes/pkg/config"
)

type fakeExecutor struct {
	called bool
	params map[string]interface{}
	cfg    *k8sconfig.ConfigData
	output string
	err    error
}

func (f *fakeExecutor) Execute(_ context.Context, params map[string]interface{}, cfg *k8sconfig.ConfigData) (string, error) {
	f.called = true
	f.params = params
	f.cfg = cfg
	return f.output, f.err
}

func TestExecutorAdapterDelegatesWithConvertedStdioConfig(t *testing.T) {
	underlying := &fakeExecutor{output: "ok"}
	params := map[string]interface{}{"command": "kubectl get pods"}
	cfg := &config.ConfigData{
		Timeout:           42,
		AccessLevel:       "readonly",
		EnabledComponents: []string{"helm"},
		AllowNamespaces:   "default",
	}

	output, err := WrapK8sExecutor(underlying).Execute(context.Background(), params, cfg)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if output != "ok" || !underlying.called || underlying.params["command"] != params["command"] {
		t.Fatalf("executor was not called as expected: output=%q called=%t params=%v", output, underlying.called, underlying.params)
	}
	if underlying.cfg.Transport != "stdio" || underlying.cfg.Timeout != 42 || underlying.cfg.AccessLevel != "readonly" {
		t.Fatalf("unexpected converted config: %#v", underlying.cfg)
	}
	if !underlying.cfg.AdditionalTools["helm"] || len(underlying.cfg.AdditionalTools) != 1 {
		t.Fatalf("unexpected additional tools: %#v", underlying.cfg.AdditionalTools)
	}
}

func TestExecutorAdapterPropagatesExecutorError(t *testing.T) {
	want := errors.New("executor failed")
	_, err := WrapK8sExecutor(&fakeExecutor{err: want}).Execute(context.Background(), nil, &config.ConfigData{})
	if !errors.Is(err, want) {
		t.Fatalf("Execute() error = %v, want %v", err, want)
	}
}

func TestExecutorAdapterBlocksAuthReconcileOnlyInReadonly(t *testing.T) {
	for _, tc := range []struct {
		name    string
		level   string
		command string
		blocked bool
	}{
		{"readonly reconcile", "readonly", "kubectl auth reconcile -f rbac.yaml", true},
		{"readonly quoted reconcile", "readonly", "kubectl\t-v=2 auth reconcile -f rbac.yaml", true},
		{"readonly can-i", "readonly", "kubectl auth can-i get pods", false},
		{"readwrite reconcile", "readwrite", "kubectl auth reconcile -f rbac.yaml", false},
		{"exec arguments", "readonly", "kubectl exec pod -- auth reconcile", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			underlying := &fakeExecutor{output: "ok"}
			output, err := WrapK8sExecutor(underlying).Execute(context.Background(), map[string]interface{}{"command": tc.command}, &config.ConfigData{AccessLevel: tc.level})
			if tc.blocked {
				if err == nil || underlying.called {
					t.Fatalf("reconcile must be rejected before delegation: error=%v called=%t", err, underlying.called)
				}
				return
			}
			if err != nil || !underlying.called || output != "ok" {
				t.Fatalf("command should be delegated: output=%q error=%v called=%t", output, err, underlying.called)
			}
		})
	}
}
