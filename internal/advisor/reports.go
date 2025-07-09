package advisor

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// GenerateExecutiveSummary creates an executive summary of AKS recommendations
func GenerateExecutiveSummary(report AKSAdvisorReport) (string, error) {
	summary := ExecutiveSummary{
		GeneratedAt:          report.GeneratedAt,
		SubscriptionID:       report.SubscriptionID,
		TotalClusters:        report.Summary.ClustersAffected,
		TotalRecommendations: report.Summary.TotalRecommendations,
		HighPriorityCount:    report.Summary.BySeverity["High"],
		MediumPriorityCount:  report.Summary.BySeverity["Medium"],
		LowPriorityCount:     report.Summary.BySeverity["Low"],
		TopCategories:        getTopCategories(report.Summary.ByCategory),
		KeyFindings:          generateKeyFindings(report),
		NextSteps:            generateNextSteps(report),
	}

	result, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal executive summary: %w", err)
	}

	return string(result), nil
}

// GenerateDetailedReport creates a detailed report with all recommendation information
func GenerateDetailedReport(report AKSAdvisorReport) (string, error) {
	detailedReport := DetailedReport{
		ExecutiveSummary: ExecutiveSummary{
			GeneratedAt:          report.GeneratedAt,
			SubscriptionID:       report.SubscriptionID,
			TotalClusters:        report.Summary.ClustersAffected,
			TotalRecommendations: report.Summary.TotalRecommendations,
			HighPriorityCount:    report.Summary.BySeverity["High"],
			MediumPriorityCount:  report.Summary.BySeverity["Medium"],
			LowPriorityCount:     report.Summary.BySeverity["Low"],
			TopCategories:        getTopCategories(report.Summary.ByCategory),
			KeyFindings:          generateKeyFindings(report),
			NextSteps:            generateNextSteps(report),
		},
		CategoryBreakdown:      generateCategoryBreakdown(report),
		ClusterAnalysis:        generateClusterAnalysis(report),
		PriorityMatrix:         generatePriorityMatrix(report),
		ImplementationTimeline: generateImplementationTimeline(report),
		AllRecommendations:     report.Recommendations,
	}

	result, err := json.MarshalIndent(detailedReport, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal detailed report: %w", err)
	}

	return string(result), nil
}

// GenerateActionableReport creates a focused report with actionable items
func GenerateActionableReport(report AKSAdvisorReport) (string, error) {
	actionableReport := ActionableReport{
		GeneratedAt:          report.GeneratedAt,
		SubscriptionID:       report.SubscriptionID,
		QuickWins:            identifyQuickWins(report),
		HighImpactItems:      identifyHighImpactItems(report),
		CostOptimization:     identifyCostOptimization(report),
		SecurityImprovements: identifySecurityImprovements(report),
		PerformanceBoosts:    identifyPerformanceBoosts(report),
		ImplementationGuide:  generateImplementationGuide(report),
	}

	result, err := json.MarshalIndent(actionableReport, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal actionable report: %w", err)
	}

	return string(result), nil
}

// Report structure types
type ExecutiveSummary struct {
	GeneratedAt          time.Time       `json:"generated_at"`
	SubscriptionID       string          `json:"subscription_id"`
	TotalClusters        int             `json:"total_clusters"`
	TotalRecommendations int             `json:"total_recommendations"`
	HighPriorityCount    int             `json:"high_priority_count"`
	MediumPriorityCount  int             `json:"medium_priority_count"`
	LowPriorityCount     int             `json:"low_priority_count"`
	TopCategories        []CategoryCount `json:"top_categories"`
	KeyFindings          []string        `json:"key_findings"`
	NextSteps            []string        `json:"next_steps"`
}

type DetailedReport struct {
	ExecutiveSummary       ExecutiveSummary           `json:"executive_summary"`
	CategoryBreakdown      []CategoryBreakdown        `json:"category_breakdown"`
	ClusterAnalysis        []ClusterAnalysis          `json:"cluster_analysis"`
	PriorityMatrix         []PriorityMatrixItem       `json:"priority_matrix"`
	ImplementationTimeline []TimelineItem             `json:"implementation_timeline"`
	AllRecommendations     []AKSRecommendationSummary `json:"all_recommendations"`
}

type ActionableReport struct {
	GeneratedAt          time.Time           `json:"generated_at"`
	SubscriptionID       string              `json:"subscription_id"`
	QuickWins            []ActionableItem    `json:"quick_wins"`
	HighImpactItems      []ActionableItem    `json:"high_impact_items"`
	CostOptimization     []ActionableItem    `json:"cost_optimization"`
	SecurityImprovements []ActionableItem    `json:"security_improvements"`
	PerformanceBoosts    []ActionableItem    `json:"performance_boosts"`
	ImplementationGuide  ImplementationGuide `json:"implementation_guide"`
}

