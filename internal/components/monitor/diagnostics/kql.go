package diagnostics

import (
	"fmt"
	"strings"
	"time"
)

// Log level prefixes used in Kubernetes logs
var logLevelPrefixes = map[string]string{
	"info":    "I",
	"warning": "W",
	"error":   "E",
}

// Audit log categories that use different log level format
var auditCategories = map[string]bool{
	"kube-audit":       true,
	"kube-audit-admin": true,
}

// isAuditCategory checks if the given category is an audit log category
func isAuditCategory(category string) bool {
	return auditCategories[category]
}

// BuildSafeKQLQuery builds pre-validated KQL queries to prevent injection, scoped to specific AKS cluster
func BuildSafeKQLQuery(category, logLevel string, maxRecords int, clusterResourceID string) string {
	// Convert resource ID to uppercase as it's stored in uppercase in Log Analytics
	upperResourceID := strings.ToUpper(clusterResourceID)
	baseQuery := fmt.Sprintf("AzureDiagnostics | where Category == '%s' and ResourceId == '%s'", category, upperResourceID)

	if logLevel != "" && !isAuditCategory(category) {
		// For Kubernetes component logs (not audit), use the log_s prefix pattern
		// Kubernetes logs use format like "I0715" (Info), "W0715" (Warning), "E0715" (Error)
		// Audit logs don't follow this pattern, so we skip log level filtering for them
		if prefix, exists := logLevelPrefixes[strings.ToLower(logLevel)]; exists {
			baseQuery += fmt.Sprintf(" | where log_s startswith '%s'", prefix)
		}
	}

	baseQuery += " | order by TimeGenerated desc"
	baseQuery += fmt.Sprintf(" | limit %d", maxRecords)

	// Project only essential fields: log content, timestamp, and level
	baseQuery += " | project TimeGenerated, Level, log_s"

	return baseQuery
}

// CalculateTimespan converts start/end times to Azure CLI timespan format
func CalculateTimespan(startTime, endTime string) (string, error) {
	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return "", fmt.Errorf("invalid start time format: %w", err)
	}

	var end time.Time
	if endTime != "" {
		end, err = time.Parse(time.RFC3339, endTime)
		if err != nil {
			return "", fmt.Errorf("invalid end time format: %w", err)
		}
	} else {
		// Default to current time if no end time specified
		end = time.Now()
	}

	// Azure CLI timespan format: start_time/end_time in ISO8601
	timespan := fmt.Sprintf("%s/%s", start.Format(time.RFC3339), end.Format(time.RFC3339))
	return timespan, nil
}
