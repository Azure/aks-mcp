package advisor

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// AdvisorClient represents a client for interacting with Azure Advisor APIs
type AdvisorClient struct {
	subscriptionID string
	credential     azcore.TokenCredential
	httpClient     *http.Client
	baseURL        string
}

// NewAdvisorClient creates a new Azure Advisor client
func NewAdvisorClient(subscriptionID string, credential *azidentity.DefaultAzureCredential) (*AdvisorClient, error) {
	if subscriptionID == "" {
		return nil, fmt.Errorf("subscription ID cannot be empty")
	}

	return &AdvisorClient{
		subscriptionID: subscriptionID,
		credential:     credential,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		baseURL:        "https://management.azure.com",
	}, nil
}

// GetRecommendations retrieves Azure Advisor recommendations based on the provided filter
func (c *AdvisorClient) GetRecommendations(ctx context.Context, filter *RecommendationFilter) ([]AdvisorRecommendation, error) {
	// For now, return mock data
	// In a real implementation, this would call the Azure Advisor API
	
	recommendations := generateMockRecommendations(c.subscriptionID, filter)
	
	// Apply filters if provided
	if filter != nil {
		recommendations = applyRecommendationFilters(recommendations, filter)
	}

	return recommendations, nil
}

// GetRecommendationDetails retrieves detailed information about a specific recommendation
func (c *AdvisorClient) GetRecommendationDetails(ctx context.Context, recommendationID string, includeImplementationStatus bool) (*RecommendationDetails, error) {
	if recommendationID == "" {
		return nil, fmt.Errorf("recommendation ID cannot be empty")
	}

	// Generate mock detailed recommendation
	// In a real implementation, this would call the Azure Advisor API
	
	details := generateMockRecommendationDetails(recommendationID, includeImplementationStatus)
	return details, nil
}

