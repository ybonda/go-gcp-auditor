// pkg/gcp/services.go
package gcp

import "strings"

// Common GCP service categories
const (
	CategoryCompute    = "compute"
	CategoryStorage    = "storage"
	CategoryDatabase   = "database"
	CategoryNetworking = "networking"
	CategorySecurity   = "security"
	CategoryMonitoring = "monitoring"
	CategoryDeveloper  = "developer"
)

// ServiceCategory maps a service name to its category
func ServiceCategory(serviceName string) string {
	categories := map[string]string{
		"compute":           CategoryCompute,
		"containerregistry": CategoryCompute,
		"container":         CategoryCompute,
		"run":               CategoryCompute,

		"storage":  CategoryStorage,
		"bigquery": CategoryStorage,
		"bigtable": CategoryStorage,

		"sql":     CategoryDatabase,
		"redis":   CategoryDatabase,
		"spanner": CategoryDatabase,

		"dns":           CategoryNetworking,
		"vpc":           CategoryNetworking,
		"loadbalancing": CategoryNetworking,

		"cloudkms":      CategorySecurity,
		"iap":           CategorySecurity,
		"secretmanager": CategorySecurity,

		"monitoring": CategoryMonitoring,
		"logging":    CategoryMonitoring,
		"cloudtrace": CategoryMonitoring,

		"cloudbuild":       CategoryDeveloper,
		"sourcerepo":       CategoryDeveloper,
		"artifactregistry": CategoryDeveloper,
	}

	serviceName = strings.ToLower(serviceName)
	for key, category := range categories {
		if strings.Contains(serviceName, key) {
			return category
		}
	}

	return "other"
}

// IsInfrastructureService returns true if the service is infrastructure-related
func IsInfrastructureService(serviceName string) bool {
	infraServices := map[string]bool{
		"compute":       true,
		"container":     true,
		"storage":       true,
		"sql":           true,
		"networking":    true,
		"dns":           true,
		"loadbalancing": true,
	}

	serviceName = strings.ToLower(serviceName)
	for service := range infraServices {
		if strings.Contains(serviceName, service) {
			return true
		}
	}

	return false
}
