package advisor

import (
	"context"
	"testing"
)

func TestValidateRecommendationID(t *testing.T) {
	tests := []struct {
		name           string
		recommendationID string
		shouldError    bool
	}{
		{
			name:           "valid recommendation ID",
			recommendationID: "/subscriptions/12345678-1234-1234-1234-123456789012/recommendations/cost-optimization",
			shouldError:    false,
		},
		{
			name:           "empty recommendation ID",
			recommendationID: "",
			shouldError:    true,
		},
		{
			name:           "invalid format - no subscriptions prefix",
			recommendationID: "recommendations/cost-optimization",
			shouldError:    true,
		},
		{
			name:           "invalid format - too few segments",
			recommendationID: "/subscriptions/12345678-1234-1234-1234-123456789012",
			shouldError:    true,
		},
		{
			name:           "invalid format - wrong segment",
			recommendationID: "/subscriptions/12345678-1234-1234-1234-123456789012/advisors/cost-optimization",
			shouldError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecommendationID(tt.recommendationID)
			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateFilterParameters(t *testing.T) {
	tests := []struct {
		name        string
		categories  []string
		severities  []string
		shouldError bool
	}{
		{
			name:        "valid categories and severities",
			categories:  []string{"Cost", "Security"},
			severities:  []string{"High", "Medium"},
			shouldError: false,
		},
		{
			name:        "empty filters",
			categories:  []string{},
			severities:  []string{},
			shouldError: false,
		},
		{
			name:        "invalid category",
			categories:  []string{"InvalidCategory"},
			severities:  []string{"High"},
			shouldError: true,
		},
		{
			name:        "invalid severity",
			categories:  []string{"Cost"},
			severities:  []string{"InvalidSeverity"},
			shouldError: true,
		},
		{
			name:        "mixed valid and invalid categories",
			categories:  []string{"Cost", "InvalidCategory"},
			severities:  []string{"High"},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilterParameters(tt.categories, tt.severities)
			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewAdvisorClient(t *testing.T) {
	// Test with empty subscription ID
	_, err := NewAdvisorClient("", nil)
	if err == nil {
		t.Error("expected error for empty subscription ID")
	}

	// Test with valid subscription ID
	client, err := NewAdvisorClient("12345678-1234-1234-1234-123456789012", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if client == nil {
		t.Error("expected client to be created")
	}
}

func TestNewRecommendationManager(t *testing.T) {
	// Test with empty subscription ID
	_, err := NewRecommendationManager("", nil)
	if err == nil {
		t.Error("expected error for empty subscription ID")
	}

	// Test with valid subscription ID
	manager, err := NewRecommendationManager("12345678-1234-1234-1234-123456789012", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if manager == nil {
		t.Error("expected recommendation manager to be created")
	}
}

func TestGenerateMockRecommendations(t *testing.T) {
	subscriptionID := "12345678-1234-1234-1234-123456789012"

	// Test without filter
	recommendations := generateMockRecommendations(subscriptionID, nil)
	if len(recommendations) == 0 {
		t.Error("expected at least one recommendation to be generated")
	}

	// Verify basic structure
	for _, rec := range recommendations {
		if rec.ID == "" {
			t.Error("expected recommendation ID to be set")
		}
		if rec.Title == "" {
			t.Error("expected recommendation title to be set")
		}
		if rec.Category == "" {
			t.Error("expected recommendation category to be set")
		}
		if rec.Severity == "" {
			t.Error("expected recommendation severity to be set")
		}
		if rec.SubscriptionID != subscriptionID {
			t.Errorf("expected subscription ID %s, got %s", subscriptionID, rec.SubscriptionID)
		}
	}

	// Test with filter
	filter := &RecommendationFilter{
		Category: []RecommendationCategory{RecommendationCategoryCost},
		Severity: []RecommendationSeverity{RecommendationSeverityHigh},
	}
	filteredRecs := generateMockRecommendations(subscriptionID, filter)
	if len(filteredRecs) == 0 {
		t.Error("expected at least one recommendation after filtering")
	}
}

func TestApplyRecommendationFilters(t *testing.T) {
	recommendations := []AdvisorRecommendation{
		{
			Category:      RecommendationCategoryCost,
			Severity:      RecommendationSeverityHigh,
			Status:        RecommendationStatusActive,
			ResourceGroup: "rg1",
			ResourceType:  "Microsoft.Compute/virtualMachines",
		},
		{
			Category:      RecommendationCategorySecurity,
			Severity:      RecommendationSeverityMedium,
			Status:        RecommendationStatusActive,
			ResourceGroup: "rg2",
			ResourceType:  "Microsoft.Network/networkSecurityGroups",
		},
		{
			Category:      RecommendationCategoryPerformance,
			Severity:      RecommendationSeverityLow,
			Status:        RecommendationStatusDismissed,
			ResourceGroup: "rg1",
			ResourceType:  "Microsoft.ContainerService/managedClusters",
		},
	}

	// Test category filter
	filter := &RecommendationFilter{
		Category: []RecommendationCategory{RecommendationCategoryCost},
	}
	filtered := applyRecommendationFilters(recommendations, filter)
	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered recommendation, got %d", len(filtered))
	}
	if filtered[0].Category != RecommendationCategoryCost {
		t.Errorf("expected Cost category, got %s", filtered[0].Category)
	}

	// Test severity filter
	filter = &RecommendationFilter{
		Severity: []RecommendationSeverity{RecommendationSeverityHigh, RecommendationSeverityMedium},
	}
	filtered = applyRecommendationFilters(recommendations, filter)
	if len(filtered) != 2 {
		t.Errorf("expected 2 filtered recommendations, got %d", len(filtered))
	}

	// Test resource group filter
	filter = &RecommendationFilter{
		ResourceGroup: "rg1",
	}
	filtered = applyRecommendationFilters(recommendations, filter)
	if len(filtered) != 2 {
		t.Errorf("expected 2 filtered recommendations, got %d", len(filtered))
	}

	// Test no filter
	filtered = applyRecommendationFilters(recommendations, nil)
	if len(filtered) != len(recommendations) {
		t.Errorf("expected %d recommendations with no filter, got %d", len(recommendations), len(filtered))
	}
}

func TestGenerateRecommendationSummary(t *testing.T) {
	recommendations := []AdvisorRecommendation{
		{
			Category: RecommendationCategoryCost,
			Severity: RecommendationSeverityHigh,
			Status:   RecommendationStatusActive,
			EstimatedSavings: &EstimatedSavings{
				Amount:   100.50,
				Currency: "USD",
				Unit:     "Monthly",
			},
		},
		{
			Category: RecommendationCategorySecurity,
			Severity: RecommendationSeverityHigh,
			Status:   RecommendationStatusActive,
		},
		{
			Category: RecommendationCategoryPerformance,
			Severity: RecommendationSeverityMedium,
			Status:   RecommendationStatusDismissed,
		},
	}

	summary := generateRecommendationSummary(recommendations)

	if summary.TotalRecommendations != 3 {
		t.Errorf("expected total recommendations 3, got %d", summary.TotalRecommendations)
	}

	if summary.CategoryCounts[RecommendationCategoryCost] != 1 {
		t.Errorf("expected 1 cost recommendation, got %d", summary.CategoryCounts[RecommendationCategoryCost])
	}

	if summary.SeverityCounts[RecommendationSeverityHigh] != 2 {
		t.Errorf("expected 2 high severity recommendations, got %d", summary.SeverityCounts[RecommendationSeverityHigh])
	}

	if summary.HighPriorityCount != 2 {
		t.Errorf("expected 2 high priority recommendations, got %d", summary.HighPriorityCount)
	}

	if summary.EstimatedSavings == nil {
		t.Error("expected estimated savings to be calculated")
	} else {
		if summary.EstimatedSavings.TotalAmount != 100.50 {
			t.Errorf("expected total savings 100.50, got %f", summary.EstimatedSavings.TotalAmount)
		}
		if summary.EstimatedSavings.CostRecommendations != 1 {
			t.Errorf("expected 1 cost recommendation, got %d", summary.EstimatedSavings.CostRecommendations)
		}
	}
}

func TestFilterHighPriorityRecommendations(t *testing.T) {
	recommendations := []AdvisorRecommendation{
		{Severity: RecommendationSeverityHigh},
		{Severity: RecommendationSeverityMedium},
		{Severity: RecommendationSeverityHigh},
		{Severity: RecommendationSeverityLow},
	}

	highPriority := filterHighPriorityRecommendations(recommendations)
	if len(highPriority) != 2 {
		t.Errorf("expected 2 high priority recommendations, got %d", len(highPriority))
	}

	for _, rec := range highPriority {
		if rec.Severity != RecommendationSeverityHigh {
			t.Errorf("expected high severity, got %s", rec.Severity)
		}
	}
}

func TestFilterCostRecommendations(t *testing.T) {
	recommendations := []AdvisorRecommendation{
		{Category: RecommendationCategoryCost},
		{Category: RecommendationCategorySecurity},
		{Category: RecommendationCategoryCost},
		{Category: RecommendationCategoryPerformance},
	}

	costRecs := filterCostRecommendations(recommendations)
	if len(costRecs) != 2 {
		t.Errorf("expected 2 cost recommendations, got %d", len(costRecs))
	}

	for _, rec := range costRecs {
		if rec.Category != RecommendationCategoryCost {
			t.Errorf("expected cost category, got %s", rec.Category)
		}
	}
}

func TestCalculateTotalSavings(t *testing.T) {
	recommendations := []AdvisorRecommendation{
		{
			EstimatedSavings: &EstimatedSavings{Amount: 100.50},
		},
		{
			EstimatedSavings: &EstimatedSavings{Amount: 50.25},
		},
		{
			EstimatedSavings: nil, // No savings
		},
	}

	totalSavings := calculateTotalSavings(recommendations)
	if totalSavings == nil {
		t.Fatal("expected total savings to be calculated")
	}

	if totalSavings.TotalAmount != 150.75 {
		t.Errorf("expected total amount 150.75, got %f", totalSavings.TotalAmount)
	}

	if totalSavings.CostRecommendations != 2 {
		t.Errorf("expected 2 cost recommendations, got %d", totalSavings.CostRecommendations)
	}

	// Test with no savings
	noSavingsRecs := []AdvisorRecommendation{
		{EstimatedSavings: nil},
		{EstimatedSavings: nil},
	}

	noSavings := calculateTotalSavings(noSavingsRecs)
	if noSavings != nil {
		t.Error("expected no savings for recommendations without savings data")
	}
}

func TestGetRecommendations(t *testing.T) {
	// Create recommendation manager
	manager, err := NewRecommendationManager("12345678-1234-1234-1234-123456789012", nil)
	if err != nil {
		t.Fatalf("failed to create recommendation manager: %v", err)
	}

	subscriptionID := "12345678-1234-1234-1234-123456789012"
	resourceGroup := "test-rg"

	// Test getting all recommendations
	result, err := manager.GetRecommendations(context.Background(), subscriptionID, resourceGroup, nil, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Test with category filter
	categories := []string{"Cost", "Security"}
	result, err = manager.GetRecommendations(context.Background(), subscriptionID, resourceGroup, categories, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Test with severity filter
	severities := []string{"High"}
	result, err = manager.GetRecommendations(context.Background(), subscriptionID, resourceGroup, nil, severities)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestGetRecommendationDetails(t *testing.T) {
	// Create recommendation manager
	manager, err := NewRecommendationManager("12345678-1234-1234-1234-123456789012", nil)
	if err != nil {
		t.Fatalf("failed to create recommendation manager: %v", err)
	}

	recommendationID := "/subscriptions/12345678-1234-1234-1234-123456789012/recommendations/cost-optimization"

	// Test getting recommendation details
	result, err := manager.GetRecommendationDetails(context.Background(), recommendationID, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Test with implementation status
	result, err = manager.GetRecommendationDetails(context.Background(), recommendationID, true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Test with empty recommendation ID
	_, err = manager.GetRecommendationDetails(context.Background(), "", false)
	if err == nil {
		t.Error("expected error for empty recommendation ID")
	}
}

func TestGenerateMockRecommendationDetails(t *testing.T) {
	recommendationID := "/subscriptions/12345678-1234-1234-1234-123456789012/recommendations/cost-optimization"

	details := generateMockRecommendationDetails(recommendationID, true)

	if details == nil {
		t.Fatal("expected recommendation details to be generated")
	}

	if details.Recommendation.ID != recommendationID {
		t.Errorf("expected recommendation ID %s, got %s", recommendationID, details.Recommendation.ID)
	}

	if details.Recommendation.Title == "" {
		t.Error("expected recommendation title to be set")
	}

	if len(details.RelatedResources) == 0 {
		t.Error("expected at least one related resource")
	}

	if details.ImplementationRisk.Level == "" {
		t.Error("expected implementation risk level to be set")
	}

	if details.BusinessImpact.Cost == "" {
		t.Error("expected business impact cost to be set")
	}
}

func TestGenerateImplementationSummary(t *testing.T) {
	steps := []ImplementationStep{
		{IsRequired: true, Effort: "High"},
		{IsRequired: true, Effort: "Medium"},
		{IsRequired: false, Effort: "Low"},
		{IsRequired: true, Effort: "Medium"},
	}

	summary := generateImplementationSummary(steps)

	if summary["totalSteps"] != 4 {
		t.Errorf("expected total steps 4, got %v", summary["totalSteps"])
	}

	if summary["requiredSteps"] != 3 {
		t.Errorf("expected required steps 3, got %v", summary["requiredSteps"])
	}

	if summary["overallEffort"] != "High" {
		t.Errorf("expected overall effort High, got %v", summary["overallEffort"])
	}

	effortLevels := summary["effortLevels"].(map[string]int)
	if effortLevels["High"] != 1 {
		t.Errorf("expected 1 high effort step, got %d", effortLevels["High"])
	}
	if effortLevels["Medium"] != 2 {
		t.Errorf("expected 2 medium effort steps, got %d", effortLevels["Medium"])
	}
	if effortLevels["Low"] != 1 {
		t.Errorf("expected 1 low effort step, got %d", effortLevels["Low"])
	}
}