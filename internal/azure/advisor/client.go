package advisor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
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
	// Build the API URL for Azure Advisor recommendations
	apiURL := fmt.Sprintf("%s/subscriptions/%s/providers/Microsoft.Advisor/recommendations", c.baseURL, c.subscriptionID)

	// Add query parameters
	params := url.Values{}
	params.Add("api-version", "2023-01-01")

	// Add filter for AKS resources to match the comment requirement
	filterQuery := "resourceType eq 'Microsoft.ContainerService/managedClusters'"

	// Add additional filters if provided
	if filter != nil {
		if len(filter.Category) > 0 {
			categoryFilters := make([]string, len(filter.Category))
			for i, cat := range filter.Category {
				categoryFilters[i] = fmt.Sprintf("category eq '%s'", string(cat))
			}
			filterQuery += " and (" + strings.Join(categoryFilters, " or ") + ")"
		}

		if filter.ResourceGroup != "" {
			filterQuery += fmt.Sprintf(" and resourceGroup eq '%s'", filter.ResourceGroup)
		}
	}

	params.Add("$filter", filterQuery)
	apiURL += "?" + params.Encode()

	// Make the API call
	resp, err := c.makeAuthenticatedRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call Azure Advisor API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Azure Advisor API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var advisorResponse struct {
		Value []struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Type       string `json:"type"`
			Properties struct {
				Category         string `json:"category"`
				Impact           string `json:"impact"`
				ShortDescription struct {
					Problem  string `json:"problem"`
					Solution string `json:"solution"`
				} `json:"shortDescription"`
				ExtendedProperties map[string]interface{} `json:"extendedProperties"`
				ResourceMetadata   struct {
					ResourceID string `json:"resourceId"`
					Source     string `json:"source"`
				} `json:"resourceMetadata"`
				LastUpdated string `json:"lastUpdated"`
			} `json:"properties"`
		} `json:"value"`
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&advisorResponse); err != nil {
		return nil, fmt.Errorf("failed to parse advisor response: %w", err)
	}

	// Convert API response to our format
	var recommendations []AdvisorRecommendation
	for _, item := range advisorResponse.Value {
		rec := convertAPIResponseToRecommendation(item, c.subscriptionID)
		recommendations = append(recommendations, rec)
	}

	// Apply additional filters that aren't supported by the API
	if filter != nil {
		recommendations = applyClientSideFilters(recommendations, filter)
	}

	return recommendations, nil
}

// GetRecommendationDetails retrieves detailed information about a specific recommendation
func (c *AdvisorClient) GetRecommendationDetails(ctx context.Context, recommendationID string, includeImplementationStatus bool) (*RecommendationDetails, error) {
	if recommendationID == "" {
		return nil, fmt.Errorf("recommendation ID cannot be empty")
	}

	// Build the API URL for the specific recommendation
	apiURL := fmt.Sprintf("%s%s", c.baseURL, recommendationID)

	// Add query parameters
	params := url.Values{}
	params.Add("api-version", "2023-01-01")
	apiURL += "?" + params.Encode()

	// Make the API call
	resp, err := c.makeAuthenticatedRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call Azure Advisor API for recommendation details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Azure Advisor API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var detailResponse struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Type       string `json:"type"`
		Properties struct {
			Category         string `json:"category"`
			Impact           string `json:"impact"`
			ShortDescription struct {
				Problem  string `json:"problem"`
				Solution string `json:"solution"`
			} `json:"shortDescription"`
			ExtendedProperties map[string]interface{} `json:"extendedProperties"`
			ResourceMetadata   struct {
				ResourceID string `json:"resourceId"`
				Source     string `json:"source"`
			} `json:"resourceMetadata"`
			LastUpdated string `json:"lastUpdated"`
		} `json:"properties"`
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&detailResponse); err != nil {
		return nil, fmt.Errorf("failed to parse recommendation details response: %w", err)
	}

	// Convert API response to our format
	recommendation := convertAPIResponseToRecommendation(detailResponse, c.subscriptionID)

	// Create detailed response with enhanced information
	details := &RecommendationDetails{
		Recommendation:     recommendation,
		RelatedResources:   extractRelatedResources(detailResponse.Properties.ExtendedProperties),
		ImplementationRisk: generateImplementationRisk(recommendation.Category),
		BusinessImpact:     generateBusinessImpact(recommendation.Category),
		TechnicalDetails: map[string]interface{}{
			"source":        detailResponse.Properties.ResourceMetadata.Source,
			"extendedProps": detailResponse.Properties.ExtendedProperties,
			"lastAnalysis":  detailResponse.Properties.LastUpdated,
		},
		SimilarRecommendations: []string{}, // Would need additional API calls to populate
	}

	return details, nil
}

// makeAuthenticatedRequest makes an authenticated HTTP request to Azure Management API
func (c *AdvisorClient) makeAuthenticatedRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Get access token
	tokenRequestOptions := policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	}

	token, err := c.credential.GetToken(ctx, tokenRequestOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	return c.httpClient.Do(req)
}

