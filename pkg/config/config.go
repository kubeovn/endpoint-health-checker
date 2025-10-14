package config

import (
	"fmt"
	"os"
	"time"

	"k8s.io/klog/v2"
)

// Config application configuration
type Config struct {
	HealthCheckInterval    time.Duration
	HealthCheckTimeout     time.Duration
	HealthCheckConcurrency int
	HealthCheckRetryCount  int
	PodName                string
	PodNamespace           string
	LeaseLockName          string
	LeaseLockNamespace     string
	LeaseDuration          time.Duration
	RenewDeadline          time.Duration
	RetryPeriod            time.Duration
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	config := &Config{}

	// Set default values
	config.HealthCheckInterval = 1 * time.Second
	config.HealthCheckTimeout = 1 * time.Second
	config.HealthCheckConcurrency = 10
	config.HealthCheckRetryCount = 3
	config.LeaseLockName = "endpoint-health-checker-leader"
	config.LeaseDuration = 4 * time.Second
	config.RenewDeadline = 2 * time.Second
	config.RetryPeriod = 500 * time.Millisecond

	// Parse health check interval
	if intervalStr := os.Getenv("HEALTH_CHECK_INTERVAL"); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err != nil {
			return nil, fmt.Errorf("invalid HEALTH_CHECK_INTERVAL: %v", err)
		} else {
			config.HealthCheckInterval = interval
		}
	}

	// Parse health check timeout
	if timeoutStr := os.Getenv("HEALTH_CHECK_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err != nil {
			return nil, fmt.Errorf("invalid HEALTH_CHECK_TIMEOUT: %v", err)
		} else {
			config.HealthCheckTimeout = timeout
		}
	}

	// Parse health check concurrency
	if concurrencyStr := os.Getenv("HEALTH_CHECK_CONCURRENCY"); concurrencyStr != "" {
		var concurrency int
		if count, err := fmt.Sscanf(concurrencyStr, "%d", &concurrency); err != nil || count != 1 {
			klog.Warningf("Invalid HEALTH_CHECK_CONCURRENCY: %s, using default: %d", concurrencyStr, config.HealthCheckConcurrency)
		} else if concurrency > 0 {
			config.HealthCheckConcurrency = concurrency
		}
	}

	// Parse health check retry count
	if retryCountStr := os.Getenv("HEALTH_CHECK_RETRY_COUNT"); retryCountStr != "" {
		var retryCount int
		if count, err := fmt.Sscanf(retryCountStr, "%d", &retryCount); err != nil || count != 1 {
			klog.Warningf("Invalid HEALTH_CHECK_RETRY_COUNT: %s, using default: %d", retryCountStr, config.HealthCheckRetryCount)
		} else if retryCount >= 0 {
			config.HealthCheckRetryCount = retryCount
		}
	}

	// Parse Pod information
	config.PodName = os.Getenv("POD_NAME")
	if config.PodName == "" {
		if hostname, err := os.Hostname(); err == nil {
			config.PodName = hostname
		}
	}

	config.PodNamespace = os.Getenv("POD_NAMESPACE")
	if config.PodNamespace == "" {
		config.PodNamespace = "kube-system"
	}

	// Parse Lease configuration
	config.LeaseLockNamespace = os.Getenv("POD_NAMESPACE")
	if config.LeaseLockNamespace == "" {
		config.LeaseLockNamespace = "kube-system"
	}

	// Parse leader election configuration
	if leaseName := os.Getenv("LEASE_NAME"); leaseName != "" {
		config.LeaseLockName = leaseName
	}

	if leaseDurationStr := os.Getenv("LEASE_DURATION"); leaseDurationStr != "" {
		if leaseDuration, err := time.ParseDuration(leaseDurationStr); err != nil {
			klog.Warningf("Invalid LEASE_DURATION: %s, using default: %v", leaseDurationStr, config.LeaseDuration)
		} else {
			config.LeaseDuration = leaseDuration
		}
	}

	if renewDeadlineStr := os.Getenv("RENEW_DEADLINE"); renewDeadlineStr != "" {
		if renewDeadline, err := time.ParseDuration(renewDeadlineStr); err != nil {
			klog.Warningf("Invalid RENEW_DEADLINE: %s, using default: %v", renewDeadlineStr, config.RenewDeadline)
		} else {
			config.RenewDeadline = renewDeadline
		}
	}

	if retryPeriodStr := os.Getenv("RETRY_PERIOD"); retryPeriodStr != "" {
		if retryPeriod, err := time.ParseDuration(retryPeriodStr); err != nil {
			klog.Warningf("Invalid RETRY_PERIOD: %s, using default: %v", retryPeriodStr, config.RetryPeriod)
		} else {
			config.RetryPeriod = retryPeriod
		}
	}

	klog.Infof("Loaded configuration: interval=%v, timeout=%v, concurrency=%d, retryCount=%d, pod=%s/%s, lease=%s/%s, leaseDuration=%v, renewDeadline=%v, retryPeriod=%v",
		config.HealthCheckInterval, config.HealthCheckTimeout, config.HealthCheckConcurrency, config.HealthCheckRetryCount,
		config.PodNamespace, config.PodName, config.LeaseLockNamespace, config.LeaseLockName, config.LeaseDuration, config.RenewDeadline, config.RetryPeriod)

	return config, nil
}

