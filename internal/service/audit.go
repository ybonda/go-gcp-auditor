package service

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/ybonda/gcp-auditor/internal/config"
	"github.com/ybonda/gcp-auditor/internal/domain"
	"github.com/ybonda/gcp-auditor/pkg/logging"
	"golang.org/x/sync/errgroup"
)

type AuditService struct {
	projectRepo domain.ProjectRepository
	serviceRepo domain.ServiceRepository
	reporters   []domain.Reporter
	config      *config.Config
	logger      *logging.Logger
}

func NewAuditService(
	projectRepo domain.ProjectRepository,
	serviceRepo domain.ServiceRepository,
	reporters []domain.Reporter,
	cfg *config.Config,
) *AuditService {
	return &AuditService{
		projectRepo: projectRepo,
		serviceRepo: serviceRepo,
		reporters:   reporters,
		config:      cfg,
		logger:      logging.NewLogger(cfg.Verbose),
	}
}

func (s *AuditService) Audit(ctx context.Context) (domain.AuditReport, error) {
	startTime := time.Now()
	report := domain.AuditReport{
		StartTime:        startTime,
		Period:           s.config.Period,
		Services:         make(map[string][]domain.Service),
		SkippedProjects:  make(map[string]error),
		ProjectDurations: make(map[string]time.Duration),
	}

	s.logger.Info("Starting GCP services audit for the last %d days...", s.config.DaysToAudit)

	// Get all projects
	s.logger.Info("Discovering GCP projects...")
	projects, err := s.projectRepo.ListProjects(ctx)
	if err != nil {
		s.logger.Error("Failed to list projects: %v", err)
		return report, err
	}
	s.logger.Info("Found %d projects", len(projects))

	// Initialize statistics
	report.Statistics.TotalProjects = len(projects)

	// Filter valid projects
	var validProjects []domain.Project
	for _, project := range projects {
		if s.projectRepo.IsValidProject(project) {
			validProjects = append(validProjects, project)
		} else {
			report.Statistics.ExcludedProjects++
		}
	}
	report.Projects = validProjects
	report.Statistics.ValidProjects = len(validProjects)

	s.logger.Info("Processing %d valid projects (excluded %d)...",
		len(validProjects),
		report.Statistics.ExcludedProjects)

	// Process projects with error group
	g, ctx := errgroup.WithContext(ctx)
	semaphore := make(chan struct{}, s.config.Concurrency)
	var mutex sync.Mutex
	processed := 0
	totalProjects := len(validProjects)

	for _, project := range validProjects {
		project := project // Create new variable for goroutine

		g.Go(func() error {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			projectStart := time.Now()
			s.logger.Debug("Processing project: %s", project.ID)

			services, err := s.serviceRepo.ListServices(ctx, project.ID, s.config.Period)
			processingDuration := time.Since(projectStart)

			mutex.Lock()
			processed++
			report.ProjectDurations[project.ID] = processingDuration
			if err != nil {
				s.logger.Error("Failed to process project %s: %v", project.ID, err)
				report.SkippedProjects[project.ID] = err
				report.Statistics.SkippedProjects++
			} else {
				report.Services[project.ID] = services
				s.logger.Info("Progress: %d/%d projects processed (%d%%) - %s completed in %s",
					processed,
					totalProjects,
					(processed*100)/totalProjects,
					project.ID,
					processingDuration.Round(time.Second))
			}
			mutex.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return report, err
	}

	report.GeneratedAt = time.Now()
	s.calculateStatistics(&report)

	duration := report.GeneratedAt.Sub(report.StartTime).Round(time.Second)
	s.logger.Info("Audit completed in %s", duration)
	s.logger.Info("Found %d unique services across %d projects",
		report.Statistics.UniqueServices,
		len(validProjects))
	s.logger.Info("Reports generated in: %s", s.config.OutputDir)

	return report, nil
}

func (s *AuditService) calculateStatistics(report *domain.AuditReport) {
	uniqueServices := make(map[string]*domain.ServiceDetail)
	servicesWithNoUsage := 0

	// Calculate service details
	for projectID, services := range report.Services {
		for _, service := range services {
			// Get or create service detail
			detail, exists := uniqueServices[service.Name]
			if !exists {
				detail = &domain.ServiceDetail{
					Name: service.Name,
				}
				uniqueServices[service.Name] = detail
			}

			detail.ProjectCount++
			detail.EnabledIn = append(detail.EnabledIn, projectID)

			if service.Usage != nil &&
				service.Usage.Status == domain.UsageStatusSuccess {
				detail.TotalRequests += service.Usage.RequestCount
				if service.Usage.RequestCount == 0 {
					servicesWithNoUsage++
				}
			}
		}
	}

	// Convert map to sorted slice
	serviceDetails := make([]*domain.ServiceDetail, 0, len(uniqueServices))
	for _, detail := range uniqueServices {
		// Sort the projects list for consistent output
		sort.Strings(detail.EnabledIn)
		serviceDetails = append(serviceDetails, detail)
	}

	// Sort by projects count (descending), and by total requests for equal project counts
	sort.Slice(serviceDetails, func(i, j int) bool {
		if serviceDetails[i].ProjectCount != serviceDetails[j].ProjectCount {
			return serviceDetails[i].ProjectCount > serviceDetails[j].ProjectCount
		}
		// Secondary sort by total requests when project counts are equal
		return serviceDetails[i].TotalRequests > serviceDetails[j].TotalRequests
	})

	// Update statistics
	report.Statistics.UniqueServices = len(uniqueServices)
	report.Statistics.ServicesWithNoUsage = servicesWithNoUsage
	report.Statistics.ServiceDetails = serviceDetails
}
