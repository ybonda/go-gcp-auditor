package gcp

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/ybonda/gcp-auditor/internal/domain"
	"github.com/ybonda/gcp-auditor/pkg/logging"
	resourcemanager "google.golang.org/api/cloudresourcemanager/v1"
)

var systemPattern = regexp.MustCompile(`^sys-\d+`)

type ProjectRepository struct {
	service *resourcemanager.Service
	logger  *logging.Logger
}

func NewProjectRepository(service *resourcemanager.Service, verbose bool) *ProjectRepository {
	return &ProjectRepository{
		service: service,
		logger:  logging.NewLogger(verbose),
	}
}

func (r *ProjectRepository) ListProjects(ctx context.Context) ([]domain.Project, error) {
	var projects []domain.Project
	pageToken := ""
	pageCount := 0
	r.logger.Debug("Starting project listing")

	for {
		pageCount++
		r.logger.Debug("Fetching projects page %d ", pageCount)

		// Create list request with page token
		call := r.service.Projects.List().Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		// Execute the request
		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list projects on page %d: %w", pageCount, err)
		}

		r.logger.Debug("Page %d: received %d projects", pageCount, len(resp.Projects))

		// Process projects from this page
		currentPageProjects := 0
		for _, p := range resp.Projects {
			createTime, err := time.Parse(time.RFC3339, p.CreateTime)
			if err != nil {
				r.logger.Debug("Failed to parse create time for project %s: %v", p.ProjectId, err)
				createTime = time.Time{}
			}

			projects = append(projects, domain.Project{
				ID:         p.ProjectId,
				Name:       p.Name,
				ProjectNum: p.ProjectNumber,
				Labels:     p.Labels,
				CreateTime: createTime,
			})
			currentPageProjects++
		}

		r.logger.Debug("Page %d: processed %d projects (total so far: %d)",
			pageCount, currentPageProjects, len(projects))

		// Check if there are more pages
		pageToken = resp.NextPageToken
		if pageToken == "" {
			r.logger.Debug("No more pages available after page %d", pageCount)
			break
		}
	}

	r.logger.Debug("Completed project listing: %d pages, %d total projects",
		pageCount, len(projects))

	return projects, nil
}

func (r *ProjectRepository) IsValidProject(project domain.Project) bool {
	return !systemPattern.MatchString(project.ID)
}