// convertAPIResponseToRecommendation converts Azure Advisor API response to our recommendation format
func convertAPIResponseToRecommendation(item interface{}, subscriptionID string) AdvisorRecommendation {
	// Type for API response structure
	type APIRecommendation struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Type       string `json:"type"`
		Properties struct {
			Category         string `json:"category"`
			Impact           string `json:"impact"`
			ShortDescription struct {
				Problem  string `json:"problem"`
				Solution string `json:"solution"`
			} `json:"shortDescription"`
			ExtendedProperties map[string]interface{} `json:"extendedProperties"`
			ResourceMetadata   struct {
				ResourceID string `json:"resourceId"`
				Source     string `json:"source"`
			} `json:"resourceMetadata"`
			LastUpdated string `json:"lastUpdated"`
		} `json:"properties"`
	}

	// Initialize variables with defaults
	var id, name, typeStr, category, impact, problem, solution, lastUpdated, resourceID string
	var extendedProps map[string]interface{}

	// Convert the interface{} to our expected type
	// In practice, this will be the struct from the API response
	jsonBytes, err := json.Marshal(item)
	if err != nil {
		// Return a fallback recommendation if we can't marshal
		return AdvisorRecommendation{
			ID:             fmt.Sprintf("/subscriptions/%s/recommendations/unknown", subscriptionID),
			Name:           "unknown",
			Type:           "Microsoft.Advisor/recommendations",
			Category:       RecommendationCategoryOperational,
			Severity:       RecommendationSeverityLow,
			Status:         RecommendationStatusActive,
			Title:          "Unknown recommendation",
			Description:    "Could not parse recommendation details",
			Problem:        "Unable to parse API response",
			Solution:       "Check API response format",
			LastUpdated:    time.Now(),
			SubscriptionID: subscriptionID,
		}
	}

	var apiRec APIRecommendation
	if err := json.Unmarshal(jsonBytes, &apiRec); err != nil {
		// Return a fallback recommendation if we can't unmarshal
		return AdvisorRecommendation{
			ID:             fmt.Sprintf("/subscriptions/%s/recommendations/parse-error", subscriptionID),
			Name:           "parse-error",
			Type:           "Microsoft.Advisor/recommendations",
			Category:       RecommendationCategoryOperational,
			Severity:       RecommendationSeverityLow,
			Status:         RecommendationStatusActive,
			Title:          "Parse error",
			Description:    "Failed to parse recommendation details",
			Problem:        "Unable to unmarshal API response",
			Solution:       "Check API response format",
			LastUpdated:    time.Now(),
			SubscriptionID: subscriptionID,
		}
	}

	// Extract values from parsed structure
	id = apiRec.ID
	name = apiRec.Name
	typeStr = apiRec.Type
	category = apiRec.Properties.Category
	impact = apiRec.Properties.Impact
	problem = apiRec.Properties.ShortDescription.Problem
	solution = apiRec.Properties.ShortDescription.Solution
	lastUpdated = apiRec.Properties.LastUpdated
	resourceID = apiRec.Properties.ResourceMetadata.ResourceID
	extendedProps = apiRec.Properties.ExtendedProperties

	// Parse last updated time
	var parsedTime time.Time
	if lastUpdated != "" {
		if parsed, err := time.Parse(time.RFC3339, lastUpdated); err == nil {
			parsedTime = parsed
		} else {
			parsedTime = time.Now()
		}
	} else {
		parsedTime = time.Now()
	}

	// Extract resource information
	resourceName, resourceType, resourceGroup := extractResourceInfo(resourceID)

	// Map impact to severity
	severity := mapImpactToSeverity(impact)

	// Create the recommendation
	rec := AdvisorRecommendation{
		ID:             id,
		Name:           name,
		Type:           typeStr,
		Category:       mapCategoryString(category),
		Severity:       severity,
		Status:         RecommendationStatusActive,
		Title:          generateTitleFromProblem(problem),
		Description:    problem,
		Problem:        problem,
		Solution:       solution,
		ResourceID:     resourceID,
		ResourceName:   resourceName,
		ResourceType:   resourceType,
		ResourceGroup:  resourceGroup,
		SubscriptionID: subscriptionID,
		LastUpdated:    parsedTime,
	}

	// Add estimated savings if available in extended properties
	if savings := extractEstimatedSavings(extendedProps); savings != nil {
		rec.EstimatedSavings = savings
	}

	return rec
}

// extractResourceInfo extracts resource information from resource ID
func extractResourceInfo(resourceID string) (name, resourceType, resourceGroup string) {
	if resourceID == "" {
		return "unknown", "unknown", "unknown"
	}

	parts := strings.Split(resourceID, "/")
	if len(parts) < 5 {
		return "unknown", "unknown", "unknown"
	}

	// Extract resource group (position 4 in the path)
	if len(parts) > 4 {
		resourceGroup = parts[4]
	}

	// Extract resource type (combine provider and type)
	if len(parts) > 7 {
		resourceType = parts[6] + "/" + parts[7]
	}

	// Extract resource name (last part)
	if len(parts) > 0 {
		name = parts[len(parts)-1]
	}

	return name, resourceType, resourceGroup
}

