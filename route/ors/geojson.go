// Code generated by github.com/swaggest/json-cli v1.11.1, DO NOT EDIT.

// Package entities contains JSON mapping structures.
package ors

// GeoJSON structure is generated from "#".
type GeoJSON struct {
	Type     string            `json:"type,omitempty"`
	Metadata *Metadata         `json:"metadata,omitempty"`
	Bbox     []float64         `json:"bbox,omitempty"`
	Features []FeaturesElement `json:"features,omitempty"`
}

// Metadata structure is generated from "#/definitions/metadata".
type Metadata struct {
	Attribution string          `json:"attribution,omitempty"`
	Service     string          `json:"service,omitempty"`
	Timestamp   int64           `json:"timestamp,omitempty"`
	Query       *MetadataQuery  `json:"query,omitempty"`
	Engine      *MetadataEngine `json:"engine,omitempty"`
}

// MetadataQuery structure is generated from "#/definitions/metadata.query".
type MetadataQuery struct {
	Coordinates [][]float64 `json:"coordinates,omitempty"`
	Profile     string      `json:"profile,omitempty"`
	Format      string      `json:"format,omitempty"`
}

// MetadataEngine structure is generated from "#/definitions/metadata.engine".
type MetadataEngine struct {
	Version   string `json:"version,omitempty"`
	BuildDate string `json:"build_date,omitempty"`
	GraphDate string `json:"graph_date,omitempty"`
}

// FeaturesElement structure is generated from "#/definitions/features.element".
type FeaturesElement struct {
	Bbox       []float64                  `json:"bbox,omitempty"`
	Type       string                     `json:"type,omitempty"`
	Properties *FeaturesElementProperties `json:"properties,omitempty"`
	Geometry   *FeaturesElementGeometry   `json:"geometry,omitempty"`
}

// FeaturesElementProperties structure is generated from "#/definitions/features.element.properties".
type FeaturesElementProperties struct {
	Transfers int64                                      `json:"transfers,omitempty"`
	Fare      int64                                      `json:"fare,omitempty"`
	Segments  []FeaturesElementPropertiesSegmentsElement `json:"segments,omitempty"`
	WayPoints []int64                                    `json:"way_points,omitempty"`
	Summary   *FeaturesElementPropertiesSummary          `json:"summary,omitempty"`
}

// FeaturesElementPropertiesSegmentsElement structure is generated from "#/definitions/features.element.properties.segments.element".
type FeaturesElementPropertiesSegmentsElement struct {
	Distance float64                                                `json:"distance,omitempty"`
	Duration float64                                                `json:"duration,omitempty"`
	Steps    []FeaturesElementPropertiesSegmentsElementStepsElement `json:"steps,omitempty"`
}

// FeaturesElementPropertiesSegmentsElementStepsElement structure is generated from "#/definitions/features.element.properties.segments.element.steps.element".
type FeaturesElementPropertiesSegmentsElementStepsElement struct {
	Distance    float64 `json:"distance,omitempty"`
	Duration    float64 `json:"duration,omitempty"`
	Type        int64   `json:"type,omitempty"`
	Instruction string  `json:"instruction,omitempty"`
	Name        string  `json:"name,omitempty"`
	WayPoints   []int64 `json:"way_points,omitempty"`
	ExitNumber  int64   `json:"exit_number,omitempty"`
}

// FeaturesElementPropertiesSummary structure is generated from "#/definitions/features.element.properties.summary".
type FeaturesElementPropertiesSummary struct {
	Distance float64 `json:"distance,omitempty"`
	Duration float64 `json:"duration,omitempty"`
}

// FeaturesElementGeometry structure is generated from "#/definitions/features.element.geometry".
type FeaturesElementGeometry struct {
	Coordinates [][]float64 `json:"coordinates,omitempty"`
	Type        string      `json:"type,omitempty"`
}
