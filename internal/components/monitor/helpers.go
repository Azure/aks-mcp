package monitor

import (
	"fmt"
	"slices"
)

// supportedMonitoringOperations defines all supported monitoring operations
var supportedMonitoringOperations = []string{
	string(OpMetrics), string(OpResourceHealth), string(OpAppInsights),
	string(OpDiagnostics), string(OpControlPlaneLogs),
}

// ValidateMonitoringOperation checks if the monitoring operation is supported
func ValidateMonitoringOperation(operation string) bool {
	return slices.Contains(supportedMonitoringOperations, operation)
}

// GetSupportedMonitoringOperations returns all supported monitoring operations
func GetSupportedMonitoringOperations() []string {
	return supportedMonitoringOperations
}

// ValidateMetricsQueryType checks if the metrics query type is supported
func ValidateMetricsQueryType(queryType string) bool {
	supportedTypes := []string{"list", "list-definitions", "list-namespaces"}
	return slices.Contains(supportedTypes, queryType)
}

// MapMetricsQueryTypeToCommand maps a metrics query type to its corresponding az command
func MapMetricsQueryTypeToCommand(queryType string) (string, error) {
	commandMap := map[string]string{
		"list":             "az monitor metrics list",
		"list-definitions": "az monitor metrics list-definitions",
		"list-namespaces":  "az monitor metrics list-namespaces",
	}

	cmd, exists := commandMap[queryType]
	if !exists {
		return "", fmt.Errorf("unsupported metrics query type '%s'. Supported types: list, list-definitions, list-namespaces", queryType)
	}

	return cmd, nil
}
