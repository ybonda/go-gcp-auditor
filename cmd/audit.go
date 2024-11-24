package cmd

import (
	"context"
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
  # Run audit with default settings
  gcp-auditor audit

  # Run audit for the last 60 days
  gcp-auditor audit --days 60

  # Run audit with verbose output
  gcp-auditor audit --verbose`,
	RunE: runAudit,
}

func init() {
	rootCmd.AddCommand(auditCmd)
	auditCmd.Flags().Bool("verbose", false, "Enable verbose output")
}

func runAudit(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Create configuration
	cfg := config.NewConfig(
		config.WithOutputDir(outputDir),
		config.WithDays(daysToAudit),
		config.WithVerbose(verbose),
		config.WithConcurrency(3),
	)

	logger = logging.NewLogger(cfg.Verbose)
	logger.Debug("Configured audit period: %d days (%s)", daysToAudit, cfg.Period)

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

	// Initialize markdown reporter
	reporter := report.NewMarkdownReporter(cfg.OutputDir)

	// Create audit service with unified config
	auditService := service.NewAuditService(
		projectRepo,
		serviceRepo,
		[]domain.Reporter{reporter},
		cfg,
	)
	// Run audit
	auditReport, err := auditService.Audit(ctx)
	if err != nil {
		logger.Error("Audit failed: %v", err)
		return err
	}

	// Generate report
	if err := reporter.GenerateReport(auditReport); err != nil {
		logger.Error("Failed to generate report: %v", err)
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
