package advisor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/aks-mcp/internal/azcli"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/logger"
)

// HandleAdvisorRecommendation is the main handler for Azure Advisor recommendation operations
func HandleAdvisorRecommendation(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		logger.Errorf("[ADVISOR] Missing operation parameter")
		return "", fmt.Errorf("operation parameter is required")
	}

	logger.Debugf("[ADVISOR] Handling operation: %s", operation)

	switch operation {
	case "list":
		return handleAKSAdvisorRecommendationList(params, cfg)
	case "report":
		return handleAKSAdvisorRecommendationReport(params, cfg)
	default:
		logger.Errorf("[ADVISOR] Invalid operation: %s", operation)
		return "", fmt.Errorf("invalid operation: %s. Allowed values: list, report", operation)
	}
}

// handleAKSAdvisorRecommendationList lists AKS-related recommendations
func handleAKSAdvisorRecommendationList(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	subscriptionID, ok := params["subscription_id"].(string)
	if !ok {
		logger.Errorf("[ADVISOR] Missing subscription_id parameter")
		return "", fmt.Errorf("subscription_id parameter is required")
	}

	// Get optional parameters
	resourceGroup, _ := params["resource_group"].(string)
	category, _ := params["category"].(string)
	severity, _ := params["severity"].(string)

	logger.Debugf("[ADVISOR] Listing recommendations for subscription: %s, resource_group: %s, category: %s, severity: %s",
		subscriptionID, resourceGroup, category, severity)

	// Get cluster names filter if provided
	var clusterNames []string
	if clusterNamesParam, ok := params["cluster_names"].(string); ok && clusterNamesParam != "" {
		// Parse comma-separated string into slice
		for _, name := range strings.Split(clusterNamesParam, ",") {
			if trimmedName := strings.TrimSpace(name); trimmedName != "" {
				clusterNames = append(clusterNames, trimmedName)
			}
		}
		logger.Debugf("[ADVISOR] Filtering by cluster names: %v", clusterNames)
	}

	// Execute Azure CLI command to get recommendations
	recommendations, err := listRecommendationsViaCLI(subscriptionID, resourceGroup, category, cfg)
	if err != nil {
		logger.Errorf("[ADVISOR] Failed to list recommendations: %v", err)
		return "", fmt.Errorf("failed to list recommendations: %w", err)
	}

	logger.Infof("[ADVISOR] Found %d total recommendations", len(recommendations))

	// Filter for AKS-related recommendations
	aksRecommendations := filterAKSRecommendationsFromCLI(recommendations)
	logger.Infof("[ADVISOR] Found %d AKS-related recommendations", len(aksRecommendations))

	// Apply additional filters
	if severity != "" {
		aksRecommendations = filterBySeverity(aksRecommendations, severity)
		logger.Debugf("[ADVISOR] After severity filter: %d recommendations", len(aksRecommendations))
	}
	if len(clusterNames) > 0 {
		aksRecommendations = filterByClusterNames(aksRecommendations, clusterNames)
		logger.Debugf("[ADVISOR] After cluster name filter: %d recommendations", len(aksRecommendations))
	}

	// Convert to AKS recommendation summaries
	summaries := convertToAKSRecommendationSummaries(aksRecommendations)

	// Return JSON response
	result, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		logger.Errorf("[ADVISOR] Failed to marshal recommendations: %v", err)
		return "", fmt.Errorf("failed to marshal recommendations: %w", err)
	}

	logger.Infof("[ADVISOR] Returning %d recommendation summaries", len(summaries))
	return string(result), nil
}

// handleAKSAdvisorRecommendationReport generates a comprehensive report
func handleAKSAdvisorRecommendationReport(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	subscriptionID, ok := params["subscription_id"].(string)
	if !ok {
		return "", fmt.Errorf("subscription_id parameter is required")
	}

	// Get optional parameters
	resourceGroup, _ := params["resource_group"].(string)
	format, _ := params["format"].(string)
	if format == "" {
		format = "summary"
	}

	// Get all AKS recommendations
	recommendations, err := listRecommendationsViaCLI(subscriptionID, resourceGroup, "", cfg)
	if err != nil {
		return "", fmt.Errorf("failed to list recommendations: %w", err)
	}

	// Filter for AKS-related recommendations
	aksRecommendations := filterAKSRecommendationsFromCLI(recommendations)
	summaries := convertToAKSRecommendationSummaries(aksRecommendations)

	// Generate report
	report := generateAKSAdvisorReport(subscriptionID, summaries, format)

	// Return JSON response
	result, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report: %w", err)
	}

	return string(result), nil
}