// mapImpactToSeverity maps Azure Advisor impact to our severity enum
func mapImpactToSeverity(impact string) RecommendationSeverity {
	switch strings.ToLower(impact) {
	case "high":
		return RecommendationSeverityHigh
	case "medium":
		return RecommendationSeverityMedium
	case "low":
		return RecommendationSeverityLow
	default:
		return RecommendationSeverityMedium
	}
}

// mapCategoryString maps Azure Advisor category to our category enum
func mapCategoryString(category string) RecommendationCategory {
	switch strings.ToLower(category) {
	case "cost":
		return RecommendationCategoryCost
	case "security":
		return RecommendationCategorySecurity
	case "performance":
		return RecommendationCategoryPerformance
	case "reliability":
		return RecommendationCategoryReliability
	case "operational":
		return RecommendationCategoryOperational
	default:
		return RecommendationCategoryOperational
	}
}

// generateTitleFromProblem generates a title from the problem description
func generateTitleFromProblem(problem string) string {
	if problem == "" {
		return "Azure Advisor Recommendation"
	}

	// Truncate if too long and add "..."
	if len(problem) > 60 {
		return problem[:57] + "..."
	}

	return problem
}

// extractEstimatedSavings extracts estimated savings from extended properties
func extractEstimatedSavings(extendedProps map[string]interface{}) *EstimatedSavings {
	if extendedProps == nil {
		return nil
	}

	// Look for cost-related properties
	var amount float64
	var currency string = "USD"
	var found bool

	// Common property names for savings in Azure Advisor
	if savingsValue, ok := extendedProps["savings"]; ok {
		if amount, ok = savingsValue.(float64); ok {
			found = true
		}
	}

	if monthlySavings, ok := extendedProps["monthlySavings"]; ok {
		if amount, ok = monthlySavings.(float64); ok {
			found = true
		}
	}

	if annualSavings, ok := extendedProps["annualSavings"]; ok {
		if annualAmount, ok := annualSavings.(float64); ok {
			amount = annualAmount / 12 // Convert to monthly
			found = true
		}
	}

	if currencyValue, ok := extendedProps["currency"]; ok {
		if currStr, ok := currencyValue.(string); ok {
			currency = currStr
		}
	}

	if !found {
		return nil
	}

	return &EstimatedSavings{
		Currency:        currency,
		Amount:          amount,
		Unit:            "Monthly",
		ConfidenceLevel: "Medium",
	}
}

// extractRelatedResources extracts related resources from extended properties
func extractRelatedResources(extendedProps map[string]interface{}) []RelatedResource {
	var resources []RelatedResource

	// This would be implementation-specific based on what's in extended properties
	// For now, return empty slice as we don't know the exact structure
	return resources
}

// applyClientSideFilters applies filters that aren't supported by the API
func applyClientSideFilters(recommendations []AdvisorRecommendation, filter *RecommendationFilter) []AdvisorRecommendation {
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

		// Check severity filter (client-side since API doesn't support it directly)
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

// generateImplementationRisk generates implementation risk based on category
func generateImplementationRisk(category RecommendationCategory) ImplementationRisk {
	switch category {
	case RecommendationCategoryCost:
		return ImplementationRisk{
			Level:      "Low",
			Factors:    []string{"Resource downtime during resizing", "Potential performance impact"},
			Mitigation: []string{"Schedule changes during maintenance windows", "Test in non-production first"},
			Downtime:   "Minimal",
		}
	case RecommendationCategorySecurity:
		return ImplementationRisk{
			Level:      "Medium",
			Factors:    []string{"Configuration changes may affect connectivity", "Potential service disruption"},
			Mitigation: []string{"Implement changes incrementally", "Have rollback plan ready"},
			Downtime:   "None",
		}
	default:
		return ImplementationRisk{
			Level:      "Low",
			Factors:    []string{"General implementation considerations"},
			Mitigation: []string{"Follow change management procedures"},
			Downtime:   "None",
		}
	}
}

// generateBusinessImpact generates business impact based on category
func generateBusinessImpact(category RecommendationCategory) BusinessImpact {
	impact := BusinessImpact{
		Performance: "Minimal impact",
		Cost:        "Minimal impact",
		Security:    "Minimal impact",
		Reliability: "Minimal impact",
		Compliance:  "Minimal impact",
	}

	switch category {
	case RecommendationCategoryCost:
		impact.Cost = "High positive impact - significant cost savings expected"
	case RecommendationCategorySecurity:
		impact.Security = "High positive impact - improved security posture"
	case RecommendationCategoryPerformance:
		impact.Performance = "Medium positive impact - better application responsiveness"
	case RecommendationCategoryReliability:
		impact.Reliability = "High positive impact - reduced downtime risk"
	case RecommendationCategoryOperational:
		impact.Performance = "Medium positive impact - improved operational efficiency"
	}

	return impact
}
