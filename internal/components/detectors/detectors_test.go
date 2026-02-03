package detectors

import (
	"context"
	"testing"
	"time"
)

func TestValidateTimeParameters(t *testing.T) {
	now := time.Now()
	validStart := now.Add(-1 * time.Hour).Format(time.RFC3339)
	validEnd := now.Format(time.RFC3339)

	tests := []struct {
		name      string
		startTime string
		endTime   string
		wantErr   bool
	}{
		{
			name:      "valid time range",
			startTime: validStart,
			endTime:   validEnd,
			wantErr:   false,
		},
		{
			name:      "invalid start time format",
			startTime: "invalid-time",
			endTime:   validEnd,
			wantErr:   true,
		},
		{
			name:      "invalid end time format",
			startTime: validStart,
			endTime:   "invalid-time",
			wantErr:   true,
		},
		{
			name:      "end time before start time",
			startTime: validEnd,
			endTime:   validStart,
			wantErr:   true,
		},
		{
			name:      "time range too long (over 24h)",
			startTime: now.Add(-25 * time.Hour).Format(time.RFC3339),
			endTime:   now.Format(time.RFC3339),
			wantErr:   true,
		},
		{
			name:      "start time too old (over 30 days)",
			startTime: now.AddDate(0, 0, -31).Format(time.RFC3339),
			endTime:   now.AddDate(0, 0, -30).Format(time.RFC3339),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTimeParameters(tt.startTime, tt.endTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTimeParameters() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCategory(t *testing.T) {
	tests := []struct {
		name     string
		category string
		wantErr  bool
	}{
		{
			name:     "valid category - Best Practices",
			category: "Best Practices",
			wantErr:  false,
		},
		{
			name:     "valid category - Node Health",
			category: "Node Health",
			wantErr:  false,
		},
		{
			name:     "invalid category",
			category: "Invalid Category",
			wantErr:  true,
		},
		{
			name:     "empty category",
			category: "",
			wantErr:  true,
		},
		{
			// Test case updated: validateCategory() uses strings.EqualFold() for case-insensitive comparison,
			// so "best practices" should be accepted as valid (same as "Best Practices")
			name:     "case insensitive validation",
			category: "best practices",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCategory(tt.category)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCategory() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleAksDetector_OperationValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		params    map[string]interface{}
		wantErr   bool
		errString string
	}{
		{
			name: "missing operation parameter",
			params: map[string]interface{}{
				"aks_resource_id": "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster",
			},
			wantErr:   true,
			errString: "missing or invalid operation parameter",
		},
		{
			name: "invalid operation parameter",
			params: map[string]interface{}{
				"operation":       "invalid_op",
				"aks_resource_id": "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster",
			},
			wantErr:   true,
			errString: "invalid operation 'invalid_op', must be one of: list, run, run_by_category",
		},
		{
			name: "list operation - missing aks_resource_id",
			params: map[string]interface{}{
				"operation": "list",
			},
			wantErr:   true,
			errString: "missing or invalid aks_resource_id parameter",
		},
		{
			name: "run operation - missing detector_name",
			params: map[string]interface{}{
				"operation":       "run",
				"aks_resource_id": "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster",
			},
			wantErr:   true,
			errString: "missing or invalid detector_name parameter",
		},
		{
			name: "run operation - missing start_time",
			params: map[string]interface{}{
				"operation":       "run",
				"aks_resource_id": "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster",
				"detector_name":   "test-detector",
			},
			wantErr:   true,
			errString: "missing or invalid start_time parameter",
		},
		{
			name: "run_by_category operation - missing category",
			params: map[string]interface{}{
				"operation":       "run_by_category",
				"aks_resource_id": "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster",
			},
			wantErr:   true,
			errString: "missing or invalid category parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := HandleAksDetector(ctx, tt.params, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleAksDetector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errString != "" && err.Error() != tt.errString {
				t.Errorf("HandleAksDetector() error = %v, want error containing %v", err, tt.errString)
			}
		})
	}
}
