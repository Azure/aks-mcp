package components

import (
	"testing"
)

func TestValidateComponents_EmptyList(t *testing.T) {
	valid, invalid, err := ValidateComponents([]string{})
	if err != nil {
		t.Errorf("Empty list should not return error, got: %v", err)
	}
	if len(valid) != 0 {
		t.Errorf("Expected 0 valid components for empty list, got: %d", len(valid))
	}
	if len(invalid) != 0 {
		t.Errorf("Expected 0 invalid components for empty list, got: %d", len(invalid))
	}
}

func TestValidateComponents_ValidComponents(t *testing.T) {
	testCases := []struct {
		name       string
		components []string
		wantValid  int
	}{
		{
			name:       "single valid component",
			components: []string{"kubectl"},
			wantValid:  1,
		},
		{
			name:       "multiple valid components",
			components: []string{"kubectl", "helm", "monitor"},
			wantValid:  3,
		},
		{
			name:       "all azure components",
			components: []string{"az_cli", "monitor", "fleet", "network", "compute", "detectors", "advisor", "inspektorgadget"},
			wantValid:  8,
		},
		{
			name:       "all kubernetes components",
			components: []string{"kubectl", "helm", "cilium", "hubble"},
			wantValid:  4,
		},
		{
			name:       "mixed components with spaces",
			components: []string{" kubectl ", " helm ", " monitor "},
			wantValid:  3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid, invalid, err := ValidateComponents(tc.components)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if len(valid) != tc.wantValid {
				t.Errorf("Expected %d valid components, got: %d", tc.wantValid, len(valid))
			}
			if len(invalid) != 0 {
				t.Errorf("Expected 0 invalid components, got: %d (%v)", len(invalid), invalid)
			}
		})
	}
}

func TestValidateComponents_InvalidComponents(t *testing.T) {
	testCases := []struct {
		name        string
		components  []string
		wantInvalid int
		wantError   bool
	}{
		{
			name:        "single invalid component",
			components:  []string{"invalid"},
			wantInvalid: 1,
			wantError:   true,
		},
		{
			name:        "multiple invalid components",
			components:  []string{"invalid1", "invalid2"},
			wantInvalid: 2,
			wantError:   true,
		},
		{
			name:        "mixed valid and invalid",
			components:  []string{"kubectl", "invalid", "helm"},
			wantInvalid: 1,
			wantError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid, invalid, err := ValidateComponents(tc.components)
			if tc.wantError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.wantError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if len(invalid) != tc.wantInvalid {
				t.Errorf("Expected %d invalid components, got: %d", tc.wantInvalid, len(invalid))
			}
			// When there are some valid components, error should be nil
			if len(valid) > 0 && err != nil {
				t.Errorf("Should not return error when at least one valid component exists, got: %v", err)
			}
		})
	}
}

func TestIsComponentEnabled_EmptyList(t *testing.T) {
	// Empty list means all components are enabled
	if !IsComponentEnabled("kubectl", []string{}) {
		t.Error("kubectl should be enabled with empty list")
	}
	if !IsComponentEnabled("helm", []string{}) {
		t.Error("helm should be enabled with empty list")
	}
	if !IsComponentEnabled("monitor", []string{}) {
		t.Error("monitor should be enabled with empty list")
	}
}

func TestIsComponentEnabled_SpecificComponents(t *testing.T) {
	enabledComponents := []string{"kubectl", "helm", "monitor"}

	testCases := []struct {
		component string
		want      bool
	}{
		{"kubectl", true},
		{"helm", true},
		{"monitor", true},
		{"cilium", false},
		{"hubble", false},
		{"az_cli", false},
	}

	for _, tc := range testCases {
		t.Run(tc.component, func(t *testing.T) {
			got := IsComponentEnabled(tc.component, enabledComponents)
			if got != tc.want {
				t.Errorf("IsComponentEnabled(%s) = %v, want %v", tc.component, got, tc.want)
			}
		})
	}
}

func TestIsComponentEnabled_CaseInsensitive(t *testing.T) {
	enabledComponents := []string{"KubeCTL", "HELM"}

	if !IsComponentEnabled("kubectl", enabledComponents) {
		t.Error("kubectl should be enabled (case insensitive)")
	}
	if !IsComponentEnabled("helm", enabledComponents) {
		t.Error("helm should be enabled (case insensitive)")
	}
	if !IsComponentEnabled("KUBECTL", enabledComponents) {
		t.Error("KUBECTL should be enabled (case insensitive)")
	}
}

func TestIsComponentEnabled_InvalidComponent(t *testing.T) {
	enabledComponents := []string{"kubectl"}

	if IsComponentEnabled("invalid", enabledComponents) {
		t.Error("invalid component should not be enabled")
	}
}

func TestGetAllComponents(t *testing.T) {
	components := GetAllComponents()
	if len(components) != 12 {
		t.Errorf("Expected 12 components, got: %d", len(components))
	}

	// Verify all expected components exist
	expectedComponents := []string{
		"az_cli", "monitor", "fleet", "network", "compute", "detectors", "advisor", "inspektorgadget",
		"kubectl", "helm", "cilium", "hubble",
	}

	componentMap := make(map[string]bool)
	for _, comp := range components {
		componentMap[comp.Name] = true
	}

	for _, expected := range expectedComponents {
		if !componentMap[expected] {
			t.Errorf("Expected component %s not found", expected)
		}
	}
}

func TestGetComponentByName(t *testing.T) {
	testCases := []struct {
		name      string
		wantError bool
	}{
		{"kubectl", false},
		{"helm", false},
		{"monitor", false},
		{"invalid", true},
		{"", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			comp, err := GetComponentByName(tc.name)
			if tc.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if comp != nil {
					t.Error("Expected nil component for error case")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if comp == nil {
					t.Error("Expected component but got nil")
				}
				if comp != nil && comp.Name != tc.name {
					t.Errorf("Expected component name %s, got: %s", tc.name, comp.Name)
				}
			}
		})
	}
}