// generateMockRecommendations creates mock Azure Advisor recommendations
func generateMockRecommendations(subscriptionID string, filter *RecommendationFilter) []AdvisorRecommendation {
	baseRecommendations := []AdvisorRecommendation{
		{
			ID:             fmt.Sprintf("/subscriptions/%s/recommendations/cost-sku-optimization", subscriptionID),
			Name:           "cost-sku-optimization",
			Type:           "Microsoft.Advisor/recommendations",
			Category:       RecommendationCategoryCost,
			Severity:       RecommendationSeverityHigh,
			Status:         RecommendationStatusActive,
			Title:          "Right-size underutilized virtual machines",
			Description:    "Your virtual machine has been identified as under-utilized based on CPU usage",
			Problem:        "VM is running with low CPU utilization (less than 5% average over 14 days)",
			Solution:       "Consider resizing to a smaller SKU or shutting down if not needed",
			EstimatedImpact: "High cost savings potential",
			EstimatedSavings: &EstimatedSavings{
				Currency:        "USD",
				Amount:          150.50,
				Unit:            "Monthly",
				ConfidenceLevel: "High",
			},
			ResourceID:     fmt.Sprintf("/subscriptions/%s/resourceGroups/rg-aks/providers/Microsoft.Compute/virtualMachines/aks-vm-1", subscriptionID),
			ResourceName:   "aks-vm-1",
			ResourceType:   "Microsoft.Compute/virtualMachines",
			ResourceGroup:  "rg-aks",
			SubscriptionID: subscriptionID,
			LastUpdated:    time.Now().Add(-2 * time.Hour),
		},
		{
			ID:             fmt.Sprintf("/subscriptions/%s/recommendations/security-nsg-rules", subscriptionID),
			Name:           "security-nsg-rules",
			Type:           "Microsoft.Advisor/recommendations",
			Category:       RecommendationCategorySecurity,
			Severity:       RecommendationSeverityHigh,
			Status:         RecommendationStatusActive,
			Title:          "Restrict access through Internet-facing endpoints",
			Description:    "Network Security Group rules allow unrestricted access from the internet",
			Problem:        "NSG rules permit broad internet access (0.0.0.0/0) on sensitive ports",
			Solution:       "Restrict source IP ranges to only required addresses and implement least privilege access",
			EstimatedImpact: "High security risk reduction",
			ResourceID:     fmt.Sprintf("/subscriptions/%s/resourceGroups/rg-aks/providers/Microsoft.Network/networkSecurityGroups/aks-nsg", subscriptionID),
			ResourceName:   "aks-nsg",
			ResourceType:   "Microsoft.Network/networkSecurityGroups",
			ResourceGroup:  "rg-aks",
			SubscriptionID: subscriptionID,
			LastUpdated:    time.Now().Add(-1 * time.Hour),
		},
		{
			ID:             fmt.Sprintf("/subscriptions/%s/recommendations/performance-aks-scaling", subscriptionID),
			Name:           "performance-aks-scaling",
			Type:           "Microsoft.Advisor/recommendations",
			Category:       RecommendationCategoryPerformance,
			Severity:       RecommendationSeverityMedium,
			Status:         RecommendationStatusActive,
			Title:          "Enable cluster autoscaler for AKS nodes",
			Description:    "AKS cluster could benefit from automatic scaling capabilities",
			Problem:        "Manual scaling may lead to resource waste or performance issues during load spikes",
			Solution:       "Configure cluster autoscaler to automatically adjust node count based on demand",
			EstimatedImpact: "Improved performance and cost optimization",
			ResourceID:     fmt.Sprintf("/subscriptions/%s/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/aks-cluster", subscriptionID),
			ResourceName:   "aks-cluster",
			ResourceType:   "Microsoft.ContainerService/managedClusters",
			ResourceGroup:  "rg-aks",
			SubscriptionID: subscriptionID,
			LastUpdated:    time.Now().Add(-3 * time.Hour),
		},
		{
			ID:             fmt.Sprintf("/subscriptions/%s/recommendations/reliability-backup", subscriptionID),
			Name:           "reliability-backup",
			Type:           "Microsoft.Advisor/recommendations",
			Category:       RecommendationCategoryReliability,
			Severity:       RecommendationSeverityMedium,
			Status:         RecommendationStatusActive,
			Title:          "Configure backup for persistent volumes",
			Description:    "Persistent volumes in AKS cluster lack backup configuration",
			Problem:        "No backup strategy configured for critical data stored in persistent volumes",
			Solution:       "Implement Azure Backup for persistent volumes or configure volume snapshots",
			EstimatedImpact: "Improved data protection and disaster recovery capability",
			ResourceID:     fmt.Sprintf("/subscriptions/%s/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/aks-cluster", subscriptionID),
			ResourceName:   "aks-cluster",
			ResourceType:   "Microsoft.ContainerService/managedClusters",
			ResourceGroup:  "rg-aks",
			SubscriptionID: subscriptionID,
			LastUpdated:    time.Now().Add(-4 * time.Hour),
		},
		{
			ID:             fmt.Sprintf("/subscriptions/%s/recommendations/operational-monitoring", subscriptionID),
			Name:           "operational-monitoring",
			Type:           "Microsoft.Advisor/recommendations",
			Category:       RecommendationCategoryOperational,
			Severity:       RecommendationSeverityLow,
			Status:         RecommendationStatusActive,
			Title:          "Enable Container Insights for AKS cluster",
			Description:    "AKS cluster monitoring could be enhanced with Container Insights",
			Problem:        "Limited visibility into cluster and workload performance metrics",
			Solution:       "Enable Azure Monitor Container Insights for comprehensive monitoring",
			EstimatedImpact: "Improved operational visibility and troubleshooting capabilities",
			ResourceID:     fmt.Sprintf("/subscriptions/%s/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/aks-cluster", subscriptionID),
			ResourceName:   "aks-cluster",
			ResourceType:   "Microsoft.ContainerService/managedClusters",
			ResourceGroup:  "rg-aks",
			SubscriptionID: subscriptionID,
			LastUpdated:    time.Now().Add(-5 * time.Hour),
		},
	}

	// Add implementation guides to some recommendations
	baseRecommendations[0].ImplementationGuide = []ImplementationStep{
		{
			StepNumber:  1,
			Title:       "Analyze current usage",
			Description: "Review CPU and memory utilization over the past 30 days",
			Action:      "Use Azure Monitor to analyze performance metrics",
			IsRequired:  true,
			Effort:      "Low",
		},
		{
			StepNumber:  2,
			Title:       "Select appropriate SKU",
			Description: "Choose a smaller VM SKU that meets your workload requirements",
			Action:      "Use Azure VM Size recommendations tool",
			IsRequired:  true,
			Effort:      "Medium",
		},
		{
			StepNumber:  3,
			Title:       "Schedule downtime",
			Description: "Plan a maintenance window for VM resizing",
			Action:      "Coordinate with stakeholders for downtime window",
			IsRequired:  true,
			Effort:      "Low",
		},
		{
			StepNumber:  4,
			Title:       "Resize VM",
			Description: "Change the VM SKU to the recommended size",
			Action:      "Use Azure portal or CLI to resize the VM",
			IsRequired:  true,
			Effort:      "Medium",
		},
	}

	baseRecommendations[1].ImplementationGuide = []ImplementationStep{
		{
			StepNumber:  1,
			Title:       "Audit current NSG rules",
			Description: "Review all existing Network Security Group rules",
			Action:      "Document current inbound and outbound rules",
			IsRequired:  true,
			Effort:      "Medium",
		},
		{
			StepNumber:  2,
			Title:       "Identify required access",
			Description: "Determine legitimate source IP ranges and ports",
			Action:      "Work with application teams to define access requirements",
			IsRequired:  true,
			Effort:      "High",
		},
		{
			StepNumber:  3,
			Title:       "Update NSG rules",
			Description: "Modify rules to restrict access to specific IP ranges",
			Action:      "Update NSG rules using Azure portal or ARM templates",
			IsRequired:  true,
			Effort:      "Medium",
		},
		{
			StepNumber:  4,
			Title:       "Test connectivity",
			Description: "Verify that legitimate traffic still works after changes",
			Action:      "Perform connectivity tests from allowed sources",
			IsRequired:  true,
			Effort:      "Medium",
		},
	}

	return baseRecommendations
}

