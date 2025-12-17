package k8s

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Azure/aks-mcp/internal/config"
	k8sconfig "github.com/Azure/mcp-kubernetes/pkg/config"
	k8ssecurity "github.com/Azure/mcp-kubernetes/pkg/security"
	k8stools "github.com/Azure/mcp-kubernetes/pkg/tools"
)

var benchOut *k8sconfig.ConfigData

// This test suite verifies config mapping (without mutating input), adapter delegation,
// error propagation, and current nil-config behavior. The benchmark provides a baseline
// for detecting performance regressions.

// mustEqual keeps assertions concise with consistent failure messages.
func mustEqual[T comparable](t *testing.T, got, want T, msg string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: got %v, want %v", msg, got, want)
	}
}

// mustDeepEqual keeps deep-structure assertions concise with consistent messages.
func mustDeepEqual(t *testing.T, got, want interface{}, msg string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %#v, want %#v", msg, got, want)
	}
}

// fakeExecutor captures inputs and returns preset output/error to observe delegation.
type fakeExecutor struct {
	lastParams map[string]interface{}
	lastCfg    *k8sconfig.ConfigData
	out        string
	err        error
}

var _ k8stools.CommandExecutor = (*fakeExecutor)(nil)

func (f *fakeExecutor) Execute(ctx context.Context, params map[string]interface{}, cfg *k8sconfig.ConfigData) (string, error) {
	f.lastParams = params
	f.lastCfg = cfg
	return f.out, f.err
}

func TestConvertConfig_MapsFields(t *testing.T) {
	t.Parallel()

	in := &config.ConfigData{
		Timeout:           600,
		Transport:         "stdio",
		Host:              "127.0.0.1",
		Port:              8000,
		AccessLevel:       "readonly",
		EnabledComponents: []string{"helm"},
		AllowNamespaces:   "default,platform",
		OTLPEndpoint:      "otel:4317",
	}

	got := ConvertConfig(in)
	if got == nil {
		t.Fatal("ConvertConfig returned nil")
	}

	mustEqual(t, got.Timeout, in.Timeout, "Timeout")
	mustEqual(t, got.Transport, in.Transport, "Transport")
	mustEqual(t, got.Host, in.Host, "Host")
	mustEqual(t, got.Port, in.Port, "Port")
	mustEqual(t, got.AccessLevel, in.AccessLevel, "AccessLevel")
	mustEqual(t, got.OTLPEndpoint, in.OTLPEndpoint, "OTLPEndpoint")
	expectedAdditionalTools := map[string]bool{"helm": true}
	mustDeepEqual(t, got.AdditionalTools, expectedAdditionalTools, "AdditionalTools")
	mustEqual(t, got.AllowNamespaces, in.AllowNamespaces, "AllowNamespaces")

	if got.SecurityConfig == nil {
		t.Fatal("SecurityConfig is nil")
	}
	mustEqual(t, got.SecurityConfig.AccessLevel, k8ssecurity.AccessLevel(in.AccessLevel), "SecurityConfig.AccessLevel")
}

func TestConvertConfig_DoesNotMutateInput(t *testing.T) {
	t.Parallel()

	in := &config.ConfigData{
		Timeout:           42,
		Transport:         "stdio",
		Host:              "127.0.0.1",
		Port:              8000,
		AccessLevel:       "readonly",
		EnabledComponents: []string{"helm"},
		AllowNamespaces:   "default",
		OTLPEndpoint:      "otel:4317",
	}

	// Verify the "no input mutation" guarantee by comparing to a copy.
	orig := *in
	orig.EnabledComponents = make([]string, len(in.EnabledComponents))
	copy(orig.EnabledComponents, in.EnabledComponents)

	out := ConvertConfig(in)
	mustDeepEqual(t, in, &orig, "input should remain unchanged")

	if out == nil || out.SecurityConfig == nil {
		t.Fatalf("expected non-nil output and SecurityConfig, got %#v", out)
	}

	mustEqual(t, out.Timeout, in.Timeout, "Timeout")
	mustEqual(t, out.Transport, in.Transport, "Transport")
	mustEqual(t, out.Host, in.Host, "Host")
	mustEqual(t, out.Port, in.Port, "Port")
	mustEqual(t, out.AccessLevel, in.AccessLevel, "AccessLevel")
	mustEqual(t, out.OTLPEndpoint, in.OTLPEndpoint, "OTLPEndpoint")
	mustEqual(t, out.AllowNamespaces, in.AllowNamespaces, "AllowNamespaces")
	expectedAdditionalTools := map[string]bool{"helm": true}
	mustDeepEqual(t, out.AdditionalTools, expectedAdditionalTools, "AdditionalTools")
	mustEqual(t, out.SecurityConfig.AccessLevel, k8ssecurity.AccessLevel(in.AccessLevel), "SecurityConfig.AccessLevel")
}

