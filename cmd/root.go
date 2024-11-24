package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile     string
	outputDir   string
	daysToAudit int
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "gcp-auditor",
	Short: "A tool for auditing GCP services usage",
	Long: `GCP Auditor is a comprehensive tool for analyzing Google Cloud Platform services.
It helps identify enabled services, their usage patterns, and potential cost optimizations.`,
}

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gcp-auditor.yaml)")
	rootCmd.PersistentFlags().StringVar(&outputDir, "output-dir", "reports", "output directory for reports")
	rootCmd.PersistentFlags().IntVar(&daysToAudit, "days", 30, "number of days to analyze usage")

	// Bind flags to viper
	viper.BindPFlag("output-dir", rootCmd.PersistentFlags().Lookup("output-dir"))
	viper.BindPFlag("days", rootCmd.PersistentFlags().Lookup("days"))
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".gcp-auditor" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".gcp-auditor")
	}

	// Read in environment variables with prefix GCP_AUDITOR
	viper.SetEnvPrefix("GCP_AUDITOR")
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
