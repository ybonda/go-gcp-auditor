package domain

import (
	"time"
)

type UsageStatus string

const (
	UsageStatusSuccess  UsageStatus = "SUCCESS"
	UsageStatusNoAccess UsageStatus = "NO_ACCESS"
	UsageStatusError    UsageStatus = "ERROR"
)

// Project represents a GCP project
type Project struct {
	ID         string            // Project ID (e.g., "my-project")
	Name       string            // Project name
	ProjectNum int64             // Project number as string
	Labels     map[string]string // Project labels
	CreateTime time.Time         // Project creation time
}

// Service represents a GCP service and its state
type Service struct {
	Name      string // Service name (e.g., "compute", "storage", etc.)
	State     string // Service state (e.g., "ENABLED")
	Title     string // Human-readable title
	ProjectID string // Parent project ID
	Usage     *Usage // Usage metrics
}

// Usage represents service usage metrics
type Usage struct {
	RequestCount int64         // Total API requests
	Period       time.Duration // Time period for the metrics
	LastUpdated  time.Time     // Last time metrics were updated
	Status       UsageStatus
	Error        string
}

// AuditReport represents the final audit report
type AuditReport struct {
	StartTime        time.Time
	GeneratedAt      time.Time
	Period           time.Duration
	Projects         []Project
	Services         map[string][]Service
	SkippedProjects  map[string]error
	Statistics       AuditStatistics
	ProjectDurations map[string]time.Duration
}

// AuditStatistics contains summary statistics
type AuditStatistics struct {
	TotalProjects       int
	ValidProjects       int
	ExcludedProjects    int
	SkippedProjects     int
	UniqueServices      int
	ServicesWithNoUsage int
	ServiceDetails      []*ServiceDetail
}

type ServiceStatistics struct {
	TotalServices    int
	ActiveServices   int
	InactiveServices int
	NoAccessServices int
	ErrorServices    int
	TotalRequests    int64
}

type ServiceDetail struct {
	Name          string
	ProjectCount  int
	TotalRequests int64
	EnabledIn     []string
}
