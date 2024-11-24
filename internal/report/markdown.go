// internal/report/markdown.go
package report

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ybonda/gcp-auditor/internal/domain"
)

type MarkdownReporter struct {
	outputDir string
}

type projectOverview struct {
	ProjectID      string
	TotalServices  int
	ActiveServices int
	Duration       time.Duration
}

func NewMarkdownReporter(outputDir string) *MarkdownReporter {
	return &MarkdownReporter{
		outputDir: outputDir,
	}
}

func (r *MarkdownReporter) GenerateReport(report domain.AuditReport) error {
	// Create timestamped directory
	timestamp := time.Now().Format("20060102_150405")
	reportDir := filepath.Join(r.outputDir, timestamp)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// Create projects directory
	projectsDir := filepath.Join(reportDir, "projects_report")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		return fmt.Errorf("failed to create projects directory: %w", err)
	}

	// Generate main report
	if err := r.generateMainReport(reportDir, projectsDir, report); err != nil {
		return fmt.Errorf("failed to generate main report: %w", err)
	}

	// Generate individual project reports
	for projectID, services := range report.Services {
		if err := r.generateProjectReport(projectsDir, projectID, services, report.GeneratedAt); err != nil {
			return fmt.Errorf("failed to generate project report for %s: %w", projectID, err)
		}
	}

	return nil
}

func (r *MarkdownReporter) generateMainReport(reportDir, projectsDir string, report domain.AuditReport) error {
	filename := filepath.Join(reportDir, "report.md")
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create main report file: %w", err)
	}
	defer file.Close()

	executionTime := report.GeneratedAt.Sub(report.StartTime)

	// Write report header
	fmt.Fprintf(file, "# GCP Services Audit Report\n\n")
	fmt.Fprintf(file, "## Execution Information\n\n")
	fmt.Fprintf(file, "- Start time: %s\n", report.StartTime.Format(time.RFC3339))
	fmt.Fprintf(file, "- End time: %s\n", report.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(file, "- Total execution time: %s\n\n", executionTime.Round(time.Second))

	// Write summary statistics
	fmt.Fprintf(file, "## Summary\n\n")
	fmt.Fprintf(file, "- Analysis Period: %d days\n", report.Period/(24*time.Hour))
	fmt.Fprintf(file, "- Date Range: %s to %s\n",
		report.StartTime.Add(-report.Period).Format("2006-01-02"),
		report.StartTime.Format("2006-01-02"))
	fmt.Fprintf(file, "- Total Projects: %d\n", report.Statistics.TotalProjects)
	fmt.Fprintf(file, "- Valid Projects: %d\n", report.Statistics.ValidProjects)
	fmt.Fprintf(file, "- Excluded Projects: %d\n", report.Statistics.ExcludedProjects)
	fmt.Fprintf(file, "- Skipped Projects: %d\n", report.Statistics.SkippedProjects)
	fmt.Fprintf(file, "- Unique Services: %d\n\n", report.Statistics.UniqueServices)

	// Create and sort project overviews
	projects := make([]projectOverview, 0, len(report.Services))
	for projectID, services := range report.Services {
		activeCount := 0
		for _, service := range services {
			if service.Usage != nil &&
				service.Usage.Status == domain.UsageStatusSuccess &&
				service.Usage.RequestCount > 0 {
				activeCount++
			}
		}
		projects = append(projects, projectOverview{
			ProjectID:      projectID,
			TotalServices:  len(services),
			ActiveServices: activeCount,
			Duration:       report.ProjectDurations[projectID],
		})
	}

	// Sort projects by total services count (descending)
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].TotalServices > projects[j].TotalServices
	})

	projectsDirRelative := filepath.Base(projectsDir)

	// Write projects overview
	fmt.Fprintf(file, "## Projects Overview\n\n")
	fmt.Fprintf(file, "| Project ID | Services | Active Services* | Processing Time |\n")
	fmt.Fprintf(file, "|------------|----------|------------------|----------------|\n")

	for _, project := range projects {
		// Use the relative path in the link
		fmt.Fprintf(file, "| [%s](./%s/%s.md) | %d | %d | %s |\n",
			project.ProjectID,
			projectsDirRelative,
			project.ProjectID,
			project.TotalServices,
			project.ActiveServices,
			project.Duration.Round(time.Second),
		)
	}
	fmt.Fprintf(file, "\n*Active services are those with request count > 0 in the specified period\n\n")

	// Calculate and show timing statistics
	var totalProjectTime time.Duration
	var maxDuration time.Duration
	var slowestProject string
	for _, p := range projects {
		totalProjectTime += p.Duration
		if p.Duration > maxDuration {
			maxDuration = p.Duration
			slowestProject = p.ProjectID
		}
	}

	if len(projects) > 0 {
		avgDuration := totalProjectTime / time.Duration(len(projects))

		fmt.Fprintf(file, "## Timing Statistics\n\n")
		fmt.Fprintf(file, "- Total execution time: %s\n", executionTime.Round(time.Second))
		fmt.Fprintf(file, "- Average project processing time: %s\n", avgDuration.Round(time.Second))
		fmt.Fprintf(file, "- Slowest project: %s (%s)\n\n", slowestProject, maxDuration.Round(time.Second))
	}

	// Write skipped projects if any
	if len(report.SkippedProjects) > 0 {
		fmt.Fprintf(file, "## Skipped Projects\n\n")
		fmt.Fprintf(file, "| Project ID | Error |\n")
		fmt.Fprintf(file, "|------------|-------|\n")

		skippedIDs := make([]string, 0, len(report.SkippedProjects))
		for projectID := range report.SkippedProjects {
			skippedIDs = append(skippedIDs, projectID)
		}
		sort.Strings(skippedIDs)

		for _, projectID := range skippedIDs {
			fmt.Fprintf(file, "| %s | %s |\n", projectID, report.SkippedProjects[projectID])
		}
		fmt.Fprintf(file, "\n")
	}

	return nil
}