// listRecommendationsViaCLI executes Azure CLI command to list recommendations
func listRecommendationsViaCLI(subscriptionID, resourceGroup, category string, cfg *config.ConfigData) ([]CLIRecommendation, error) {
	executor := azcli.NewExecutor()

	// Build command arguments
	args := []string{"advisor", "recommendation", "list", "--subscription", subscriptionID, "--output", "json"}

	if resourceGroup != "" {
		args = append(args, "--resource-group", resourceGroup)
	}

	// Create command parameters
	cmdParams := map[string]interface{}{
		"command": "az " + strings.Join(args, " "),
	}

	logger.Debugf("[ADVISOR] Executing command: %s", cmdParams["command"])

	// Execute command
	output, err := executor.Execute(cmdParams, cfg)
	if err != nil {
		logger.Errorf("[ADVISOR] Command execution failed: %v", err)
		return nil, fmt.Errorf("failed to execute Azure CLI command: %w", err)
	}

	logger.Debugf("[ADVISOR] Command output length: %d characters", len(output))

	// Parse JSON output
	var recommendations []CLIRecommendation
	if err := json.Unmarshal([]byte(output), &recommendations); err != nil {
		logger.Errorf("[ADVISOR] Failed to parse JSON output: %v", err)
		return nil, fmt.Errorf("failed to parse recommendations JSON: %w", err)
	}

	logger.Debugf("[ADVISOR] Successfully parsed %d recommendations from CLI output", len(recommendations))
	return recommendations, nil
}

// filterAKSRecommendationsFromCLI filters recommendations to only AKS-related resources
func filterAKSRecommendationsFromCLI(recommendations []CLIRecommendation) []CLIRecommendation {
	var aksRecommendations []CLIRecommendation
	for _, rec := range recommendations {
		if isAKSRelatedCLI(rec.ID) {
			aksRecommendations = append(aksRecommendations, rec)
		}
	}
	return aksRecommendations
}

// isAKSRelatedCLI checks if a resource ID is related to AKS
func isAKSRelatedCLI(resourceID string) bool {
	if resourceID == "" {
		return false
	}
	return strings.Contains(resourceID, "Microsoft.ContainerService/managedClusters") ||
		strings.Contains(resourceID, "Microsoft.ContainerService/managedClusters/agentPools") ||
		(strings.Contains(resourceID, "Microsoft.Network/loadBalancers") && strings.Contains(resourceID, "kubernetes")) ||
		(strings.Contains(resourceID, "Microsoft.Network/publicIPAddresses") && strings.Contains(resourceID, "kubernetes"))
}

// filterBySeverity filters recommendations by severity level
func filterBySeverity(recommendations []CLIRecommendation, severity string) []CLIRecommendation {
	var filtered []CLIRecommendation
	for _, rec := range recommendations {
		if strings.EqualFold(rec.Impact, severity) {
			filtered = append(filtered, rec)
		}
	}
	return filtered
}

// filterByClusterNames filters recommendations by cluster names
func filterByClusterNames(recommendations []CLIRecommendation, clusterNames []string) []CLIRecommendation {
	var filtered []CLIRecommendation
	for _, rec := range recommendations {
		clusterName := extractAKSClusterNameFromCLI(rec.ID)
		for _, filterName := range clusterNames {
			if strings.EqualFold(clusterName, filterName) {
				filtered = append(filtered, rec)
				break
			}
		}
	}
	return filtered
}

