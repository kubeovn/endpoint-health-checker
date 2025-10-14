package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"

	"endpoint_health_checker/pkg/config"
	"endpoint_health_checker/pkg/controller"
)

var (
	kubeconfig    string
	leaseLockNS   string
	leaseLockName string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file, if not running in cluster")
	flag.StringVar(&leaseLockNS, "lease-namespace", os.Getenv("POD_NAMESPACE"), "Namespace for leader election lease")
	flag.StringVar(&leaseLockName, "lease-name", "endpoint-health-checker-leader", "Name for leader election lease")
}

// InitLog initializes logging configuration
func InitLog() {
	// Configure klog to output to file
	logDir := "/var/log/endpoint_health_checker"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// If we can't create the directory, just continue with default logging
		klog.Warningf("Failed to create log directory %s: %v, using default logging", logDir, err)
		return
	}

	// Set klog flags to output to both file and stderr
	flag.Set("alsologtostderr", "true")
	flag.Set("log_dir", logDir)
	flag.Set("log_file", filepath.Join(logDir, "endpoint_health_checker.log"))

	klog.Infof("Logging configured to output to %s", filepath.Join(logDir, "endpoint_health_checker.log"))
}

func main() {
	// Initialize klog flags first so they are available for command line parsing
	klog.InitFlags(nil)

	flag.Parse()

	// Initialize logging
	InitLog()

	// Load configuration
	cfg, err := config.LoadFromEnv()
	if err != nil {
		klog.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		klog.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize k8s client
	var k8sConfig *rest.Config
	if kubeconfig == "" {
		k8sConfig, err = rest.InClusterConfig()
	} else {
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		klog.Fatalf("Failed to build kubeconfig: %v", err)
	}

	// Optimize client configuration to reduce Watch connection issues
	k8sConfig.QPS = 100                  // Reduce QPS to avoid overwhelming API server
	k8sConfig.Burst = 200                // Reduce burst to avoid rate limiting
	k8sConfig.Timeout = 60 * time.Second // Increase timeout for better stability
	k8sConfig.RateLimiter = nil          // Disable rate limiter for better performance

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		klog.Fatalf("Failed to create clientset: %v", err)
	}

	// Create/ensure Lease object exists
	leaseLock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      cfg.GetLeaseLockName(),
			Namespace: cfg.GetLeaseLockNamespace(),
		},
		Client: clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: cfg.GetPodName(),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	podSet := controller.NewPodSet()

	// Create health check configuration and scheduler directly in main
	healthConfig := controller.NewHealthChecker()
	healthConfig.SetHealthCheckInterval(cfg.GetHealthCheckInterval())
	healthConfig.SetHealthCheckTimeout(cfg.GetHealthCheckTimeout())
	healthConfig.SetWorkerCount(cfg.GetHealthCheckConcurrency())
	healthConfig.SetRetryCount(cfg.GetHealthCheckRetryCount())

	// Create scheduler with configuration
	scheduler := controller.NewScheduler(clientset, podSet)
	scheduler.SetConfig(healthConfig)

	ctrl := controller.NewController(clientset, 0, podSet)

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            leaseLock,
		ReleaseOnCancel: true,
		LeaseDuration:   cfg.GetLeaseDuration(),
		RenewDeadline:   cfg.GetRenewDeadline(),
		RetryPeriod:     cfg.GetRetryPeriod(),
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				klog.Infof("%s: I am the leader, start health check loop", cfg.GetPodName())
				stopCh := make(chan struct{})
				go ctrl.Run(stopCh)
				go scheduler.StartHealthCheckWorkers(ctx)
				<-ctx.Done()
				close(stopCh)
			},
			OnStoppedLeading: func() {
				klog.Warningf("%s: lost leadership, now standby", cfg.GetPodName())
			},
			OnNewLeader: func(identity string) {
				if identity == cfg.GetPodName() {
					klog.Infof("%s: I am the new leader", cfg.GetPodName())
				} else {
					klog.Infof("%s: new leader is %s", cfg.GetPodName(), identity)
				}
			},
		},
	})
}
