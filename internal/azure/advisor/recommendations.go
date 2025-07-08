package advisor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// RecommendationManager manages Azure Advisor recommendation operations
type RecommendationManager struct {
	client *AdvisorClient
}

// NewRecommendationManager creates a new recommendation manager
func NewRecommendationManager(subscriptionID string, credential *azidentity.DefaultAzureCredential) (*RecommendationManager, error) {
	client, err := NewAdvisorClient(subscriptionID, credential)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure Advisor client: %w", err)
	}

	return &RecommendationManager{
		client: client,
	}, nil
}

// GetRecommendations returns formatted Azure Advisor recommendations
func (rm *RecommendationManager) GetRecommendations(ctx context.Context, subscriptionID, resourceGroup string, categories, severities []string) (string, error) {
	filter := &RecommendationFilter{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
	}

	// Convert string arrays to typed arrays
	if len(categories) > 0 {
		for _, cat := range categories {
			filter.Category = append(filter.Category, RecommendationCategory(cat))
		}
	}

	if len(severities) > 0 {
		for _, sev := range severities {
			filter.Severity = append(filter.Severity, RecommendationSeverity(sev))
		}
	}

	recommendations, err := rm.client.GetRecommendations(ctx, filter)
	if err != nil {
		return "", fmt.Errorf("failed to get recommendations: %w", err)
	}

	// Format the response for better readability
	result := formatRecommendationsResponse(subscriptionID, resourceGroup, recommendations, filter)

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal recommendations response: %w", err)
	}

	return string(resultJSON), nil
}

// GetRecommendationDetails returns detailed information about a specific recommendation
func (rm *RecommendationManager) GetRecommendationDetails(ctx context.Context, recommendationID string, includeImplementationStatus bool) (string, error) {
	if recommendationID == "" {
		return "", fmt.Errorf("recommendation ID cannot be empty")
	}

	details, err := rm.client.GetRecommendationDetails(ctx, recommendationID, includeImplementationStatus)
	if err != nil {
		return "", fmt.Errorf("failed to get recommendation details: %w", err)
	}

	// Format the response for better readability
	result := formatRecommendationDetailsResponse(details, includeImplementationStatus)

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal recommendation details response: %w", err)
	}

	return string(resultJSON), nil
}

// formatRecommendationsResponse formats the recommendations response for better readability
func formatRecommendationsResponse(subscriptionID, resourceGroup string, recommendations []AdvisorRecommendation, filter *RecommendationFilter) map[string]interface{} {
	result := map[string]interface{}{
		"subscriptionId":      subscriptionID,
		"resourceGroup":       resourceGroup,
		"requestTime":         time.Now().UTC().Format(time.RFC3339),
		"recommendationCount": len(recommendations),
		"recommendations":     recommendations,
	}

	if filter != nil {
		result["appliedFilters"] = formatAppliedFilters(filter)
	}

	if len(recommendations) > 0 {
		// Add summary
		summary := generateRecommendationSummary(recommendations)
		result["summary"] = summary

		// Add high-priority recommendations
		highPriority := filterHighPriorityRecommendations(recommendations)
		if len(highPriority) > 0 {
			result["highPriorityRecommendations"] = map[string]interface{}{
				"count":           len(highPriority),
				"recommendations": highPriority,
			}
		}

		// Add cost recommendations with savings
		costRecommendations := filterCostRecommendations(recommendations)
		if len(costRecommendations) > 0 {
			totalSavings := calculateTotalSavings(costRecommendations)
			result["costOptimization"] = map[string]interface{}{
				"count":                 len(costRecommendations),
				"recommendations":       costRecommendations,
				"estimatedTotalSavings": totalSavings,
			}
		}

		// Add security recommendations
		securityRecommendations := filterSecurityRecommendations(recommendations)
		if len(securityRecommendations) > 0 {
			result["securityRecommendations"] = map[string]interface{}{
				"count":           len(securityRecommendations),
				"recommendations": securityRecommendations,
			}
		}
	}

	return result
}

