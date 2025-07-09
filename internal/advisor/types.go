package advisor

import (
	"time"
)

// AKSRecommendationSummary represents an Azure Advisor recommendation for AKS resources
type AKSRecommendationSummary struct {
	ID               string                   `json:"id"`
	Category         string                   `json:"category"`
	Impact           string                   `json:"impact"`
	ClusterName      string                   `json:"cluster_name"`
	ResourceGroup    string                   `json:"resource_group"`
	ResourceID       string                   `json:"resource_id"`
	Description      string                   `json:"description"`
	Severity         string                   `json:"severity"`
	PotentialSavings *CostSavings             `json:"potential_savings,omitempty"`
	LastUpdated      time.Time                `json:"last_updated"`
	Status           string                   `json:"status"`
	AKSSpecific      AKSRecommendationDetails `json:"aks_specific"`
}

// AKSRecommendationDetails contains AKS-specific information
type AKSRecommendationDetails struct {
	ClusterVersion    string   `json:"cluster_version,omitempty"`
	NodePoolNames     []string `json:"node_pool_names,omitempty"`
	WorkloadType      string   `json:"workload_type,omitempty"`
	ConfigurationArea string   `json:"configuration_area,omitempty"` // networking, compute, storage, security
}

// CostSavings represents potential cost savings information
type CostSavings struct {
	Currency       string  `json:"currency"`
	AnnualSavings  float64 `json:"annual_savings"`
	MonthlySavings float64 `json:"monthly_savings"`
}

// AKSAdvisorReport represents a comprehensive report of AKS recommendations
type AKSAdvisorReport struct {
	SubscriptionID   string                     `json:"subscription_id"`
	GeneratedAt      time.Time                  `json:"generated_at"`
	Summary          AKSReportSummary           `json:"summary"`
	Recommendations  []AKSRecommendationSummary `json:"recommendations"`
	ActionItems      []AKSActionItem            `json:"action_items"`
	ClusterBreakdown []ClusterRecommendations   `json:"cluster_breakdown"`
}

// AKSReportSummary provides high-level statistics
type AKSReportSummary struct {
	TotalRecommendations  int            `json:"total_recommendations"`
	ByCategory            map[string]int `json:"by_category"`
	BySeverity            map[string]int `json:"by_severity"`
	TotalPotentialSavings *CostSavings   `json:"total_potential_savings,omitempty"`
	ClustersAffected      int            `json:"clusters_affected"`
}

// AKSActionItem represents a prioritized action item
type AKSActionItem struct {
	Priority         int    `json:"priority"`
	RecommendationID string `json:"recommendation_id"`
	ClusterName      string `json:"cluster_name"`
	Category         string `json:"category"`
	Description      string `json:"description"`
	EstimatedEffort  string `json:"estimated_effort"`
	PotentialImpact  string `json:"potential_impact"`
}

// ClusterRecommendations groups recommendations by cluster
type ClusterRecommendations struct {
	ClusterName     string                     `json:"cluster_name"`
	ResourceGroup   string                     `json:"resource_group"`
	Recommendations []AKSRecommendationSummary `json:"recommendations"`
	TotalSavings    *CostSavings               `json:"total_savings,omitempty"`
}

// CLIRecommendation represents the raw Azure CLI recommendation structure
type CLIRecommendation struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Category         string `json:"category"`
	Impact           string `json:"impact"`
	ImpactedValue    string `json:"impactedValue"`
	LastUpdated      string `json:"lastUpdated"`
	ShortDescription struct {
		Problem  string `json:"problem"`
		Solution string `json:"solution"`
	} `json:"shortDescription"`
}
