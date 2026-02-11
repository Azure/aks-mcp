package advisor

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// Advisory-related tool registrations

// RegisterAdvisorRecommendationTool registers the aks_advisor_recommendation tool
func RegisterAdvisorRecommendationTool() mcp.Tool {
	return mcp.NewTool(
		"aks_advisor_recommendation",
		mcp.WithDescription("Retrieve and manage Azure Advisor recommendations for AKS clusters"),
		mcp.WithTitleAnnotation("Azure Advisor Recommendations"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("operation",
			mcp.Description("Operation to perform: list or report"),
			mcp.Required(),
		),
		mcp.WithString("subscription_id",
			mcp.Description("Azure subscription ID to query recommendations"),
			mcp.Required(),
		),
		mcp.WithString("resource_group",
			mcp.Description("Filter by specific resource group containing AKS clusters"),
		),
		mcp.WithString("cluster_names",
			mcp.Description("Comma-separated list of specific AKS cluster names to filter recommendations"),
		),
		mcp.WithString("category",
			mcp.Description("Filter by recommendation category: Cost, HighAvailability, Performance, Security"),
		),
		mcp.WithString("severity",
			mcp.Description("Filter by severity level: High, Medium, Low"),
		),
		mcp.WithString("format",
			mcp.Description("Output format for reports: summary, detailed, actionable"),
		),
	)
}
