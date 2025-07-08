package advisor

import (
	"time"
)

// RecommendationCategory represents the category of an Azure Advisor recommendation
type RecommendationCategory string

const (
	RecommendationCategoryCost        RecommendationCategory = "Cost"
	RecommendationCategoryPerformance RecommendationCategory = "Performance"
	RecommendationCategorySecurity    RecommendationCategory = "Security"
	RecommendationCategoryReliability RecommendationCategory = "Reliability"
	RecommendationCategoryOperational RecommendationCategory = "Operational"
)

// RecommendationSeverity represents the severity level of a recommendation
type RecommendationSeverity string

const (
	RecommendationSeverityHigh   RecommendationSeverity = "High"
	RecommendationSeverityMedium RecommendationSeverity = "Medium"
	RecommendationSeverityLow    RecommendationSeverity = "Low"
)

// RecommendationStatus represents the status of a recommendation
type RecommendationStatus string

const (
	RecommendationStatusActive    RecommendationStatus = "Active"
	RecommendationStatusDismissed RecommendationStatus = "Dismissed"
	RecommendationStatusPostponed RecommendationStatus = "Postponed"
	RecommendationStatusResolved  RecommendationStatus = "Resolved"
)

// AdvisorRecommendation represents an Azure Advisor recommendation
type AdvisorRecommendation struct {
	ID                  string                 `json:"id"`
	Name                string                 `json:"name"`
	Type                string                 `json:"type"`
	Category            RecommendationCategory `json:"category"`
	Severity            RecommendationSeverity `json:"severity"`
	Status              RecommendationStatus   `json:"status"`
	Title               string                 `json:"title"`
	Description         string                 `json:"description"`
	Problem             string                 `json:"problem"`
	Solution            string                 `json:"solution"`
	EstimatedImpact     string                 `json:"estimatedImpact"`
	EstimatedSavings    *EstimatedSavings      `json:"estimatedSavings,omitempty"`
	ResourceID          string                 `json:"resourceId"`
	ResourceName        string                 `json:"resourceName"`
	ResourceType        string                 `json:"resourceType"`
	ResourceGroup       string                 `json:"resourceGroup"`
	SubscriptionID      string                 `json:"subscriptionId"`
	LastUpdated         time.Time              `json:"lastUpdated"`
	ImplementationGuide []ImplementationStep   `json:"implementationGuide,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	Tags                map[string]string      `json:"tags,omitempty"`
}

// EstimatedSavings represents the estimated cost savings for a recommendation
type EstimatedSavings struct {
	Currency        string  `json:"currency"`
	Amount          float64 `json:"amount"`
	Unit            string  `json:"unit"`            // e.g., "Monthly", "Annual"
	ConfidenceLevel string  `json:"confidenceLevel"` // e.g., "High", "Medium", "Low"
}

// ImplementationStep represents a step in implementing a recommendation
type ImplementationStep struct {
	StepNumber  int    `json:"stepNumber"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`
	IsRequired  bool   `json:"isRequired"`
	Effort      string `json:"effort"` // e.g., "Low", "Medium", "High"
}

// RecommendationFilter represents filtering options for recommendations
type RecommendationFilter struct {
	SubscriptionID string                   `json:"subscriptionId,omitempty"`
	ResourceGroup  string                   `json:"resourceGroup,omitempty"`
	Category       []RecommendationCategory `json:"category,omitempty"`
	Severity       []RecommendationSeverity `json:"severity,omitempty"`
	Status         []RecommendationStatus   `json:"status,omitempty"`
	ResourceType   string                   `json:"resourceType,omitempty"`
}

// RecommendationSummary represents a summary of recommendations
type RecommendationSummary struct {
	TotalRecommendations int                            `json:"totalRecommendations"`
	CategoryCounts       map[RecommendationCategory]int `json:"categoryCounts"`
	SeverityCounts       map[RecommendationSeverity]int `json:"severityCounts"`
	StatusCounts         map[RecommendationStatus]int   `json:"statusCounts"`
	TopCategories        []CategorySummary              `json:"topCategories"`
	EstimatedSavings     *TotalEstimatedSavings         `json:"estimatedSavings,omitempty"`
	HighPriorityCount    int                            `json:"highPriorityCount"`
	LastUpdated          time.Time                      `json:"lastUpdated"`
}

// CategorySummary represents a summary for a specific category
type CategorySummary struct {
	Category          RecommendationCategory `json:"category"`
	Count             int                    `json:"count"`
	HighSeverityCount int                    `json:"highSeverityCount"`
	EstimatedSavings  *EstimatedSavings      `json:"estimatedSavings,omitempty"`
}

// TotalEstimatedSavings represents total estimated savings across all recommendations
type TotalEstimatedSavings struct {
	Currency            string  `json:"currency"`
	TotalAmount         float64 `json:"totalAmount"`
	Unit                string  `json:"unit"`
	CostRecommendations int     `json:"costRecommendations"`
}

// RecommendationDetails represents detailed information about a specific recommendation
type RecommendationDetails struct {
	Recommendation         AdvisorRecommendation  `json:"recommendation"`
	RelatedResources       []RelatedResource      `json:"relatedResources,omitempty"`
	ImplementationRisk     ImplementationRisk     `json:"implementationRisk"`
	BusinessImpact         BusinessImpact         `json:"businessImpact"`
	TechnicalDetails       map[string]interface{} `json:"technicalDetails,omitempty"`
	SimilarRecommendations []string               `json:"similarRecommendations,omitempty"`
}

// RelatedResource represents a resource related to the recommendation
type RelatedResource struct {
	ResourceID   string `json:"resourceId"`
	ResourceName string `json:"resourceName"`
	ResourceType string `json:"resourceType"`
	Relationship string `json:"relationship"` // e.g., "dependent", "related", "parent"
}

// ImplementationRisk represents the risk assessment for implementing a recommendation
type ImplementationRisk struct {
	Level      string   `json:"level"` // e.g., "Low", "Medium", "High"
	Factors    []string `json:"factors"`
	Mitigation []string `json:"mitigation"`
	Downtime   string   `json:"downtime"` // e.g., "None", "Minimal", "Significant"
}

// BusinessImpact represents the business impact of a recommendation
type BusinessImpact struct {
	Performance string `json:"performance"`
	Cost        string `json:"cost"`
	Security    string `json:"security"`
	Reliability string `json:"reliability"`
	Compliance  string `json:"compliance"`
}
