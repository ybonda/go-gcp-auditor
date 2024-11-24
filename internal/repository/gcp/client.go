// internal/repository/gcp/client.go
package gcp

import (
	"context"
	"fmt"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	resourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	serviceusage "google.golang.org/api/serviceusage/v1"
)

type Client struct {
	ResourceManager *resourcemanager.Service
	ServiceUsage    *serviceusage.Service
	Monitoring      *monitoring.MetricClient
}

func NewClient(ctx context.Context) (*Client, error) {
	// Initialize Resource Manager client
	resourceManagerService, err := resourcemanager.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource manager client: %w", err)
	}

	// Initialize Service Usage client
	serviceUsageService, err := serviceusage.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create service usage client: %w", err)
	}

	// Initialize Monitoring client
	monitoringClient, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring client: %w", err)
	}

	return &Client{
		ResourceManager: resourceManagerService,
		ServiceUsage:    serviceUsageService,
		Monitoring:      monitoringClient,
	}, nil
}

func (c *Client) Close() error {
	if err := c.Monitoring.Close(); err != nil {
		return fmt.Errorf("failed to close monitoring client: %w", err)
	}
	return nil
}
