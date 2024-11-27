package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/ybonda/gcp-auditor/internal/config"
	"github.com/ybonda/gcp-auditor/internal/domain"
	"github.com/ybonda/gcp-auditor/internal/report"
	"github.com/ybonda/gcp-auditor/internal/repository/gcp"
	"github.com/ybonda/gcp-auditor/internal/service"
	"github.com/ybonda/gcp-auditor/pkg/logging"
)

var logger *logging.Logger

// auditCmd represents the audit command
var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit GCP services and their usage",
	Long: `Performs a comprehensive audit of GCP services across all accessible projects.

Examples:
  # Run audit with default settings (generates both markdown and JSON)
  gcp-auditor audit

  # Run audit for the last 60 days
  gcp-auditor audit --days 60

  # Run audit with specific format
  gcp-auditor audit --format markdown
  gcp-auditor audit --format json

  # Run audit with verbose output
  gcp-auditor audit --verbose`,
	RunE: runAudit,
}

func init() {
	rootCmd.AddCommand(auditCmd)
	auditCmd.Flags().Bool("verbose", false, "Enable verbose output")
	auditCmd.Flags().String("format", "", "Report format (markdown, json, all)")
}

func runAudit(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	format, _ := cmd.Flags().GetString("format")

	// Create configuration with default values
	cfg := config.NewConfig(
		config.WithOutputDir(outputDir),
		config.WithDays(daysToAudit),
		config.WithVerbose(verbose),
		config.WithConcurrency(3),
	)

	// Only override format if explicitly specified
	if format != "" {
		// Validate format
		switch format {
		case "markdown", "json", "all":
			cfg = config.NewConfig(
				config.WithOutputDir(outputDir),
				config.WithDays(daysToAudit),
				config.WithVerbose(verbose),
				config.WithConcurrency(3),
				config.WithFormat(format),
			)
		default:
			return fmt.Errorf("invalid format %q. Must be one of: markdown, json, all", format)
		}
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	logger = logging.NewLogger(cfg.Verbose)
	logger.Debug("Configured audit period: %d days (%s)", daysToAudit, cfg.Period)
	logger.Debug("Using report format: %s", cfg.Format)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Initialize GCP client
	gcpClient, err := gcp.NewClient(ctx)
	if err != nil {
		logger.Error("Failed to initialize GCP client: %v", err)
		return err
	}
	defer gcpClient.Close()

	// Initialize repositories
	projectRepo := gcp.NewProjectRepository(gcpClient.ResourceManager, cfg.Verbose)
	serviceRepo := gcp.NewServiceRepository(gcpClient.ServiceUsage, gcpClient.Monitoring, cfg.Verbose)

	// Initialize reporters based on format
	var reporters []domain.Reporter
	switch cfg.Format {
	case "markdown":
		reporters = append(reporters, report.NewMarkdownReporter(outputDir))
	case "json":
		reporters = append(reporters, report.NewJSONReporter(outputDir))
	case "all":
		reporters = append(reporters, report.NewMarkdownReporter(outputDir))
		reporters = append(reporters, report.NewJSONReporter(outputDir))
	}

	// Create audit service with unified config
	auditService := service.NewAuditService(
		projectRepo,
		serviceRepo,
		reporters,
		cfg,
	)

	// Run audit
	auditReport, err := auditService.Audit(ctx)
	if err != nil {
		logger.Error("Audit failed: %v", err)
		return err
	}

	printAuditSummary(auditReport)
	return nil
}

func printAuditSummary(report domain.AuditReport) {
	logger.Info("\nAudit Summary")
	logger.Info("-------------")
	logger.Info("Projects analyzed: %d", report.Statistics.ValidProjects)
	logger.Info("Excluded projects: %d", report.Statistics.ExcludedProjects)
	logger.Info("Skipped projects: %d", report.Statistics.SkippedProjects)
	logger.Info("Unique services found: %d", report.Statistics.UniqueServices)
	logger.Info("Services with no usage: %d", report.Statistics.ServicesWithNoUsage)

	if len(report.SkippedProjects) > 0 {
		logger.Info("\nSkipped Projects:")
		for projectID, err := range report.SkippedProjects {
			logger.Info("- %s: %v", projectID, err)
		}
	}

	logger.Info("\nReport has been generated in: %s", outputDir)
}
