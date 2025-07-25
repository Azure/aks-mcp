package diagnostics

import (
	"fmt"
	"strings"
	"time"
)

// LogLevelMapping defines the mapping between log levels and their representations
type LogLevelMapping struct {
	AzureDiagnosticsPrefix string // Prefix used in Azure Diagnostics log_s field
	ResourceSpecificLevel  string // Level value used in resource-specific tables
}

// logLevelMappings contains the mapping for each log level
var logLevelMappings = map[string]LogLevelMapping{
	"info":    {AzureDiagnosticsPrefix: "I", ResourceSpecificLevel: "INFO"},
	"warning": {AzureDiagnosticsPrefix: "W", ResourceSpecificLevel: "WARNING"},
	"error":   {AzureDiagnosticsPrefix: "E", ResourceSpecificLevel: "ERROR"},
}

// AuditCategories defines which log categories are audit logs (different handling)
var auditCategories = map[string]bool{
	"kube-audit":       true,
	"kube-audit-admin": true,
}

// ResourceSpecificTableMapping defines the mapping from log categories to resource-specific table names
var resourceSpecificTableMapping = map[string]string{
	"kube-audit":               "AKSAudit",
	"kube-audit-admin":         "AKSAuditAdmin",
	"kube-apiserver":           "AKSControlPlane",
	"kube-controller-manager":  "AKSControlPlane",
	"kube-scheduler":           "AKSControlPlane",
	"cluster-autoscaler":       "AKSControlPlane",
	"cloud-controller-manager": "AKSControlPlane",
	"guard":                    "AKSControlPlane",
	"csi-azuredisk-controller": "AKSControlPlane",
	"csi-azurefile-controller": "AKSControlPlane",
	"csi-snapshot-controller":  "AKSControlPlane",
}

// KQLQueryBuilder builds KQL queries for AKS control plane logs
type KQLQueryBuilder struct {
	category           string    // The log category (e.g., "kube-audit", "kube-apiserver").
	logLevel           string    // The log level (e.g., "info", "warning", "error").
	maxRecords         int       // The maximum number of records to retrieve.
	clusterResourceID  string    // The resource ID of the cluster being queried.
	isResourceSpecific bool      // Indicates whether the query targets a resource-specific table.
	actualTableMode    TableMode // Specifies the mode of the table being queried (e.g., AzureDiagnosticsMode or ResourceSpecificMode).
	// Note: `isResourceSpecific` and `actualTableMode` are related. If `isResourceSpecific` is true,
	// `actualTableMode` is typically set to ResourceSpecificMode. Otherwise, it is set to AzureDiagnosticsMode.
	selectedTable       string // The name of the table selected for the query.
	processedResourceID string // The processed resource ID used in the query.
}

// TableMode represents the type of table being used
type TableMode int

const (
	AzureDiagnosticsMode TableMode = iota
	ResourceSpecificMode
)

// NewKQLQueryBuilder creates a new KQL query builder instance
func NewKQLQueryBuilder(category, logLevel string, maxRecords int, clusterResourceID string, isResourceSpecific bool) *KQLQueryBuilder {
	return &KQLQueryBuilder{
		category:           category,
		logLevel:           logLevel,
		maxRecords:         maxRecords,
		clusterResourceID:  clusterResourceID,
		isResourceSpecific: isResourceSpecific,
	}
}

// determineTableStrategy decides which table to use and processes the resource ID accordingly
func (q *KQLQueryBuilder) determineTableStrategy() {
	if q.isResourceSpecific {
		if tableName, exists := resourceSpecificTableMapping[q.category]; exists {
			q.actualTableMode = ResourceSpecificMode
			q.selectedTable = tableName
			// Resource-specific tables store _ResourceId in lowercase
			q.processedResourceID = strings.ToLower(q.clusterResourceID)
		} else {
			// Fallback to Azure Diagnostics for unmapped categories
			q.actualTableMode = AzureDiagnosticsMode
			q.selectedTable = "AzureDiagnostics"
			// Azure Diagnostics stores ResourceId in uppercase
			q.processedResourceID = strings.ToUpper(q.clusterResourceID)
		}
	} else {
		q.actualTableMode = AzureDiagnosticsMode
		q.selectedTable = "AzureDiagnostics"
		q.processedResourceID = strings.ToUpper(q.clusterResourceID)
	}
}