func TestConvertConfig_ZeroValueCfg(t *testing.T) {
	t.Parallel()

	// Document current behavior when callers pass an uninitialized config.
	in := &config.ConfigData{}
	orig := *in

	out := ConvertConfig(in)

	mustDeepEqual(t, in, &orig, "input unchanged")

	if out == nil || out.SecurityConfig == nil {
		t.Fatalf("non-nil out and SecurityConfig required, got %#v", out)
	}

	mustEqual(t, out.Timeout, 0, "Timeout")
	mustEqual(t, out.Transport, "", "Transport")
	mustEqual(t, out.Host, "", "Host")
	mustEqual(t, out.Port, 0, "Port")
	mustEqual(t, out.AccessLevel, "", "AccessLevel")
	mustEqual(t, out.OTLPEndpoint, "", "OTLPEndpoint")
	mustEqual(t, out.AllowNamespaces, "", "AllowNamespaces")
	expectedAdditionalTools := map[string]bool{"helm": true, "cilium": true, "hubble": true}
	mustDeepEqual(t, out.AdditionalTools, expectedAdditionalTools, "AdditionalTools")
	mustEqual(t, out.SecurityConfig.AccessLevel, k8ssecurity.AccessLevel(""), "SecurityConfig.AccessLevel")
}

func TestExecutorAdapter_DelegatesAndForwards(t *testing.T) {
	t.Parallel()

	fe := &fakeExecutor{out: "ok"}
	adapter := WrapK8sExecutor(fe)

	params := map[string]interface{}{"k": "v"}
	inCfg := &config.ConfigData{
		Timeout:           10,
		Transport:         "stdio",
		Host:              "127.0.0.1",
		Port:              8000,
		AccessLevel:       "readonly",
		EnabledComponents: []string{"helm"},
		AllowNamespaces:   "default",
	}

	got, err := adapter.Execute(context.Background(), params, inCfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mustEqual(t, got, "ok", "adapter output")
	mustDeepEqual(t, fe.lastParams, params, "params forwarded")

	if fe.lastCfg == nil || fe.lastCfg.SecurityConfig == nil {
		t.Fatalf("expected non-nil converted cfg + SecurityConfig, got %#v", fe.lastCfg)
	}

	mustEqual(t, fe.lastCfg.Port, inCfg.Port, "Port")
	mustEqual(t, fe.lastCfg.AccessLevel, inCfg.AccessLevel, "AccessLevel")
	expectedAdditionalTools := map[string]bool{"helm": true}
	mustDeepEqual(t, fe.lastCfg.AdditionalTools, expectedAdditionalTools, "AdditionalTools")
	mustEqual(t, fe.lastCfg.AllowNamespaces, inCfg.AllowNamespaces, "AllowNamespaces")
	mustEqual(t, fe.lastCfg.SecurityConfig.AccessLevel, k8ssecurity.AccessLevel("readonly"), "SecurityConfig.AccessLevel")
}

func TestExecutorAdapter_PropagatesError(t *testing.T) {
	t.Parallel()

	fe := &fakeExecutor{err: errors.New("boom")}
	adapter := WrapK8sExecutor(fe)

	_, err := adapter.Execute(context.Background(), map[string]interface{}{"x": 1}, &config.ConfigData{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExecutorAdapter_PanicsOnNilConfig_CurrentBehavior(t *testing.T) {
	t.Parallel()

	// Document the current precondition: cfg must be non-nil.
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic when cfg is nil")
		}
	}()

	fe := &fakeExecutor{}
	adapter := WrapK8sExecutor(fe)
	_, _ = adapter.Execute(context.Background(), map[string]interface{}{"x": 1}, nil)
}

// BenchmarkConvertConfig tracks drift in allocation/time costs over time.
// Helps detect subtle regressions when config mapping logic evolves.
func BenchmarkConvertConfig(b *testing.B) {
	in := &config.ConfigData{
		Timeout:           600,
		Transport:         "stdio",
		Host:              "127.0.0.1",
		Port:              8000,
		AccessLevel:       "readonly",
		EnabledComponents: []string{"helm"},
		AllowNamespaces:   "default,platform",
		OTLPEndpoint:      "otel:4317",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchOut = ConvertConfig(in)
	}
}
