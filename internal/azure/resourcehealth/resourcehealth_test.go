package resourcehealth

import (
	"context"
	"testing"
	"time"
)

func TestValidateResourceIDs(t *testing.T) {
	tests := []struct {
		name        string
		resourceIDs []string
		shouldError bool
	}{
		{
			name: "valid resource IDs",
			resourceIDs: []string{
				"/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/aks-test",
				"/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-test/providers/Microsoft.Compute/virtualMachines/vm-test",
			},
			shouldError: false,
		},
		{
			name:        "empty resource IDs array",
			resourceIDs: []string{},
			shouldError: true,
		},
		{
			name:        "nil resource IDs array",
			resourceIDs: nil,
			shouldError: true,
		},
		{
			name: "invalid resource ID - too short",
			resourceIDs: []string{
				"/subscriptions/12345678-1234-1234-1234-123456789012",
			},
			shouldError: true,
		},
		{
			name: "invalid resource ID - wrong format",
			resourceIDs: []string{
				"invalid-resource-id",
			},
			shouldError: true,
		},
		{
			name: "mixed valid and invalid resource IDs",
			resourceIDs: []string{
				"/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/aks-test",
				"invalid-resource-id",
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResourceIDs(tt.resourceIDs)
			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseTimeFilter(t *testing.T) {
	tests := []struct {
		name        string
		startTime   string
		endTime     string
		shouldError bool
	}{
		{
			name:        "valid time range",
			startTime:   "2023-01-01T00:00:00Z",
			endTime:     "2023-01-02T00:00:00Z",
			shouldError: false,
		},
		{
			name:        "empty times",
			startTime:   "",
			endTime:     "",
			shouldError: false,
		},
		{
			name:        "only start time",
			startTime:   "2023-01-01T00:00:00Z",
			endTime:     "",
			shouldError: false,
		},
		{
			name:        "only end time",
			startTime:   "",
			endTime:     "2023-01-02T00:00:00Z",
			shouldError: false,
		},
		{
			name:        "invalid start time format",
			startTime:   "invalid-time",
			endTime:     "2023-01-02T00:00:00Z",
			shouldError: true,
		},
		{
			name:        "invalid end time format",
			startTime:   "2023-01-01T00:00:00Z",
			endTime:     "invalid-time",
			shouldError: true,
		},
		{
			name:        "start time after end time",
			startTime:   "2023-01-02T00:00:00Z",
			endTime:     "2023-01-01T00:00:00Z",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime, endTime, err := ParseTimeFilter(tt.startTime, tt.endTime)
			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Additional validation for successful cases
			if !tt.shouldError && err == nil {
				if tt.startTime != "" && startTime == nil {
					t.Errorf("expected start time to be parsed")
				}
				if tt.endTime != "" && endTime == nil {
					t.Errorf("expected end time to be parsed")
				}
			}
		})
	}
}

func TestNewResourceHealthClient(t *testing.T) {
	// Test with empty subscription ID
	_, err := NewResourceHealthClient("", nil)
	if err == nil {
		t.Error("expected error for empty subscription ID")
	}

	// Test with valid subscription ID
	client, err := NewResourceHealthClient("12345678-1234-1234-1234-123456789012", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if client == nil {
		t.Error("expected client to be created")
	}
}

func TestNewEventManager(t *testing.T) {
	// Test with empty subscription ID
	_, err := NewEventManager("", nil)
	if err == nil {
		t.Error("expected error for empty subscription ID")
	}

	// Test with valid subscription ID
	manager, err := NewEventManager("12345678-1234-1234-1234-123456789012", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if manager == nil {
		t.Error("expected event manager to be created")
	}
}

func TestGenerateMockHealthStatus(t *testing.T) {
	tests := []string{
		"aks-cluster-1",
		"vm-test",
		"storage-account",
		"app-service",
	}

	for _, resourceName := range tests {
		t.Run(resourceName, func(t *testing.T) {
			status := generateMockHealthStatus(resourceName)

			// Verify it's one of the valid statuses
			validStatuses := map[HealthStatus]bool{
				HealthStatusAvailable:   true,
				HealthStatusUnavailable: true,
				HealthStatusDegraded:    true,
				HealthStatusUnknown:     true,
			}

			if !validStatuses[status] {
				t.Errorf("unexpected health status: %s", status)
			}

			// Test determinism - same input should give same output
			status2 := generateMockHealthStatus(resourceName)
			if status != status2 {
				t.Errorf("expected deterministic result, got %s and %s", status, status2)
			}
		})
	}
}

func TestGenerateMockHealthEvents(t *testing.T) {
	resourceID := "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/aks-test"
	startTime := time.Now().Add(-30 * 24 * time.Hour)
	endTime := time.Now()

	events := generateMockHealthEvents(resourceID, startTime, endTime)

	if len(events) == 0 {
		t.Error("expected at least one event to be generated")
	}

	// Verify event structure
	for _, event := range events {
		if event.ResourceID != resourceID {
			t.Errorf("expected resource ID %s, got %s", resourceID, event.ResourceID)
		}
		if event.ID == "" {
			t.Error("expected event ID to be set")
		}
		if event.Title == "" {
			t.Error("expected event title to be set")
		}
		if event.Summary == "" {
			t.Error("expected event summary to be set")
		}
		if event.StartTime.Before(startTime) || event.StartTime.After(endTime) {
			t.Errorf("event start time %v is outside range %v to %v", event.StartTime, startTime, endTime)
		}
	}
}

func TestApplyEventFilters(t *testing.T) {
	// Create mock events
	now := time.Now()
	events := []ResourceHealthEvent{
		{
			ID:        "event1",
			EventType: HealthEventTypeServiceIssue,
			Status:    HealthStatusUnavailable,
			StartTime: now.Add(-2 * time.Hour),
		},
		{
			ID:        "event2",
			EventType: HealthEventTypePlannedMaintenance,
			Status:    HealthStatusDegraded,
			StartTime: now.Add(-1 * time.Hour),
		},
		{
			ID:        "event3",
			EventType: HealthEventTypeHealthAdvisory,
			Status:    HealthStatusAvailable,
			StartTime: now.Add(-30 * time.Minute),
		},
	}

	// Test status filter
	filter := &HealthEventFilter{
		HealthStatusFilter: []HealthStatus{HealthStatusUnavailable},
	}
	filtered := applyEventFilters(events, filter)
	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered event, got %d", len(filtered))
	}
	if filtered[0].ID != "event1" {
		t.Errorf("expected event1, got %s", filtered[0].ID)
	}

	// Test time filter
	filter = &HealthEventFilter{
		StartTime: &[]time.Time{now.Add(-90 * time.Minute)}[0],
		EndTime:   &[]time.Time{now.Add(-45 * time.Minute)}[0],
	}
	filtered = applyEventFilters(events, filter)
	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered event, got %d", len(filtered))
	}
	if filtered[0].ID != "event2" {
		t.Errorf("expected event2, got %s", filtered[0].ID)
	}

	// Test event type filter
	filter = &HealthEventFilter{
		EventTypeFilter: []HealthEventType{HealthEventTypeHealthAdvisory},
	}
	filtered = applyEventFilters(events, filter)
	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered event, got %d", len(filtered))
	}
	if filtered[0].ID != "event3" {
		t.Errorf("expected event3, got %s", filtered[0].ID)
	}

	// Test no filter
	filtered = applyEventFilters(events, nil)
	if len(filtered) != len(events) {
		t.Errorf("expected %d events with no filter, got %d", len(events), len(filtered))
	}
}

func TestGetResourceHealthStatus(t *testing.T) {
	// Create event manager
	manager, err := NewEventManager("12345678-1234-1234-1234-123456789012", nil)
	if err != nil {
		t.Fatalf("failed to create event manager: %v", err)
	}

	resourceIDs := []string{
		"/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/aks-test",
		"/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-test/providers/Microsoft.Compute/virtualMachines/vm-test",
	}

	// Test getting health status
	result, err := manager.GetResourceHealthStatus(context.Background(), resourceIDs, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Test with include history
	result, err = manager.GetResourceHealthStatus(context.Background(), resourceIDs, true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestGetResourceHealthEvents(t *testing.T) {
	// Create event manager
	manager, err := NewEventManager("12345678-1234-1234-1234-123456789012", nil)
	if err != nil {
		t.Fatalf("failed to create event manager: %v", err)
	}

	resourceID := "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-test/providers/Microsoft.ContainerService/managedClusters/aks-test"

	// Test getting events
	result, err := manager.GetResourceHealthEvents(context.Background(), resourceID, nil, nil, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Test with time range
	startTime := time.Now().Add(-7 * 24 * time.Hour)
	endTime := time.Now()
	result, err = manager.GetResourceHealthEvents(context.Background(), resourceID, &startTime, &endTime, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Test with status filter
	healthStatusFilter := []string{"Available", "Degraded"}
	result, err = manager.GetResourceHealthEvents(context.Background(), resourceID, nil, nil, healthStatusFilter)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestGenerateHealthSummary(t *testing.T) {
	statuses := []ResourceHealthStatus{
		{
			ResourceName:  "aks-test",
			Status:        HealthStatusAvailable,
			StatusSummary: "All good",
		},
		{
			ResourceName:  "vm-test",
			Status:        HealthStatusDegraded,
			StatusSummary: "Performance issues",
		},
		{
			ResourceName:  "storage-test",
			Status:        HealthStatusUnavailable,
			StatusSummary: "Service outage",
		},
	}

	summary := generateHealthSummary(statuses)

	if summary["totalResources"] != 3 {
		t.Errorf("expected total resources 3, got %v", summary["totalResources"])
	}

	statusCounts := summary["statusCounts"].(map[string]int)
	if statusCounts["Available"] != 1 {
		t.Errorf("expected 1 available resource, got %d", statusCounts["Available"])
	}
	if statusCounts["Degraded"] != 1 {
		t.Errorf("expected 1 degraded resource, got %d", statusCounts["Degraded"])
	}
	if statusCounts["Unavailable"] != 1 {
		t.Errorf("expected 1 unavailable resource, got %d", statusCounts["Unavailable"])
	}

	if summary["overallHealth"] != "Critical" {
		t.Errorf("expected overall health 'Critical', got %v", summary["overallHealth"])
	}

	issues := summary["issuesFound"].([]string)
	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}
}

func TestFilterRecentEvents(t *testing.T) {
	now := time.Now()
	events := []ResourceHealthEvent{
		{ID: "event1", StartTime: now.Add(-1 * time.Hour)},       // Recent
		{ID: "event2", StartTime: now.Add(-10 * 24 * time.Hour)}, // Old
		{ID: "event3", StartTime: now.Add(-30 * time.Minute)},    // Recent
	}

	recent := filterRecentEvents(events, 2*time.Hour)
	if len(recent) != 2 {
		t.Errorf("expected 2 recent events, got %d", len(recent))
	}

	// Check that the right events are included
	foundEvent1, foundEvent3 := false, false
	for _, event := range recent {
		if event.ID == "event1" {
			foundEvent1 = true
		}
		if event.ID == "event3" {
			foundEvent3 = true
		}
	}

	if !foundEvent1 || !foundEvent3 {
		t.Error("expected event1 and event3 to be in recent events")
	}
}

func TestFilterActiveEvents(t *testing.T) {
	now := time.Now()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(1 * time.Hour)

	events := []ResourceHealthEvent{
		{ID: "event1", EndTime: nil},         // Active (no end time)
		{ID: "event2", EndTime: &pastTime},   // Resolved
		{ID: "event3", EndTime: &futureTime}, // Active (future end time)
	}

	active := filterActiveEvents(events)
	if len(active) != 2 {
		t.Errorf("expected 2 active events, got %d", len(active))
	}

	// Check that the right events are included
	foundEvent1, foundEvent3 := false, false
	for _, event := range active {
		if event.ID == "event1" {
			foundEvent1 = true
		}
		if event.ID == "event3" {
			foundEvent3 = true
		}
	}

	if !foundEvent1 || !foundEvent3 {
		t.Error("expected event1 and event3 to be in active events")
	}
}
