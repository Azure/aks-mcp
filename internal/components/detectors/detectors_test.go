package detectors

import (
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
			name:     "case sensitive validation",
			category: "best practices",
			wantErr:  true,
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
