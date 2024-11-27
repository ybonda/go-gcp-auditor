// internal/report/json.go
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ybonda/gcp-auditor/internal/domain"
)

type JSONReporter struct {
	outputDir string
}

// ServiceUsage represents service usage in a specific project
type ServiceUsage struct {
	ProjectID    string `json:"projectId"`
	RequestCount int64  `json:"requestCount"`
	State        string `json:"state"`
	LastUpdated  string `json:"lastUpdated,omitempty"`
}

// ServiceReport represents the structure for services-based report
type ServiceReport struct {
	Name     string         `json:"name"`
	Title    string         `json:"title,omitempty"`
	Projects []ServiceUsage `json:"projects"`
}

// ProjectService represents a service used in a project
type ProjectService struct {
	Name         string `json:"name"`
	Title        string `json:"title,omitempty"`
	RequestCount int64  `json:"requestCount"`
	State        string `json:"state"`
	LastUpdated  string `json:"lastUpdated,omitempty"`
}

// ProjectReport represents the structure for project-based report
type ProjectReport struct {
	ProjectID string           `json:"projectId"`
	Services  []ProjectService `json:"services"`
}

func NewJSONReporter(outputDir string) *JSONReporter {
	return &JSONReporter{
		outputDir: outputDir,
	}
}

func (r *JSONReporter) GenerateReport(report domain.AuditReport) error {
	// Create reports directory with timestamp
	timestamp := report.GeneratedAt.Format("20060102_150405")
	reportDir := filepath.Join(r.outputDir, timestamp)

	// Generate service-centric report
	servicesReport := r.generateServicesReport(report)
	if err := r.writeJSONReport(filepath.Join(reportDir, "services.json"), servicesReport); err != nil {
		return fmt.Errorf("failed to write services report: %w", err)
	}

	// Generate project-centric report
	projectsReport := r.generateProjectsReport(report)
	if err := r.writeJSONReport(filepath.Join(reportDir, "projects.json"), projectsReport); err != nil {
		return fmt.Errorf("failed to write projects report: %w", err)
	}

	return nil
}

func (r *JSONReporter) generateServicesReport(report domain.AuditReport) []ServiceReport {
	// Map to collect all services
	serviceMap := make(map[string]*ServiceReport)

	// Process all projects and their services
	for projectID, services := range report.Services {
		for _, service := range services {
			// Get or create service report
			serviceRep, exists := serviceMap[service.Name]
			if !exists {
				serviceRep = &ServiceReport{
					Name:  service.Name,
					Title: service.Title,
				}
				serviceMap[service.Name] = serviceRep
			}

			// Add project usage
			usage := ServiceUsage{
				ProjectID: projectID,
				State:     service.State,
			}

			if service.Usage != nil {
				usage.RequestCount = service.Usage.RequestCount
				if !service.Usage.LastUpdated.IsZero() {
					usage.LastUpdated = service.Usage.LastUpdated.Format("2006-01-02T15:04:05Z")
				}
			}

			serviceRep.Projects = append(serviceRep.Projects, usage)
		}
	}

	// Convert map to slice
	services := make([]ServiceReport, 0, len(serviceMap))
	for _, service := range serviceMap {
		services = append(services, *service)
	}

	return services
}

func (r *JSONReporter) generateProjectsReport(report domain.AuditReport) []ProjectReport {
	projects := make([]ProjectReport, 0)

	// Process each project
	for projectID, services := range report.Services {
		projectReport := ProjectReport{
			ProjectID: projectID,
			Services:  make([]ProjectService, 0, len(services)),
		}

		// Add each service
		for _, service := range services {
			projectService := ProjectService{
				Name:  service.Name,
				Title: service.Title,
				State: service.State,
			}

			if service.Usage != nil {
				projectService.RequestCount = service.Usage.RequestCount
				if !service.Usage.LastUpdated.IsZero() {
					projectService.LastUpdated = service.Usage.LastUpdated.Format("2006-01-02T15:04:05Z")
				}
			}

			projectReport.Services = append(projectReport.Services, projectService)
		}

		projects = append(projects, projectReport)
	}

	return projects
}

func (r *JSONReporter) writeJSONReport(filepath string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}
