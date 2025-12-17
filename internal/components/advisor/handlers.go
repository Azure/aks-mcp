package advisor

import (
	"context"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/tools"
)

// =============================================================================
// Advisory-related Handlers
// =============================================================================

// GetAdvisorRecommendationHandler returns a handler for the az_advisor_recommendation command
func GetAdvisorRecommendationHandler(cfg *config.ConfigData) tools.ResourceHandler {
	return tools.ResourceHandlerFunc(func(ctx context.Context, params map[string]interface{}, _ *config.ConfigData) (string, error) {
		// Use the advisor package handler directly
		return HandleAdvisorRecommendation(ctx, params, cfg)
	})
}