// Validate validates configuration
func (c *Config) Validate() error {
	if c.HealthCheckInterval <= 0 {
		return fmt.Errorf("health check interval must be positive")
	}
	if c.HealthCheckTimeout <= 0 {
		return fmt.Errorf("health check timeout must be positive")
	}
	if c.HealthCheckConcurrency <= 0 {
		return fmt.Errorf("health check concurrency must be positive")
	}
	if c.HealthCheckRetryCount < 0 {
		return fmt.Errorf("health check retry count must be non-negative")
	}
	if c.PodName == "" {
		return fmt.Errorf("pod name cannot be empty")
	}
	if c.PodNamespace == "" {
		return fmt.Errorf("pod namespace cannot be empty")
	}
	if c.LeaseDuration <= 0 {
		return fmt.Errorf("lease duration must be positive")
	}
	if c.RenewDeadline <= 0 {
		return fmt.Errorf("renew deadline must be positive")
	}
	if c.RetryPeriod <= 0 {
		return fmt.Errorf("retry period must be positive")
	}
	if c.RenewDeadline >= c.LeaseDuration {
		return fmt.Errorf("renew deadline must be less than lease duration")
	}
	return nil
}

// GetHealthCheckInterval gets health check interval
func (c *Config) GetHealthCheckInterval() time.Duration {
	return c.HealthCheckInterval
}

// GetHealthCheckTimeout gets health check timeout
func (c *Config) GetHealthCheckTimeout() time.Duration {
	return c.HealthCheckTimeout
}

// GetHealthCheckConcurrency gets health check concurrency
func (c *Config) GetHealthCheckConcurrency() int {
	return c.HealthCheckConcurrency
}

// GetHealthCheckRetryCount gets health check retry count
func (c *Config) GetHealthCheckRetryCount() int {
	return c.HealthCheckRetryCount
}

// GetPodName gets Pod name
func (c *Config) GetPodName() string {
	return c.PodName
}

// GetPodNamespace gets Pod namespace
func (c *Config) GetPodNamespace() string {
	return c.PodNamespace
}

// GetLeaseLockName gets Lease lock name
func (c *Config) GetLeaseLockName() string {
	return c.LeaseLockName
}

// GetLeaseLockNamespace gets Lease lock namespace
func (c *Config) GetLeaseLockNamespace() string {
	return c.LeaseLockNamespace
}

// GetLeaseDuration gets lease duration
func (c *Config) GetLeaseDuration() time.Duration {
	return c.LeaseDuration
}

// GetRenewDeadline gets renew deadline
func (c *Config) GetRenewDeadline() time.Duration {
	return c.RenewDeadline
}

// GetRetryPeriod gets retry period
func (c *Config) GetRetryPeriod() time.Duration {
	return c.RetryPeriod
}
