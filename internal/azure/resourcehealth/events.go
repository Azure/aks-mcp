package resourcehealth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// EventManager manages Resource Health event operations
type EventManager struct {
	client *ResourceHealthClient
}

// NewEventManager creates a new event manager
func NewEventManager(subscriptionID string, credential *azidentity.DefaultAzureCredential) (*EventManager, error) {
	client, err := NewResourceHealthClient(subscriptionID, credential)
	if err != nil {
		return nil, fmt.Errorf("failed to create Resource Health client: %w", err)
	}

	return &EventManager{
		client: client,
	}, nil
}

// GetResourceHealthStatus returns a formatted health status for resources
func (em *EventManager) GetResourceHealthStatus(ctx context.Context, resourceIDs []string, includeHistory bool) (string, error) {
	statuses, err := em.client.GetResourceHealthStatus(ctx, resourceIDs, includeHistory)
	if err != nil {
		return "", fmt.Errorf("failed to get resource health status: %w", err)
	}

	result := map[string]interface{}{
		"requestTime":      time.Now().UTC().Format(time.RFC3339),
		"resourceCount":    len(resourceIDs),
		"includeHistory":   includeHistory,
		"healthStatuses":   statuses,
		"summary":          generateHealthSummary(statuses),
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal health status response: %w", err)
	}

	return string(resultJSON), nil
}

// GetResourceHealthEvents returns formatted historical health events
func (em *EventManager) GetResourceHealthEvents(ctx context.Context, resourceID string, startTime, endTime *time.Time, healthStatusFilter []string) (string, error) {
	filter := &HealthEventFilter{}
	
	if startTime != nil {
		filter.StartTime = startTime
	}
	if endTime != nil {
		filter.EndTime = endTime
	}
	
	if len(healthStatusFilter) > 0 {
		for _, status := range healthStatusFilter {
			filter.HealthStatusFilter = append(filter.HealthStatusFilter, HealthStatus(status))
		}
	}

	events, err := em.client.GetResourceHealthEvents(ctx, resourceID, filter)
	if err != nil {
		return "", fmt.Errorf("failed to get resource health events: %w", err)
	}

	// Format the response for better readability
	result := formatHealthEventsResponse(resourceID, events, filter)

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal health events response: %w", err)
	}

	return string(resultJSON), nil
}

// generateHealthSummary creates a summary of health statuses
func generateHealthSummary(statuses []ResourceHealthStatus) map[string]interface{} {
	summary := map[string]interface{}{
		"totalResources": len(statuses),
		"statusCounts": map[string]int{
			"Available":   0,
			"Unavailable": 0,
			"Degraded":    0,
			"Unknown":     0,
		},
		"overallHealth": "Unknown",
		"issuesFound":   []string{},
	}

	statusCounts := summary["statusCounts"].(map[string]int)
	var issues []string

	for _, status := range statuses {
		statusCounts[string(status.Status)]++
		
		if status.Status != HealthStatusAvailable {
			issues = append(issues, fmt.Sprintf("%s: %s", status.ResourceName, status.StatusSummary))
		}
	}

	summary["issuesFound"] = issues

	// Determine overall health
	if statusCounts["Unavailable"] > 0 {
		summary["overallHealth"] = "Critical"
	} else if statusCounts["Degraded"] > 0 {
		summary["overallHealth"] = "Warning"
	} else if statusCounts["Available"] == len(statuses) {
		summary["overallHealth"] = "Healthy"
	} else {
		summary["overallHealth"] = "Unknown"
	}

	// Calculate availability percentage
	availableCount := statusCounts["Available"]
	if len(statuses) > 0 {
		availabilityPercent := float64(availableCount) / float64(len(statuses)) * 100
		summary["availabilityPercent"] = fmt.Sprintf("%.1f%%", availabilityPercent)
	}

	return summary
}

// formatHealthEventsResponse formats the health events response for better readability
func formatHealthEventsResponse(resourceID string, events []ResourceHealthEvent, filter *HealthEventFilter) map[string]interface{} {
	result := map[string]interface{}{
		"resourceId":    resourceID,
		"requestTime":   time.Now().UTC().Format(time.RFC3339),
		"eventCount":    len(events),
		"timeRange":     formatTimeRange(filter),
		"events":        events,
	}

	if len(events) > 0 {
		// Add event summary
		summary := generateEventSummary(events)
		result["summary"] = summary

		// Add recent events (last 7 days)
		recentEvents := filterRecentEvents(events, 7*24*time.Hour)
		if len(recentEvents) > 0 {
			result["recentEvents"] = map[string]interface{}{
				"count":  len(recentEvents),
				"events": recentEvents,
			}
		}

		// Add active issues
		activeIssues := filterActiveEvents(events)
		if len(activeIssues) > 0 {
			result["activeIssues"] = map[string]interface{}{
				"count":  len(activeIssues),
				"events": activeIssues,
			}
		}
	}

	return result
}

