package domain

import (
	"context"
	"time"
)

// ProjectRepository handles GCP project operations
type ProjectRepository interface {
	ListProjects(ctx context.Context) ([]Project, error)
	IsValidProject(project Project) bool
}

// ServiceRepository handles GCP service operations
type ServiceRepository interface {
	ListServices(ctx context.Context, projectID string, period time.Duration) ([]Service, error)
	GetServiceUsage(ctx context.Context, projectID, serviceName string, period time.Duration) (*Usage, error)
}

// Reporter generates audit reports
type Reporter interface {
	GenerateReport(report AuditReport) error
}

// Auditor defines the main audit operation
type Auditor interface {
	Audit(ctx context.Context) (AuditReport, error)
}