// extractAKSClusterNameFromCLI extracts AKS cluster name from resource ID
func extractAKSClusterNameFromCLI(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	for i, part := range parts {
		if part == "managedClusters" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// convertToAKSRecommendationSummaries converts CLI recommendations to AKS recommendation summaries
func convertToAKSRecommendationSummaries(recommendations []CLIRecommendation) []AKSRecommendationSummary {
	var summaries []AKSRecommendationSummary
	for _, rec := range recommendations {
		summary := convertToAKSRecommendationSummary(rec)
		summaries = append(summaries, summary)
	}
	return summaries
}

// convertToAKSRecommendationSummary converts a single CLI recommendation to AKS recommendation summary
func convertToAKSRecommendationSummary(rec CLIRecommendation) AKSRecommendationSummary {
	clusterName := extractAKSClusterNameFromCLI(rec.ID)
	resourceGroup := extractResourceGroupFromResourceID(rec.ID)

	// Parse last updated time
	lastUpdated, _ := time.Parse(time.RFC3339, rec.LastUpdated)

	return AKSRecommendationSummary{
		ID:            rec.ID,
		Category:      rec.Category,
		Impact:        rec.Impact,
		ClusterName:   clusterName,
		ResourceGroup: resourceGroup,
		ResourceID:    rec.ID,
		Description:   rec.ShortDescription.Problem + " " + rec.ShortDescription.Solution,
		Severity:      rec.Impact, // Map impact to severity
		LastUpdated:   lastUpdated,
		Status:        "Active",
		AKSSpecific: AKSRecommendationDetails{
			ConfigurationArea: mapCategoryToConfigArea(rec.Category),
		},
	}
}

// extractResourceGroupFromResourceID extracts resource group name from Azure resource ID
func extractResourceGroupFromResourceID(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	for i, part := range parts {
		if part == "resourceGroups" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// mapCategoryToConfigArea maps Azure Advisor categories to configuration areas
func mapCategoryToConfigArea(category string) string {
	switch strings.ToLower(category) {
	case "cost":
		return "compute"
	case "security":
		return "security"
	case "performance":
		return "compute"
	case "highavailability":
		return "networking"
	default:
		return "general"
	}
}

// generateAKSAdvisorReport generates a comprehensive AKS advisor report
func generateAKSAdvisorReport(subscriptionID string, recommendations []AKSRecommendationSummary, format string) AKSAdvisorReport {
	// Generate summary statistics
	summary := generateReportSummary(recommendations)

	// Generate action items based on priority
	actionItems := generateActionItems(recommendations)

	// Group recommendations by cluster
	clusterBreakdown := groupRecommendationsByCluster(recommendations)

	return AKSAdvisorReport{
		SubscriptionID:   subscriptionID,
		GeneratedAt:      time.Now(),
		Summary:          summary,
		Recommendations:  recommendations,
		ActionItems:      actionItems,
		ClusterBreakdown: clusterBreakdown,
	}
}

// generateReportSummary generates summary statistics for the report
func generateReportSummary(recommendations []AKSRecommendationSummary) AKSReportSummary {
	byCategory := make(map[string]int)
	bySeverity := make(map[string]int)
	clustersMap := make(map[string]bool)

	for _, rec := range recommendations {
		byCategory[rec.Category]++
		bySeverity[rec.Severity]++
		if rec.ClusterName != "" {
			clustersMap[rec.ClusterName] = true
		}
	}

	return AKSReportSummary{
		TotalRecommendations: len(recommendations),
		ByCategory:           byCategory,
		BySeverity:           bySeverity,
		ClustersAffected:     len(clustersMap),
	}
}

// generateActionItems creates prioritized action items from recommendations
func generateActionItems(recommendations []AKSRecommendationSummary) []AKSActionItem {
	var actionItems []AKSActionItem
	priority := 1

	// Sort by severity (High > Medium > Low)
	highPriority := filterRecommendationsBySeverity(recommendations, "High")
	mediumPriority := filterRecommendationsBySeverity(recommendations, "Medium")
	lowPriority := filterRecommendationsBySeverity(recommendations, "Low")

	// Create action items in priority order
	for _, rec := range append(append(highPriority, mediumPriority...), lowPriority...) {
		actionItems = append(actionItems, AKSActionItem{
			Priority:         priority,
			RecommendationID: rec.ID,
			ClusterName:      rec.ClusterName,
			Category:         rec.Category,
			Description:      rec.Description,
			EstimatedEffort:  mapSeverityToEffort(rec.Severity),
			PotentialImpact:  rec.Severity,
		})
		priority++
	}

	return actionItems
}

// filterRecommendationsBySeverity filters recommendations by severity
func filterRecommendationsBySeverity(recommendations []AKSRecommendationSummary, severity string) []AKSRecommendationSummary {
	var filtered []AKSRecommendationSummary
	for _, rec := range recommendations {
		if strings.EqualFold(rec.Severity, severity) {
			filtered = append(filtered, rec)
		}
	}
	return filtered
}

// mapSeverityToEffort maps severity levels to estimated effort
func mapSeverityToEffort(severity string) string {
	switch strings.ToLower(severity) {
	case "high":
		return "Medium"
	case "medium":
		return "Low"
	case "low":
		return "Minimal"
	default:
		return "Unknown"
	}
}

// groupRecommendationsByCluster groups recommendations by cluster name
func groupRecommendationsByCluster(recommendations []AKSRecommendationSummary) []ClusterRecommendations {
	clusterMap := make(map[string][]AKSRecommendationSummary)
	rgMap := make(map[string]string)

	for _, rec := range recommendations {
		if rec.ClusterName != "" {
			clusterMap[rec.ClusterName] = append(clusterMap[rec.ClusterName], rec)
			rgMap[rec.ClusterName] = rec.ResourceGroup
		}
	}

	var clusterBreakdown []ClusterRecommendations
	for clusterName, recs := range clusterMap {
		clusterBreakdown = append(clusterBreakdown, ClusterRecommendations{
			ClusterName:     clusterName,
			ResourceGroup:   rgMap[clusterName],
			Recommendations: recs,
		})
	}

	return clusterBreakdown
}