// formatRecommendationDetailsResponse formats the recommendation details response
func formatRecommendationDetailsResponse(details *RecommendationDetails, includeImplementationStatus bool) map[string]interface{} {
	result := map[string]interface{}{
		"recommendationId":            details.Recommendation.ID,
		"requestTime":                 time.Now().UTC().Format(time.RFC3339),
		"includeImplementationStatus": includeImplementationStatus,
		"recommendation":              details.Recommendation,
		"implementationRisk":          details.ImplementationRisk,
		"businessImpact":              details.BusinessImpact,
	}

	if len(details.RelatedResources) > 0 {
		result["relatedResources"] = details.RelatedResources
	}

	if len(details.TechnicalDetails) > 0 {
		result["technicalDetails"] = details.TechnicalDetails
	}

	if len(details.SimilarRecommendations) > 0 {
		result["similarRecommendations"] = details.SimilarRecommendations
	}

	// Add implementation summary
	if len(details.Recommendation.ImplementationGuide) > 0 {
		result["implementationSummary"] = generateImplementationSummary(details.Recommendation.ImplementationGuide)
	}

	return result
}

// generateRecommendationSummary creates a summary of recommendations
func generateRecommendationSummary(recommendations []AdvisorRecommendation) RecommendationSummary {
	summary := RecommendationSummary{
		TotalRecommendations: len(recommendations),
		CategoryCounts:       make(map[RecommendationCategory]int),
		SeverityCounts:       make(map[RecommendationSeverity]int),
		StatusCounts:         make(map[RecommendationStatus]int),
		LastUpdated:          time.Now(),
	}

	var totalSavings float64
	var costRecommendationCount int

	for _, rec := range recommendations {
		// Count by category
		summary.CategoryCounts[rec.Category]++

		// Count by severity
		summary.SeverityCounts[rec.Severity]++

		// Count by status
		summary.StatusCounts[rec.Status]++

		// Count high priority
		if rec.Severity == RecommendationSeverityHigh {
			summary.HighPriorityCount++
		}

		// Calculate savings
		if rec.EstimatedSavings != nil {
			totalSavings += rec.EstimatedSavings.Amount
			costRecommendationCount++
		}
	}

	// Generate top categories
	summary.TopCategories = generateTopCategories(summary.CategoryCounts, recommendations)

	// Add estimated savings if any cost recommendations exist
	if costRecommendationCount > 0 {
		summary.EstimatedSavings = &TotalEstimatedSavings{
			Currency:            "USD",
			TotalAmount:         totalSavings,
			Unit:                "Monthly",
			CostRecommendations: costRecommendationCount,
		}
	}

	return summary
}

// generateTopCategories creates a list of top categories by count
func generateTopCategories(categoryCounts map[RecommendationCategory]int, recommendations []AdvisorRecommendation) []CategorySummary {
	var topCategories []CategorySummary

	for category, count := range categoryCounts {
		if count == 0 {
			continue
		}

		categorySummary := CategorySummary{
			Category: category,
			Count:    count,
		}

		// Count high severity in this category
		var highSeverityCount int
		var categorySavings float64

		for _, rec := range recommendations {
			if rec.Category == category {
				if rec.Severity == RecommendationSeverityHigh {
					highSeverityCount++
				}
				if rec.EstimatedSavings != nil {
					categorySavings += rec.EstimatedSavings.Amount
				}
			}
		}

		categorySummary.HighSeverityCount = highSeverityCount

		if categorySavings > 0 {
			categorySummary.EstimatedSavings = &EstimatedSavings{
				Currency:        "USD",
				Amount:          categorySavings,
				Unit:            "Monthly",
				ConfidenceLevel: "Medium",
			}
		}

		topCategories = append(topCategories, categorySummary)
	}

	return topCategories
}

// formatAppliedFilters formats the applied filters for display
func formatAppliedFilters(filter *RecommendationFilter) map[string]interface{} {
	appliedFilters := make(map[string]interface{})

	if filter.ResourceGroup != "" {
		appliedFilters["resourceGroup"] = filter.ResourceGroup
	}

	if len(filter.Category) > 0 {
		categories := make([]string, len(filter.Category))
		for i, cat := range filter.Category {
			categories[i] = string(cat)
		}
		appliedFilters["categories"] = categories
	}

	if len(filter.Severity) > 0 {
		severities := make([]string, len(filter.Severity))
		for i, sev := range filter.Severity {
			severities[i] = string(sev)
		}
		appliedFilters["severities"] = severities
	}

	if filter.ResourceType != "" {
		appliedFilters["resourceType"] = filter.ResourceType
	}

	return appliedFilters
}