// generateMockRecommendationDetails creates mock detailed recommendation information
func generateMockRecommendationDetails(recommendationID string, includeImplementationStatus bool) *RecommendationDetails {
	// Extract category from ID for mock purposes
	var category RecommendationCategory
	if strings.Contains(recommendationID, "cost") {
		category = RecommendationCategoryCost
	} else if strings.Contains(recommendationID, "security") {
		category = RecommendationCategorySecurity
	} else if strings.Contains(recommendationID, "performance") {
		category = RecommendationCategoryPerformance
	} else if strings.Contains(recommendationID, "reliability") {
		category = RecommendationCategoryReliability
	} else {
		category = RecommendationCategoryOperational
	}

	// Create base recommendation
	recommendation := AdvisorRecommendation{
		ID:           recommendationID,
		Name:         extractNameFromID(recommendationID),
		Type:         "Microsoft.Advisor/recommendations",
		Category:     category,
		Severity:     RecommendationSeverityMedium,
		Status:       RecommendationStatusActive,
		Title:        generateMockTitle(category),
		Description:  generateMockDescription(category),
		Problem:      generateMockProblem(category),
		Solution:     generateMockSolution(category),
		LastUpdated:  time.Now().Add(-2 * time.Hour),
	}

	details := &RecommendationDetails{
		Recommendation: recommendation,
		RelatedResources: []RelatedResource{
			{
				ResourceID:   "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks/vnet1",
				ResourceName: "vnet1",
				ResourceType: "Microsoft.Network/virtualNetworks",
				Relationship: "related",
			},
		},
		ImplementationRisk: ImplementationRisk{
			Level:      generateMockRiskLevel(category),
			Factors:    generateMockRiskFactors(category),
			Mitigation: generateMockRiskMitigation(category),
			Downtime:   generateMockDowntime(category),
		},
		BusinessImpact: BusinessImpact{
			Performance: generateMockBusinessImpact(category, "performance"),
			Cost:        generateMockBusinessImpact(category, "cost"),
			Security:    generateMockBusinessImpact(category, "security"),
			Reliability: generateMockBusinessImpact(category, "reliability"),
			Compliance:  generateMockBusinessImpact(category, "compliance"),
		},
		TechnicalDetails: map[string]interface{}{
			"detectionMethod": "Automated analysis",
			"analysisVersion": "1.2.3",
			"lastAnalysis":    time.Now().Add(-6 * time.Hour).Format(time.RFC3339),
		},
		SimilarRecommendations: []string{
			fmt.Sprintf("%s-similar-1", recommendationID),
			fmt.Sprintf("%s-similar-2", recommendationID),
		},
	}

	return details
}

