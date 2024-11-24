// internal/repository/gcp/service.go
package gcp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/ybonda/gcp-auditor/internal/domain"
	"github.com/ybonda/gcp-auditor/pkg/logging"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/iterator"
	serviceusage "google.golang.org/api/serviceusage/v1"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ServiceRepository struct {
	usageService     *serviceusage.Service
	monitoringClient *monitoring.MetricClient
	logger           *logging.Logger
	workerCount      int
}

func NewServiceRepository(
	usageService *serviceusage.Service,
	monitoringClient *monitoring.MetricClient,
	verbose bool,
) *ServiceRepository {
	return &ServiceRepository{
		usageService:     usageService,
		monitoringClient: monitoringClient,
		logger:           logging.NewLogger(verbose),
		workerCount:      10,
	}
}

type serviceWork struct {
	service *serviceusage.GoogleApiServiceusageV1Service
	index   int
}

func (r *ServiceRepository) ListServices(ctx context.Context, projectID string, period time.Duration) ([]domain.Service, error) {
	// First, get all services
	services, err := r.listAllServices(ctx, projectID)
	if err != nil {
		return nil, err
	}

	r.logger.Debug("Found %d services for project %s", len(services), projectID)

	// Create channels for work distribution and results
	workChan := make(chan serviceWork, len(services))
	resultsChan := make(chan domain.Service, len(services))

	// Create error group for concurrent processing
	g, ctx := errgroup.WithContext(ctx)

	// Start workers
	for i := 0; i < r.workerCount; i++ {
		g.Go(func() error {
			for work := range workChan {
				serviceName := cleanServiceName(work.service.Name)
				service := domain.Service{
					Name:      serviceName,
					State:     work.service.State,
					ProjectID: projectID,
				}

				if work.service.Config != nil {
					service.Title = work.service.Config.Title
				}

				// Get usage metrics with timeout
				usageCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				usage, err := r.GetServiceUsage(usageCtx, projectID, serviceName, period)
				cancel()

				if err != nil {
					r.logger.Debug("Failed to get usage for service %s %s: %v", serviceName, projectID, err)
					service.Usage = &domain.Usage{}
				} else {
					service.Usage = usage
				}

				select {
				case resultsChan <- service:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	}

	// Send work to workers
	go func() {
		for i, service := range services {
			select {
			case workChan <- serviceWork{service: service, index: i}:
			case <-ctx.Done():
				return
			}
		}
		close(workChan)
	}()

	// Collect results
	results := make([]domain.Service, len(services))
	go func() {
		for i := 0; i < len(services); i++ {
			select {
			case result := <-resultsChan:
				results[i] = result
			case <-ctx.Done():
				return
			}
		}
		close(resultsChan)
	}()

	// Wait for all workers to complete
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("error processing services: %w", err)
	}

	return results, nil
}

func (r *ServiceRepository) listAllServices(ctx context.Context, projectID string) ([]*serviceusage.GoogleApiServiceusageV1Service, error) {
	var services []*serviceusage.GoogleApiServiceusageV1Service
	var mu sync.Mutex
	pageToken := ""

	for {
		parent := fmt.Sprintf("projects/%s", projectID)
		call := r.usageService.Services.List(parent).Filter("state:ENABLED").Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list services for project %s: %w", projectID, err)
		}

		mu.Lock()
		services = append(services, resp.Services...)
		mu.Unlock()

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return services, nil
}

func (r *ServiceRepository) GetServiceUsage(
	ctx context.Context,
	projectID string,
	serviceName string,
	period time.Duration,
) (*domain.Usage, error) {
	endTime := time.Now()
	startTime := endTime.Add(-period)

	usage := &domain.Usage{
		Period:      period,
		LastUpdated: endTime,
		Status:      domain.UsageStatusSuccess,
	}

	if !strings.HasSuffix(serviceName, ".googleapis.com") {
		serviceName = serviceName + ".googleapis.com"
	}

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   fmt.Sprintf("projects/%s", projectID),
		Filter: fmt.Sprintf(`metric.type = "serviceruntime.googleapis.com/api/request_count" AND resource.labels.service = "%s"`, serviceName),
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(startTime),
			EndTime:   timestamppb.New(endTime),
		},
		Aggregation: &monitoringpb.Aggregation{
			AlignmentPeriod:    durationpb.New(24 * time.Hour),
			PerSeriesAligner:   monitoringpb.Aggregation_ALIGN_SUM,
			CrossSeriesReducer: monitoringpb.Aggregation_REDUCE_SUM,
		},
	}

	it := r.monitoringClient.ListTimeSeries(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "PermissionDenied") {
				usage.Status = domain.UsageStatusNoAccess
				usage.Error = "No access to monitoring data"
				return usage, nil
			}
			usage.Status = domain.UsageStatusError
			usage.Error = err.Error()
			return usage, nil
		}

		for _, point := range resp.Points {
			if val := point.Value.GetInt64Value(); val != 0 {
				usage.RequestCount += val
			} else if val := point.Value.GetDoubleValue(); val != 0 {
				usage.RequestCount += int64(val)
			}
		}
	}

	return usage, nil
}

func cleanServiceName(name string) string {
	if idx := strings.Index(name, "services/"); idx != -1 {
		name = name[idx+len("services/"):]
	}
	return name
}