// filterHighPriorityRecommendations filters recommendations to high priority ones
func filterHighPriorityRecommendations(recommendations []AdvisorRecommendation) []AdvisorRecommendation {
	var highPriority []AdvisorRecommendation

	for _, rec := range recommendations {
		if rec.Severity == RecommendationSeverityHigh {
			highPriority = append(highPriority, rec)
		}
	}

	return highPriority
}

// filterCostRecommendations filters recommendations to cost optimization ones
func filterCostRecommendations(recommendations []AdvisorRecommendation) []AdvisorRecommendation {
	var costRecs []AdvisorRecommendation

	for _, rec := range recommendations {
		if rec.Category == RecommendationCategoryCost {
			costRecs = append(costRecs, rec)
		}
	}

	return costRecs
}

// filterSecurityRecommendations filters recommendations to security ones
func filterSecurityRecommendations(recommendations []AdvisorRecommendation) []AdvisorRecommendation {
	var securityRecs []AdvisorRecommendation

	for _, rec := range recommendations {
		if rec.Category == RecommendationCategorySecurity {
			securityRecs = append(securityRecs, rec)
		}
	}

	return securityRecs
}

// calculateTotalSavings calculates the total estimated savings
func calculateTotalSavings(recommendations []AdvisorRecommendation) *TotalEstimatedSavings {
	var totalAmount float64
	count := 0

	for _, rec := range recommendations {
		if rec.EstimatedSavings != nil {
			totalAmount += rec.EstimatedSavings.Amount
			count++
		}
	}

	if count == 0 {
		return nil
	}

	return &TotalEstimatedSavings{
		Currency:            "USD",
		TotalAmount:         totalAmount,
		Unit:                "Monthly",
		CostRecommendations: count,
	}
}

// generateImplementationSummary creates a summary of implementation steps
func generateImplementationSummary(steps []ImplementationStep) map[string]interface{} {
	summary := map[string]interface{}{
		"totalSteps":    len(steps),
		"requiredSteps": 0,
		"effortLevels": map[string]int{
			"Low":    0,
			"Medium": 0,
			"High":   0,
		},
	}

	effortLevels := summary["effortLevels"].(map[string]int)

	for _, step := range steps {
		if step.IsRequired {
			summary["requiredSteps"] = summary["requiredSteps"].(int) + 1
		}
		effortLevels[step.Effort]++
	}

	// Determine overall effort
	if effortLevels["High"] > 0 {
		summary["overallEffort"] = "High"
	} else if effortLevels["Medium"] > 0 {
		summary["overallEffort"] = "Medium"
	} else {
		summary["overallEffort"] = "Low"
	}

	return summary
}

// ValidateRecommendationID validates that the provided recommendation ID is valid
func ValidateRecommendationID(recommendationID string) error {
	if recommendationID == "" {
		return fmt.Errorf("recommendation ID cannot be empty")
	}

	// Basic validation for recommendation ID format
	// Expected format: /subscriptions/{subscription-id}/recommendations/{recommendation-name}
	if !strings.HasPrefix(recommendationID, "/subscriptions/") {
		return fmt.Errorf("invalid recommendation ID format: must start with /subscriptions/")
	}

	parts := strings.Split(recommendationID, "/")
	if len(parts) < 4 {
		return fmt.Errorf("invalid recommendation ID format: too few segments")
	}

	if parts[3] != "recommendations" {
		return fmt.Errorf("invalid recommendation ID format: missing recommendations segment")
	}

	return nil
}

// ValidateFilterParameters validates filter parameters
func ValidateFilterParameters(categories, severities []string) error {
	// Validate categories
	validCategories := map[string]bool{
		string(RecommendationCategoryCost):        true,
		string(RecommendationCategoryPerformance): true,
		string(RecommendationCategorySecurity):    true,
		string(RecommendationCategoryReliability): true,
		string(RecommendationCategoryOperational): true,
	}

	for _, cat := range categories {
		if !validCategories[cat] {
			return fmt.Errorf("invalid category '%s': must be one of Cost, Performance, Security, Reliability, Operational", cat)
		}
	}

	// Validate severities
	validSeverities := map[string]bool{
		string(RecommendationSeverityHigh):   true,
		string(RecommendationSeverityMedium): true,
		string(RecommendationSeverityLow):    true,
	}

	for _, sev := range severities {
		if !validSeverities[sev] {
			return fmt.Errorf("invalid severity '%s': must be one of High, Medium, Low", sev)
		}
	}

	return nil
}