// generateEventSummary creates a summary of health events
func generateEventSummary(events []ResourceHealthEvent) map[string]interface{} {
	summary := map[string]interface{}{
		"totalEvents": len(events),
		"eventTypes": map[string]int{},
		"statusDistribution": map[string]int{},
		"levelDistribution": map[string]int{},
		"timeSpan": map[string]interface{}{},
	}

	eventTypes := summary["eventTypes"].(map[string]int)
	statusDistribution := summary["statusDistribution"].(map[string]int)
	levelDistribution := summary["levelDistribution"].(map[string]int)

	var earliestTime, latestTime time.Time
	var activeCount, resolvedCount int

	for i, event := range events {
		// Count event types
		eventTypes[string(event.EventType)]++
		
		// Count status distribution
		statusDistribution[string(event.Status)]++
		
		// Count level distribution
		levelDistribution[event.Level]++

		// Track time span
		if i == 0 {
			earliestTime = event.StartTime
			latestTime = event.StartTime
		} else {
			if event.StartTime.Before(earliestTime) {
				earliestTime = event.StartTime
			}
			if event.StartTime.After(latestTime) {
				latestTime = event.StartTime
			}
		}

		// Count active vs resolved
		if event.EndTime == nil || event.EndTime.After(time.Now()) {
			activeCount++
		} else {
			resolvedCount++
		}
	}

	if len(events) > 0 {
		timeSpan := summary["timeSpan"].(map[string]interface{})
		timeSpan["earliest"] = earliestTime.Format(time.RFC3339)
		timeSpan["latest"] = latestTime.Format(time.RFC3339)
		timeSpan["span"] = latestTime.Sub(earliestTime).String()
	}

	summary["resolutionStatus"] = map[string]int{
		"active":   activeCount,
		"resolved": resolvedCount,
	}

	return summary
}

// formatTimeRange formats the time range for display
func formatTimeRange(filter *HealthEventFilter) map[string]interface{} {
	timeRange := map[string]interface{}{}
	
	if filter != nil {
		if filter.StartTime != nil {
			timeRange["startTime"] = filter.StartTime.Format(time.RFC3339)
		}
		if filter.EndTime != nil {
			timeRange["endTime"] = filter.EndTime.Format(time.RFC3339)
		}
		if len(filter.HealthStatusFilter) > 0 {
			statusFilter := make([]string, len(filter.HealthStatusFilter))
			for i, status := range filter.HealthStatusFilter {
				statusFilter[i] = string(status)
			}
			timeRange["statusFilter"] = statusFilter
		}
		if len(filter.EventTypeFilter) > 0 {
			typeFilter := make([]string, len(filter.EventTypeFilter))
			for i, eventType := range filter.EventTypeFilter {
				typeFilter[i] = string(eventType)
			}
			timeRange["eventTypeFilter"] = typeFilter
		}
	}

	return timeRange
}

// filterRecentEvents filters events to those within the specified duration
func filterRecentEvents(events []ResourceHealthEvent, duration time.Duration) []ResourceHealthEvent {
	cutoff := time.Now().Add(-duration)
	var recent []ResourceHealthEvent

	for _, event := range events {
		if event.StartTime.After(cutoff) {
			recent = append(recent, event)
		}
	}

	return recent
}

// filterActiveEvents filters events to those that are currently active
func filterActiveEvents(events []ResourceHealthEvent) []ResourceHealthEvent {
	now := time.Now()
	var active []ResourceHealthEvent

	for _, event := range events {
		if event.EndTime == nil || event.EndTime.After(now) {
			active = append(active, event)
		}
	}

	return active
}

// ValidateResourceIDs validates that the provided resource IDs are valid Azure resource IDs
func ValidateResourceIDs(resourceIDs []string) error {
	if len(resourceIDs) == 0 {
		return fmt.Errorf("at least one resource ID must be provided")
	}

	for _, resourceID := range resourceIDs {
		if err := validateSingleResourceID(resourceID); err != nil {
			return fmt.Errorf("invalid resource ID '%s': %w", resourceID, err)
		}
	}

	return nil
}

// validateSingleResourceID validates a single resource ID
func validateSingleResourceID(resourceID string) error {
	if resourceID == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}

	// Basic validation for Azure resource ID format
	parts := strings.Split(resourceID, "/")
	if len(parts) < 9 {
		return fmt.Errorf("invalid resource ID format: too few segments")
	}

	if parts[1] != "subscriptions" {
		return fmt.Errorf("invalid resource ID: must start with /subscriptions/")
	}

	if parts[3] != "resourceGroups" && parts[3] != "resourcegroups" {
		return fmt.Errorf("invalid resource ID: missing resourceGroups segment")
	}

	if parts[5] != "providers" {
		return fmt.Errorf("invalid resource ID: missing providers segment")
	}

	return nil
}

// ParseTimeFilter parses time filter strings into time.Time objects
func ParseTimeFilter(startTimeStr, endTimeStr string) (*time.Time, *time.Time, error) {
	var startTime, endTime *time.Time

	if startTimeStr != "" {
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid start time format, expected RFC3339: %w", err)
		}
		startTime = &t
	}

	if endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid end time format, expected RFC3339: %w", err)
		}
		endTime = &t
	}

	// Validate time range
	if startTime != nil && endTime != nil && startTime.After(*endTime) {
		return nil, nil, fmt.Errorf("start time cannot be after end time")
	}

	return startTime, endTime, nil
}