// buildBaseQuery creates the initial table selection and filtering clause
func (q *KQLQueryBuilder) buildBaseQuery() string {
	switch q.actualTableMode {
	case ResourceSpecificMode:
		return fmt.Sprintf("%s | where _ResourceId == '%s'", q.selectedTable, q.processedResourceID)
	case AzureDiagnosticsMode:
		return fmt.Sprintf("%s | where Category == '%s' and ResourceId == '%s'", q.selectedTable, q.category, q.processedResourceID)
	default:
		// Fallback to Azure Diagnostics
		return fmt.Sprintf("AzureDiagnostics | where Category == '%s' and ResourceId == '%s'", q.category, q.processedResourceID)
	}
}

// isAuditCategory checks if the current category is an audit log category
func (q *KQLQueryBuilder) isAuditCategory() bool {
	return auditCategories[q.category]
}

// addLogLevelFilter adds log level filtering if applicable
func (q *KQLQueryBuilder) addLogLevelFilter(baseQuery string) string {
	// Skip log level filtering for audit categories or empty log level
	if q.logLevel == "" || q.isAuditCategory() {
		return baseQuery
	}

	mapping, exists := logLevelMappings[strings.ToLower(q.logLevel)]
	if !exists {
		return baseQuery // Unknown log level, skip filtering
	}

	switch q.actualTableMode {
	case ResourceSpecificMode:
		return baseQuery + fmt.Sprintf(" | where Level == '%s'", mapping.ResourceSpecificLevel)
	case AzureDiagnosticsMode:
		return baseQuery + fmt.Sprintf(" | where log_s startswith '%s'", mapping.AzureDiagnosticsPrefix)
	default:
		return baseQuery
	}
}

// addOrderingAndLimit adds the ordering and limit clauses
func (q *KQLQueryBuilder) addOrderingAndLimit(query string) string {
	query += " | order by TimeGenerated desc"
	query += fmt.Sprintf(" | limit %d", q.maxRecords)
	return query
}

// addProjection adds the appropriate field projection based on table type
func (q *KQLQueryBuilder) addProjection(query string) string {
	switch q.actualTableMode {
	case ResourceSpecificMode:
		return q.addResourceSpecificProjection(query)
	case AzureDiagnosticsMode:
		return query + " | project TimeGenerated, Level, log_s"
	default:
		return query + " | project TimeGenerated, Level, log_s"
	}
}

// addResourceSpecificProjection adds projection for resource-specific tables
func (q *KQLQueryBuilder) addResourceSpecificProjection(query string) string {
	switch q.selectedTable {
	case "AKSAudit", "AKSAuditAdmin":
		// Audit tables have structured fields
		return query + " | project TimeGenerated, Level, AuditId, Stage, RequestUri, Verb, User"
	case "AKSControlPlane":
		// Control plane table has message field
		return query + " | project TimeGenerated, Category, Level, Message, PodName"
	default:
		// Fallback projection for unknown resource-specific tables
		return query + " | project TimeGenerated, Level, Message"
	}
}

// Build constructs the complete KQL query
func (q *KQLQueryBuilder) Build() string {
	// Step 1: Determine table strategy
	q.determineTableStrategy()

	// Step 2: Build base query with table and resource filtering
	query := q.buildBaseQuery()

	// Step 3: Add log level filtering
	query = q.addLogLevelFilter(query)

	// Step 4: Add ordering and limit
	query = q.addOrderingAndLimit(query)

	// Step 5: Add field projection
	query = q.addProjection(query)

	return query
}

// BuildSafeKQLQuery builds pre-validated KQL queries to prevent injection, scoped to specific AKS cluster
// Supports both Azure Diagnostics and Resource-specific destination tables
// This function maintains backward compatibility with the existing API
func BuildSafeKQLQuery(category, logLevel string, maxRecords int, clusterResourceID string, isResourceSpecific bool) string {
	builder := NewKQLQueryBuilder(category, logLevel, maxRecords, clusterResourceID, isResourceSpecific)
	return builder.Build()
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
