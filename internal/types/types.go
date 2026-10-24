// Package types defines the core domain model shared across Nexogent.
package types

import "time"

// Severity levels, normalized across all sources.
type Severity string

const (
	SevCritical      Severity = "CRITICAL"
	SevHigh          Severity = "HIGH"
	SevMedium        Severity = "MEDIUM"
	SevLow           Severity = "LOW"
	SevInformational Severity = "INFORMATIONAL"
)

// Resource is one discovered AWS resource, grouped by service in the dashboard.
type Resource struct {
	ID        string            `json:"id"`         // ARN or unique id
	Service   string            `json:"service"`    // ec2, s3, iam, rds, lambda, dynamodb, ecs, eks
	Type      string            `json:"type"`       // e.g. "instance", "bucket", "role"
	Name      string            `json:"name"`       // human-friendly name
	Region    string            `json:"region"`     // "global" for IAM
	AccountID string            `json:"account_id"`
	Tags      map[string]string `json:"tags,omitempty"`
}

// RiskFactors are the four rubric inputs (each 0..1).
type RiskFactors struct {
	Severity             float64 `json:"severity"`
	Exposure             float64 `json:"exposure"`
	Reachability         float64 `json:"reachability"`
	ResourceCriticality  float64 `json:"resource_criticality"`
}

// Finding is one normalized security finding on a resource.
type Finding struct {
	ID               string      `json:"id"`
	Source           string      `json:"source"`            // own-check | prowler
	NativeSeverity   string      `json:"native_severity"`
	Severity         Severity    `json:"severity"`
	RiskScore        float64     `json:"risk_score"`        // 0..100, from ranking
	Service          string      `json:"service"`
	ResourceID       string      `json:"resource_id"`
	Region           string      `json:"region"`
	AccountID        string      `json:"account_id"`
	Title            string      `json:"title"`
	Description      string      `json:"description"`
	Recommendation   string      `json:"recommendation"`
	FrameworkControl string      `json:"framework_control,omitempty"`
	Factors          RiskFactors `json:"factors"`
	FirstSeen        time.Time   `json:"first_seen"`
}

// Explanation is the AI/offline explanation of a finding.
type Explanation struct {
	Problem        string   `json:"problem"`
	WhyDangerous   string   `json:"why_dangerous"`
	Impact         string   `json:"impact"`
	HowToFix       []string `json:"how_to_fix"`
	ConsolePath    string   `json:"aws_console_path"`
	Provider       string   `json:"provider"` // bedrock | offline
	Model          string   `json:"model"`
	Note           string   `json:"note,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// PostureSummary aggregates counts for the dashboard.
type PostureSummary struct {
	Total      int            `json:"total"`
	BySeverity map[string]int `json:"by_severity"`
	ByService  map[string]int `json:"by_service"`
	BySource   map[string]int `json:"by_source"`
}

// PostureReview is the full output of one review run.
type PostureReview struct {
	ReviewID        string           `json:"review_id"`
	AccountID       string           `json:"account_id"`
	Regions         []string         `json:"regions"`
	Status          string           `json:"status"`
	CreatedAt       time.Time        `json:"created_at"`
	DurationSeconds float64          `json:"duration_seconds"`
	PostureScore    int              `json:"posture_score"`
	Rating          string           `json:"rating"`
	Summary         PostureSummary   `json:"summary"`
	Findings        []Finding        `json:"findings"`
	Resources       []Resource       `json:"resources"`
	SourcesRun      []string         `json:"sources_run"`
	CollectorErrors []CollectorError `json:"collector_errors"`
	Warnings        []string         `json:"warnings"`
}

// CollectorError records a source that could not run.
type CollectorError struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}
