package resourcehealth

import (
	"time"
)

// HealthStatus represents the current health status of a resource
type HealthStatus string

const (
	HealthStatusAvailable   HealthStatus = "Available"
	HealthStatusUnavailable HealthStatus = "Unavailable"
	HealthStatusDegraded    HealthStatus = "Degraded"
	HealthStatusUnknown     HealthStatus = "Unknown"
)

// HealthEventType represents the type of health event
type HealthEventType string

const (
	HealthEventTypeServiceIssue    HealthEventType = "ServiceIssue"
	HealthEventTypePlannedMaintenance HealthEventType = "PlannedMaintenance"
	HealthEventTypeUnplannedMaintenance HealthEventType = "UnplannedMaintenance"
	HealthEventTypeHealthAdvisory  HealthEventType = "HealthAdvisory"
	HealthEventTypeSecurity        HealthEventType = "Security"
)

// ResourceHealthStatus represents the current health status of a resource
type ResourceHealthStatus struct {
	ResourceID      string                 `json:"resourceId"`
	ResourceName    string                 `json:"resourceName"`
	ResourceType    string                 `json:"resourceType"`
	Status          HealthStatus           `json:"status"`
	StatusSummary   string                 `json:"statusSummary"`
	LastUpdated     time.Time              `json:"lastUpdated"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
	RecommendedActions []RecommendedAction   `json:"recommendedActions,omitempty"`
}

// ResourceHealthEvent represents a historical health event
type ResourceHealthEvent struct {
	ID               string                 `json:"id"`
	ResourceID       string                 `json:"resourceId"`
	EventType        HealthEventType        `json:"eventType"`
	Title            string                 `json:"title"`
	Summary          string                 `json:"summary"`
	Description      string                 `json:"description"`
	Status           HealthStatus           `json:"status"`
	Level            string                 `json:"level"`
	StartTime        time.Time              `json:"startTime"`
	EndTime          *time.Time             `json:"endTime,omitempty"`
	Duration         string                 `json:"duration,omitempty"`
	ImpactScope      string                 `json:"impactScope,omitempty"`
	RootCause        string                 `json:"rootCause,omitempty"`
	Resolution       string                 `json:"resolution,omitempty"`
	Properties       map[string]interface{} `json:"properties,omitempty"`
	RecommendedActions []RecommendedAction   `json:"recommendedActions,omitempty"`
}

// RecommendedAction represents a recommended action for a health issue
type RecommendedAction struct {
	Action      string `json:"action"`
	ActionText  string `json:"actionText"`
	ActionURL   string `json:"actionUrl,omitempty"`
	Description string `json:"description,omitempty"`
}

// HealthEventFilter represents filtering options for health events
type HealthEventFilter struct {
	StartTime        *time.Time       `json:"startTime,omitempty"`
	EndTime          *time.Time       `json:"endTime,omitempty"`
	HealthStatusFilter []HealthStatus `json:"healthStatusFilter,omitempty"`
	EventTypeFilter  []HealthEventType `json:"eventTypeFilter,omitempty"`
}

// ResourceHealthSummary represents a summary of resource health
type ResourceHealthSummary struct {
	ResourceID          string            `json:"resourceId"`
	ResourceName        string            `json:"resourceName"`
	CurrentStatus       HealthStatus      `json:"currentStatus"`
	LastUpdated         time.Time         `json:"lastUpdated"`
	HealthScore         int               `json:"healthScore"` // 0-100
	ActiveIssuesCount   int               `json:"activeIssuesCount"`
	TotalEventsCount    int               `json:"totalEventsCount"`
	AvailabilityPercent float64           `json:"availabilityPercent"`
	Metrics             map[string]float64 `json:"metrics,omitempty"`
}