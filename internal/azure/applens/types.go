package applens

import (
	"time"
)

// DetectorInfo represents information about an AppLens detector
type DetectorInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// DetectorResponse represents the response from executing a detector
type DetectorResponse struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	StartTime time.Time              `json:"startTime"`
	EndTime   time.Time              `json:"endTime"`
	Data      []DetectorData         `json:"data"`
	Status    string                 `json:"status"`
	Insights  []DetectorInsight      `json:"insights,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// DetectorData represents data returned by a detector
type DetectorData struct {
	Table               DetectorTable          `json:"table,omitempty"`
	RenderingProperties map[string]interface{} `json:"renderingProperties,omitempty"`
}

// DetectorTable represents tabular data from a detector
type DetectorTable struct {
	TableName string           `json:"tableName"`
	Columns   []DetectorColumn `json:"columns"`
	Rows      [][]interface{}  `json:"rows"`
}

// DetectorColumn represents a column in detector table data
type DetectorColumn struct {
	ColumnName string `json:"columnName"`
	DataType   string `json:"dataType"`
	ColumnType string `json:"columnType"`
}

// DetectorInsight represents an insight from a detector
type DetectorInsight struct {
	Message  string                 `json:"message"`
	Status   string                 `json:"status"`
	Level    string                 `json:"level"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AppLensOptions represents options for AppLens detector execution
type AppLensOptions struct {
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
	TimeRange string     `json:"timeRange,omitempty"`
	Category  string     `json:"category,omitempty"`
}