type CategoryCount struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

type CategoryBreakdown struct {
	Category        string                     `json:"category"`
	Count           int                        `json:"count"`
	Recommendations []AKSRecommendationSummary `json:"recommendations"`
	Impact          string                     `json:"impact"`
}

type ClusterAnalysis struct {
	ClusterName         string   `json:"cluster_name"`
	ResourceGroup       string   `json:"resource_group"`
	RecommendationCount int      `json:"recommendation_count"`
	HighPriorityCount   int      `json:"high_priority_count"`
	PrimaryCategories   []string `json:"primary_categories"`
	OverallRisk         string   `json:"overall_risk"`
}

type PriorityMatrixItem struct {
	RecommendationID string `json:"recommendation_id"`
	ClusterName      string `json:"cluster_name"`
	Category         string `json:"category"`
	Impact           string `json:"impact"`
	Effort           string `json:"effort"`
	Priority         int    `json:"priority"`
}

type TimelineItem struct {
	Week              int      `json:"week"`
	RecommendationIDs []string `json:"recommendation_ids"`
	Focus             string   `json:"focus"`
	EstimatedHours    int      `json:"estimated_hours"`
}

type ActionableItem struct {
	RecommendationID string   `json:"recommendation_id"`
	ClusterName      string   `json:"cluster_name"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Steps            []string `json:"steps"`
	ExpectedOutcome  string   `json:"expected_outcome"`
	TimeEstimate     string   `json:"time_estimate"`
}

type ImplementationGuide struct {
	Phase1 ImplementationPhase `json:"phase_1"`
	Phase2 ImplementationPhase `json:"phase_2"`
	Phase3 ImplementationPhase `json:"phase_3"`
}

type ImplementationPhase struct {
	Name           string           `json:"name"`
	Duration       string           `json:"duration"`
	Focus          string           `json:"focus"`
	Actions        []ActionableItem `json:"actions"`
	Prerequisites  []string         `json:"prerequisites"`
	SuccessMetrics []string         `json:"success_metrics"`
}

// Helper functions for report generation

func getTopCategories(categoryMap map[string]int) []CategoryCount {
	var categories []CategoryCount
	for category, count := range categoryMap {
		categories = append(categories, CategoryCount{
			Category: category,
			Count:    count,
		})
	}

	// Sort by count descending
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Count > categories[j].Count
	})

	// Return top 5 categories
	if len(categories) > 5 {
		categories = categories[:5]
	}

	return categories
}

func generateKeyFindings(report AKSAdvisorReport) []string {
	var findings []string

	if report.Summary.TotalRecommendations == 0 {
		findings = append(findings, "No Azure Advisor recommendations found for AKS clusters in this subscription")
		return findings
	}

	if report.Summary.BySeverity["High"] > 0 {
		findings = append(findings, fmt.Sprintf("%d high-priority recommendations require immediate attention", report.Summary.BySeverity["High"]))
	}

	if report.Summary.ClustersAffected > 0 {
		findings = append(findings, fmt.Sprintf("%d AKS clusters have recommendations for optimization", report.Summary.ClustersAffected))
	}

	// Find most common category
	maxCount := 0
	topCategory := ""
	for category, count := range report.Summary.ByCategory {
		if count > maxCount {
			maxCount = count
			topCategory = category
		}
	}

	if topCategory != "" {
		findings = append(findings, fmt.Sprintf("Most recommendations focus on %s (%d recommendations)", topCategory, maxCount))
	}

	return findings
}

func generateNextSteps(report AKSAdvisorReport) []string {
	var steps []string

	if report.Summary.BySeverity["High"] > 0 {
		steps = append(steps, "Address high-priority recommendations first")
	}

	if report.Summary.ByCategory["Cost"] > 0 {
		steps = append(steps, "Review cost optimization opportunities")
	}

	if report.Summary.ByCategory["Security"] > 0 {
		steps = append(steps, "Implement security recommendations to improve cluster security posture")
	}

	steps = append(steps, "Schedule regular review of Azure Advisor recommendations")
	steps = append(steps, "Set up Azure Advisor alerts for new recommendations")

	return steps
}

func generateCategoryBreakdown(report AKSAdvisorReport) []CategoryBreakdown {
	categoryMap := make(map[string][]AKSRecommendationSummary)

	for _, rec := range report.Recommendations {
		categoryMap[rec.Category] = append(categoryMap[rec.Category], rec)
	}

	var breakdown []CategoryBreakdown
	for category, recs := range categoryMap {
		impact := "Medium"
		if len(recs) > 5 {
			impact = "High"
		} else if len(recs) <= 2 {
			impact = "Low"
		}

		breakdown = append(breakdown, CategoryBreakdown{
			Category:        category,
			Count:           len(recs),
			Recommendations: recs,
			Impact:          impact,
		})
	}

	// Sort by count descending
	sort.Slice(breakdown, func(i, j int) bool {
		return breakdown[i].Count > breakdown[j].Count
	})

	return breakdown
}

func generateClusterAnalysis(report AKSAdvisorReport) []ClusterAnalysis {
	var analysis []ClusterAnalysis

	for _, cluster := range report.ClusterBreakdown {
		highPriorityCount := 0
		categoryMap := make(map[string]int)

		for _, rec := range cluster.Recommendations {
			if strings.EqualFold(rec.Severity, "High") {
				highPriorityCount++
			}
			categoryMap[rec.Category]++
		}

		// Get primary categories
		var primaryCategories []string
		for category := range categoryMap {
			primaryCategories = append(primaryCategories, category)
		}

		// Determine overall risk
		overallRisk := "Low"
		if highPriorityCount > 3 {
			overallRisk = "High"
		} else if highPriorityCount > 1 {
			overallRisk = "Medium"
		}

		analysis = append(analysis, ClusterAnalysis{
			ClusterName:         cluster.ClusterName,
			ResourceGroup:       cluster.ResourceGroup,
			RecommendationCount: len(cluster.Recommendations),
			HighPriorityCount:   highPriorityCount,
			PrimaryCategories:   primaryCategories,
			OverallRisk:         overallRisk,
		})
	}

	return analysis
}

func generatePriorityMatrix(report AKSAdvisorReport) []PriorityMatrixItem {
	var matrix []PriorityMatrixItem

	for i, rec := range report.Recommendations {
		effort := mapSeverityToEffort(rec.Severity)

		matrix = append(matrix, PriorityMatrixItem{
			RecommendationID: rec.ID,
			ClusterName:      rec.ClusterName,
			Category:         rec.Category,
			Impact:           rec.Severity,
			Effort:           effort,
			Priority:         i + 1,
		})
	}

	return matrix
}

func generateImplementationTimeline(report AKSAdvisorReport) []TimelineItem {
	var timeline []TimelineItem
	week := 1

	// Group high priority items for week 1
	var highPriorityIDs []string
	for _, rec := range report.Recommendations {
		if strings.EqualFold(rec.Severity, "High") {
			highPriorityIDs = append(highPriorityIDs, rec.ID)
		}
	}

	if len(highPriorityIDs) > 0 {
		timeline = append(timeline, TimelineItem{
			Week:              week,
			RecommendationIDs: highPriorityIDs,
			Focus:             "High Priority Items",
			EstimatedHours:    len(highPriorityIDs) * 4,
		})
		week++
	}

	// Group medium priority items for subsequent weeks
	var mediumPriorityIDs []string
	for _, rec := range report.Recommendations {
		if strings.EqualFold(rec.Severity, "Medium") {
			mediumPriorityIDs = append(mediumPriorityIDs, rec.ID)
		}
	}

	if len(mediumPriorityIDs) > 0 {
		timeline = append(timeline, TimelineItem{
			Week:              week,
			RecommendationIDs: mediumPriorityIDs,
			Focus:             "Medium Priority Items",
			EstimatedHours:    len(mediumPriorityIDs) * 2,
		})
	}

	return timeline
}

func identifyQuickWins(report AKSAdvisorReport) []ActionableItem {
	var quickWins []ActionableItem

	for _, rec := range report.Recommendations {
		if strings.EqualFold(rec.Severity, "Low") || strings.Contains(strings.ToLower(rec.Description), "configuration") {
			quickWins = append(quickWins, ActionableItem{
				RecommendationID: rec.ID,
				ClusterName:      rec.ClusterName,
				Title:            "Quick Configuration Fix",
				Description:      rec.Description,
				Steps:            []string{"Review current configuration", "Apply recommended changes", "Validate changes"},
				ExpectedOutcome:  "Improved cluster configuration",
				TimeEstimate:     "30 minutes",
			})
		}
	}

	return quickWins
}

func identifyHighImpactItems(report AKSAdvisorReport) []ActionableItem {
	var highImpact []ActionableItem

	for _, rec := range report.Recommendations {
		if strings.EqualFold(rec.Severity, "High") {
			highImpact = append(highImpact, ActionableItem{
				RecommendationID: rec.ID,
				ClusterName:      rec.ClusterName,
				Title:            "High Impact Improvement",
				Description:      rec.Description,
				Steps:            generateStepsForCategory(rec.Category),
				ExpectedOutcome:  "Significant improvement in " + strings.ToLower(rec.Category),
				TimeEstimate:     "2-4 hours",
			})
		}
	}

	return highImpact
}

func identifyCostOptimization(report AKSAdvisorReport) []ActionableItem {
	var costItems []ActionableItem

	for _, rec := range report.Recommendations {
		if strings.EqualFold(rec.Category, "Cost") {
			costItems = append(costItems, ActionableItem{
				RecommendationID: rec.ID,
				ClusterName:      rec.ClusterName,
				Title:            "Cost Optimization",
				Description:      rec.Description,
				Steps:            []string{"Analyze current resource usage", "Implement recommended changes", "Monitor cost impact"},
				ExpectedOutcome:  "Reduced Azure costs",
				TimeEstimate:     "1-2 hours",
			})
		}
	}

	return costItems
}

func identifySecurityImprovements(report AKSAdvisorReport) []ActionableItem {
	var securityItems []ActionableItem

	for _, rec := range report.Recommendations {
		if strings.EqualFold(rec.Category, "Security") {
			securityItems = append(securityItems, ActionableItem{
				RecommendationID: rec.ID,
				ClusterName:      rec.ClusterName,
				Title:            "Security Enhancement",
				Description:      rec.Description,
				Steps:            []string{"Review security configuration", "Apply security recommendations", "Validate security posture"},
				ExpectedOutcome:  "Improved cluster security",
				TimeEstimate:     "1-3 hours",
			})
		}
	}

	return securityItems
}

func identifyPerformanceBoosts(report AKSAdvisorReport) []ActionableItem {
	var performanceItems []ActionableItem

	for _, rec := range report.Recommendations {
		if strings.EqualFold(rec.Category, "Performance") {
			performanceItems = append(performanceItems, ActionableItem{
				RecommendationID: rec.ID,
				ClusterName:      rec.ClusterName,
				Title:            "Performance Optimization",
				Description:      rec.Description,
				Steps:            []string{"Benchmark current performance", "Apply performance recommendations", "Measure performance improvements"},
				ExpectedOutcome:  "Better cluster performance",
				TimeEstimate:     "2-4 hours",
			})
		}
	}

	return performanceItems
}

func generateImplementationGuide(report AKSAdvisorReport) ImplementationGuide {
	quickWins := identifyQuickWins(report)
	highImpact := identifyHighImpactItems(report)
	costOptimization := identifyCostOptimization(report)

	return ImplementationGuide{
		Phase1: ImplementationPhase{
			Name:           "Quick Wins",
			Duration:       "Week 1",
			Focus:          "Low-effort, high-value improvements",
			Actions:        quickWins,
			Prerequisites:  []string{"Cluster access", "Configuration review permissions"},
			SuccessMetrics: []string{"Number of quick wins implemented", "Configuration compliance improvement"},
		},
		Phase2: ImplementationPhase{
			Name:           "High Impact Items",
			Duration:       "Weeks 2-3",
			Focus:          "Critical improvements requiring more effort",
			Actions:        highImpact,
			Prerequisites:  []string{"Phase 1 completion", "Change management approval"},
			SuccessMetrics: []string{"Reduction in high-priority recommendations", "Improved cluster health scores"},
		},
		Phase3: ImplementationPhase{
			Name:           "Long-term Optimization",
			Duration:       "Weeks 4-6",
			Focus:          "Cost optimization and advanced improvements",
			Actions:        costOptimization,
			Prerequisites:  []string{"Phases 1-2 completion", "Budget approval for changes"},
			SuccessMetrics: []string{"Cost reduction achieved", "Performance improvements measured"},
		},
	}
}

func generateStepsForCategory(category string) []string {
	switch strings.ToLower(category) {
	case "cost":
		return []string{"Analyze resource utilization", "Right-size resources", "Implement cost controls", "Monitor savings"}
	case "security":
		return []string{"Review security policies", "Update configurations", "Enable security features", "Validate security posture"}
	case "performance":
		return []string{"Benchmark current performance", "Apply performance tuning", "Optimize resource allocation", "Monitor improvements"}
	case "highavailability":
		return []string{"Review availability configuration", "Implement redundancy", "Test failover scenarios", "Monitor availability metrics"}
	default:
		return []string{"Review recommendation details", "Plan implementation", "Apply changes", "Validate results"}
	}
}