// Helper functions for mock data generation
func extractNameFromID(id string) string {
	parts := strings.Split(id, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown-recommendation"
}

func generateMockTitle(category RecommendationCategory) string {
	switch category {
	case RecommendationCategoryCost:
		return "Optimize resource costs"
	case RecommendationCategorySecurity:
		return "Improve security posture"
	case RecommendationCategoryPerformance:
		return "Enhance performance"
	case RecommendationCategoryReliability:
		return "Increase reliability"
	case RecommendationCategoryOperational:
		return "Improve operational efficiency"
	default:
		return "General recommendation"
	}
}

func generateMockDescription(category RecommendationCategory) string {
	switch category {
	case RecommendationCategoryCost:
		return "This recommendation can help reduce your Azure costs"
	case RecommendationCategorySecurity:
		return "This recommendation can improve your security posture"
	case RecommendationCategoryPerformance:
		return "This recommendation can enhance application performance"
	case RecommendationCategoryReliability:
		return "This recommendation can increase system reliability"
	case RecommendationCategoryOperational:
		return "This recommendation can improve operational processes"
	default:
		return "This is a general recommendation"
	}
}

func generateMockProblem(category RecommendationCategory) string {
	switch category {
	case RecommendationCategoryCost:
		return "Resources are not optimally sized for current usage patterns"
	case RecommendationCategorySecurity:
		return "Security configurations do not follow best practices"
	case RecommendationCategoryPerformance:
		return "Performance could be improved with configuration changes"
	case RecommendationCategoryReliability:
		return "Current setup lacks proper redundancy and backup"
	case RecommendationCategoryOperational:
		return "Monitoring and alerting could be enhanced"
	default:
		return "General improvement opportunity identified"
	}
}

func generateMockSolution(category RecommendationCategory) string {
	switch category {
	case RecommendationCategoryCost:
		return "Resize resources to match actual usage patterns"
	case RecommendationCategorySecurity:
		return "Apply security best practices and policies"
	case RecommendationCategoryPerformance:
		return "Optimize configuration for better performance"
	case RecommendationCategoryReliability:
		return "Implement backup and redundancy strategies"
	case RecommendationCategoryOperational:
		return "Enable monitoring and configure appropriate alerts"
	default:
		return "Apply recommended changes"
	}
}

func generateMockRiskLevel(category RecommendationCategory) string {
	switch category {
	case RecommendationCategoryCost:
		return "Low"
	case RecommendationCategorySecurity:
		return "Medium"
	case RecommendationCategoryPerformance:
		return "Low"
	case RecommendationCategoryReliability:
		return "Medium"
	case RecommendationCategoryOperational:
		return "Low"
	default:
		return "Low"
	}
}

func generateMockRiskFactors(category RecommendationCategory) []string {
	switch category {
	case RecommendationCategoryCost:
		return []string{"Resource downtime during resizing", "Potential performance impact"}
	case RecommendationCategorySecurity:
		return []string{"Configuration changes may affect connectivity", "Potential service disruption"}
	case RecommendationCategoryPerformance:
		return []string{"Configuration changes may require testing", "Monitoring needed post-change"}
	case RecommendationCategoryReliability:
		return []string{"Implementation requires planning", "May involve additional costs"}
	case RecommendationCategoryOperational:
		return []string{"Learning curve for new tools", "Configuration time required"}
	default:
		return []string{"General implementation considerations"}
	}
}

func generateMockRiskMitigation(category RecommendationCategory) []string {
	switch category {
	case RecommendationCategoryCost:
		return []string{"Schedule changes during maintenance windows", "Test in non-production first"}
	case RecommendationCategorySecurity:
		return []string{"Implement changes incrementally", "Have rollback plan ready"}
	case RecommendationCategoryPerformance:
		return []string{"Monitor performance before and after changes", "Have baseline measurements"}
	case RecommendationCategoryReliability:
		return []string{"Plan implementation phases", "Validate backup procedures"}
	case RecommendationCategoryOperational:
		return []string{"Provide training for operations team", "Start with pilot deployment"}
	default:
		return []string{"Follow change management procedures"}
	}
}

func generateMockDowntime(category RecommendationCategory) string {
	switch category {
	case RecommendationCategoryCost:
		return "Minimal"
	case RecommendationCategorySecurity:
		return "None"
	case RecommendationCategoryPerformance:
		return "None"
	case RecommendationCategoryReliability:
		return "Minimal"
	case RecommendationCategoryOperational:
		return "None"
	default:
		return "None"
	}
}

func generateMockBusinessImpact(category RecommendationCategory, impactType string) string {
	if category == RecommendationCategoryCost && impactType == "cost" {
		return "High positive impact - significant cost savings expected"
	}
	if category == RecommendationCategorySecurity && impactType == "security" {
		return "High positive impact - improved security posture"
	}
	if category == RecommendationCategoryPerformance && impactType == "performance" {
		return "Medium positive impact - better application responsiveness"
	}
	if category == RecommendationCategoryReliability && impactType == "reliability" {
		return "High positive impact - reduced downtime risk"
	}
	if category == RecommendationCategoryOperational && impactType == "performance" {
		return "Medium positive impact - improved operational efficiency"
	}
	return "Minimal impact"
}

// applyRecommendationFilters applies the provided filters to the recommendations
func applyRecommendationFilters(recommendations []AdvisorRecommendation, filter *RecommendationFilter) []AdvisorRecommendation {
	if filter == nil {
		return recommendations
	}

	var filtered []AdvisorRecommendation

	for _, rec := range recommendations {
		// Check resource group filter
		if filter.ResourceGroup != "" && rec.ResourceGroup != filter.ResourceGroup {
			continue
		}

		// Check category filter
		if len(filter.Category) > 0 {
			found := false
			for _, cat := range filter.Category {
				if rec.Category == cat {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check severity filter
		if len(filter.Severity) > 0 {
			found := false
			for _, sev := range filter.Severity {
				if rec.Severity == sev {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check status filter
		if len(filter.Status) > 0 {
			found := false
			for _, status := range filter.Status {
				if rec.Status == status {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check resource type filter
		if filter.ResourceType != "" && rec.ResourceType != filter.ResourceType {
			continue
		}

		filtered = append(filtered, rec)
	}

	return filtered
}