func (r *MarkdownReporter) generateProjectReport(reportDir, projectID string, services []domain.Service, generatedAt time.Time) error {
	filename := filepath.Join(reportDir, fmt.Sprintf("%s.md", projectID))
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create project report file: %w", err)
	}
	defer file.Close()

	// Calculate project statistics
	stats := calculateProjectStats(services)

	// Write project report header
	fmt.Fprintf(file, "# Project: %s\n\n", projectID)
	fmt.Fprintf(file, "Generated on: %s\n\n", generatedAt.Format(time.RFC3339))

	// Write project summary
	fmt.Fprintf(file, "## Summary\n\n")
	fmt.Fprintf(file, "- Total Services: %d\n", stats.TotalServices)
	fmt.Fprintf(file, "- Active Services: %d\n", stats.ActiveServices)
	fmt.Fprintf(file, "- Inactive Services: %d\n", stats.InactiveServices)
	fmt.Fprintf(file, "- Services without access to metrics: %d\n", stats.NoAccessServices)
	fmt.Fprintf(file, "- Services with errors: %d\n", stats.ErrorServices)
	fmt.Fprintf(file, "- Total Requests: %d\n\n", stats.TotalRequests)

	// Collect and sort active services
	var activeServices []domain.Service
	for _, service := range services {
		if service.Usage != nil && service.Usage.Status == domain.UsageStatusSuccess && service.Usage.RequestCount > 0 {
			activeServices = append(activeServices, service)
		}
	}

	// Sort active services by request count (descending)
	sort.Slice(activeServices, func(i, j int) bool {
		return activeServices[i].Usage.RequestCount > activeServices[j].Usage.RequestCount
	})

	// Write active services
	if len(activeServices) > 0 {
		fmt.Fprintf(file, "## Active Services\n\n")
		fmt.Fprintf(file, "| Service Name | State | Request Count | Last Updated |\n")
		fmt.Fprintf(file, "|--------------|-------|---------------|---------------|\n")

		for _, service := range activeServices {
			fmt.Fprintf(file, "| %s | %s | %d | %s |\n",
				service.Name,
				service.State,
				service.Usage.RequestCount,
				service.Usage.LastUpdated.Format("2006-01-02 15:04:05"),
			)
		}
		fmt.Fprintf(file, "\n")
	}

	// Write inactive services
	if stats.InactiveServices > 0 {
		fmt.Fprintf(file, "## Inactive Services\n\n")
		fmt.Fprintf(file, "The following services are enabled but had no requests during the audit period:\n\n")
		for _, service := range services {
			if service.Usage != nil && service.Usage.Status == domain.UsageStatusSuccess && service.Usage.RequestCount == 0 {
				fmt.Fprintf(file, "- %s\n", service.Name)
			}
		}
		fmt.Fprintf(file, "\n")
	}

	// Write services without access
	if stats.NoAccessServices > 0 {
		fmt.Fprintf(file, "## Services Without Metrics Access\n\n")
		fmt.Fprintf(file, "Unable to determine usage for the following services due to insufficient permissions:\n\n")
		for _, service := range services {
			if service.Usage != nil && service.Usage.Status == domain.UsageStatusNoAccess {
				fmt.Fprintf(file, "- %s\n", service.Name)
			}
		}
		fmt.Fprintf(file, "\n")
	}

	// Write services with errors
	if stats.ErrorServices > 0 {
		fmt.Fprintf(file, "## Services With Errors\n\n")
		fmt.Fprintf(file, "The following services encountered errors while fetching metrics:\n\n")
		for _, service := range services {
			if service.Usage != nil && service.Usage.Status == domain.UsageStatusError {
				fmt.Fprintf(file, "- %s: %s\n", service.Name, service.Usage.Error)
			}
		}
	}

	return nil
}

func calculateProjectStats(services []domain.Service) domain.ServiceStatistics {
	stats := domain.ServiceStatistics{
		TotalServices: len(services),
	}

	for _, service := range services {
		if service.Usage == nil {
			continue
		}

		switch service.Usage.Status {
		case domain.UsageStatusSuccess:
			if service.Usage.RequestCount > 0 {
				stats.ActiveServices++
				stats.TotalRequests += service.Usage.RequestCount
			} else {
				stats.InactiveServices++
			}
		case domain.UsageStatusNoAccess:
			stats.NoAccessServices++
		case domain.UsageStatusError:
			stats.ErrorServices++
		}
	}

	return stats
}
