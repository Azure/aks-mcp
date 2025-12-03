package components

import (
	"fmt"
	"strings"
)

// Component represents a registrable component
type Component struct {
	Name        string
	Description string
}

// GetAllComponents returns all available components
func GetAllComponents() []Component {
	return []Component{
		// Azure Components
		{Name: "az_cli", Description: "Azure CLI operations (call_az or legacy az_aks_operations based on USE_LEGACY_TOOLS)"},
		{Name: "monitor", Description: "Azure monitoring and diagnostics for AKS"},
		{Name: "fleet", Description: "AKS Fleet management"},
		{Name: "network", Description: "Azure network resources for AKS"},
		{Name: "compute", Description: "Azure compute resources (VMSS/VM) for AKS"},
		{Name: "detectors", Description: "AppLens detector integration for AKS"},
		{Name: "advisor", Description: "Azure Advisor recommendations for AKS"},
		{Name: "inspektorgadget", Description: "eBPF-based observability tools"},

		// Kubernetes Components
		{Name: "kubectl", Description: "Core Kubernetes operations"},
		{Name: "helm", Description: "Helm package management"},
		{Name: "cilium", Description: "Cilium CNI operations"},
		{Name: "hubble", Description: "Hubble network observability"},
	}
}

// GetComponentByName returns a component by its name
func GetComponentByName(name string) (*Component, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	for _, comp := range GetAllComponents() {
		if comp.Name == name {
			return &comp, nil
		}
	}
	return nil, fmt.Errorf("unknown component: %s", name)
}

// ValidateComponents validates a list of component names
// Returns valid components, invalid component names, and an error if validation fails
// Requirements:
// - All component names must be recognized
// - At least one component must be enabled
func ValidateComponents(componentNames []string) (valid []string, invalid []string, err error) {
	// If empty list, all components are enabled - this is valid
	if len(componentNames) == 0 {
		return []string{}, []string{}, nil
	}

	for _, name := range componentNames {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "" {
			continue
		}
		if _, err := GetComponentByName(name); err != nil {
			invalid = append(invalid, name)
		} else {
			valid = append(valid, name)
		}
	}

	// Check if at least one valid component is enabled
	if len(valid) == 0 {
		if len(invalid) > 0 {
			return valid, invalid, fmt.Errorf("no valid components specified, invalid components: %s", strings.Join(invalid, ", "))
		}
		return valid, invalid, fmt.Errorf("at least one component must be enabled")
	}

	return valid, invalid, nil
}

// IsComponentEnabled checks if a component is enabled based on the configuration
// If enabledComponents is empty, all components are enabled by default
func IsComponentEnabled(componentName string, enabledComponents []string) bool {
	componentName = strings.TrimSpace(strings.ToLower(componentName))

	// Verify component exists
	_, err := GetComponentByName(componentName)
	if err != nil {
		return false
	}

	// If no components are specified, enable all
	if len(enabledComponents) == 0 {
		return true
	}

	// Check if component is in the enabled list
	for _, enabled := range enabledComponents {
		if strings.TrimSpace(strings.ToLower(enabled)) == componentName {
			return true
		}
	}

	return false
}
