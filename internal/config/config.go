package config

import "time"

type Config struct {
	OutputDir   string
	DaysToAudit int
	Format      string
	Verbose     bool
	Period      time.Duration
	Concurrency int
}

type Option func(*Config)

func WithOutputDir(dir string) Option {
	return func(c *Config) {
		c.OutputDir = dir
	}
}

func WithDays(days int) Option {
	return func(c *Config) {
		c.DaysToAudit = days
		c.Period = time.Duration(days) * 24 * time.Hour
	}
}

func WithFormat(format string) Option {
	return func(c *Config) {
		c.Format = format
	}
}

func WithVerbose(verbose bool) Option {
	return func(c *Config) {
		c.Verbose = verbose
	}
}

func WithConcurrency(n int) Option {
	return func(c *Config) {
		if n > 0 {
			c.Concurrency = n
		}
	}
}

func NewConfig(opts ...Option) *Config {
	// Default configuration
	c := &Config{
		OutputDir:   "reports",
		DaysToAudit: 30,
		Format:      "all",
		Verbose:     false,
		Period:      30 * 24 * time.Hour,
		Concurrency: 3,
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	return c
}
