package detectors

// DetectorListResponse represents the API response for listing detectors
type DetectorListResponse struct {
	Value []Detector `json:"value"`
}

// Detector represents a single detector metadata
type Detector struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Type       string             `json:"type"`
	Location   string             `json:"location"`
	Properties DetectorProperties `json:"properties"`
}

// DetectorProperties contains detector metadata
type DetectorProperties struct {
	Metadata DetectorMetadata `json:"metadata"`
	Status   DetectorStatus   `json:"status"`
}

// DetectorMetadata contains detector information
type DetectorMetadata struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// DetectorStatus contains detector status
type DetectorStatus struct {
	Message  *string `json:"message"`
	StatusID int     `json:"statusId"`
}

// DetectorRunResponse represents the API response for running a detector
type DetectorRunResponse struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Type       string                `json:"type"`
	Location   string                `json:"location"`
	Properties DetectorRunProperties `json:"properties"`
}

// DetectorRunProperties contains detector run results
type DetectorRunProperties struct {
	Dataset  []DetectorDataset `json:"dataset"`
	Metadata DetectorMetadata  `json:"metadata"`
	Status   DetectorStatus    `json:"status"`
}

// DetectorDataset represents detector output data
type DetectorDataset struct {
	RenderingProperties RenderingProperties `json:"renderingProperties"`
	Table               DetectorTable       `json:"table"`
}

// RenderingProperties defines how to display results
type RenderingProperties struct {
	Description *string `json:"description"`
	IsVisible   bool    `json:"isVisible"`
	Title       *string `json:"title"`
	Type        int     `json:"type"`
}

// DetectorTable contains tabular detector results
type DetectorTable struct {
	Columns   []DetectorColumn `json:"columns"`
	Rows      [][]interface{}  `json:"rows"`
	TableName string           `json:"tableName"`
}

// DetectorColumn defines table column metadata
type DetectorColumn struct {
	ColumnName string  `json:"columnName"`
	ColumnType *string `json:"columnType"`
	DataType   string  `json:"dataType"`
}
