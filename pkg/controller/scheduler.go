package controller

import (
	"context"
	"time"

	"github.com/gammazero/workerpool"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// Scheduler handles health check task scheduling and worker pool management
type Scheduler struct {
	clientset  kubernetes.Interface
	podSet     *PodSet
	config     *HealthChecker
	workerPool *workerpool.WorkerPool
}

// NewScheduler creates a new health check scheduler
func NewScheduler(clientset kubernetes.Interface, podSet *PodSet) *Scheduler {
	return &Scheduler{
		clientset: clientset,
		podSet:    podSet,
		config:    NewHealthChecker(),
	}
}

// SetConfig sets the health check configuration
func (s *Scheduler) SetConfig(config *HealthChecker) {
	s.config = config
}

// StartHealthCheckWorkers starts health check workers using WorkerPool
func (s *Scheduler) StartHealthCheckWorkers(ctx context.Context) {
	interval := s.config.GetHealthCheckInterval()
	workerCount := s.config.GetWorkerCount()

	klog.Infof("Scheduler: starting health check workers with interval=%v, workerCount=%d", interval, workerCount)

	// Create worker pool using official gammazero/workerpool
	s.workerPool = workerpool.New(workerCount)
	klog.Infof("Scheduler: worker pool created successfully")

	s.runHealthCheckScheduler(ctx, interval)
}

// runHealthCheckScheduler runs health check scheduler
func (s *Scheduler) runHealthCheckScheduler(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			klog.Info("Health check scheduler stopped")
			if s.workerPool != nil {
				s.workerPool.StopWait()
			}
			return
		case <-ticker.C:
			s.dispatchHealthCheckTasks(ctx)
		}
	}
}

// dispatchHealthCheckTasks dispatches health check tasks to worker pool
func (s *Scheduler) dispatchHealthCheckTasks(ctx context.Context) {
	klog.V(4).Infof("Scheduler: starting health check task dispatch")

	// Get available pods for health check
	availablePods := s.podSet.GetAvailablePods()
	if len(availablePods) == 0 {
		klog.V(4).Infof("No available pods for health check")
		return
	}

	klog.V(4).Infof("Scheduler: found %d available pods for health check", len(availablePods))

	// Log statistics
	totalCount, namespaceCount := s.podSet.GetStats()
	klog.V(4).Infof("PodSet stats: total=%d, by namespace=%v", totalCount, namespaceCount)

	if s.workerPool != nil {
		waitingCount := s.workerPool.WaitingQueueSize()
		klog.V(4).Infof("WorkerPool stats: waiting queue size=%d", waitingCount)
	} else {
		klog.Warningf("Scheduler: workerPool is nil!")
	}

	// Convert pods to tasks and submit to worker pool
	for _, pod := range availablePods {
		// Mark pod as being checked
		s.podSet.SetBeingChecked(pod.GetIP(), true)

		// Create task function for this pod
		podCopy := pod // Capture pod in closure
		task := func() {
			// Create task-specific context with timeout
			taskCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			// Check if parent context is already canceled
			if ctx.Err() != nil {
				klog.V(4).Infof("Skipping health check for pod %s: scheduler stopped", podCopy.GetName())
				podCopy.SetIsBeingChecked(false)
				return
			}

			klog.V(4).Infof("Worker: starting health check for pod %s (IP: %s)", podCopy.GetName(), podCopy.GetIP())
			start := time.Now()

			err := s.config.CheckPod(taskCtx, s.clientset, podCopy)

			duration := time.Since(start)
			if err != nil {
				switch err {
				case context.Canceled:
					klog.Infof("Health check for pod %s canceled", podCopy.GetName())
				case context.DeadlineExceeded:
					klog.Warningf("Health check for pod %s timeout after %v", podCopy.GetName(), duration)
				default:
					klog.Warningf("Worker: health check failed for pod %s: %v", podCopy.GetName(), err)
				}
			} else {
				klog.V(3).Infof("Worker: completed health check for pod %s in %v", podCopy.GetName(), duration)
			}
		}

		// Submit task to worker pool
		s.workerPool.Submit(task)
		klog.V(4).Infof("Scheduler: submitted task for pod %s (IP: %s)", pod.GetName(), pod.GetIP())
	}

	klog.V(4).Infof("Scheduler: dispatched %d health check tasks to worker pool", len(availablePods))
}

// Stop stops the scheduler and worker pool
func (s *Scheduler) Stop() {
	if s.workerPool != nil {
		s.workerPool.StopWait()
	}
}

// GetStats returns scheduler statistics
func (s *Scheduler) GetStats() (int, int, int) {
	if s.workerPool == nil {
		return 0, 0, 0
	}

	totalCount, _ := s.podSet.GetStats()
	waitingCount := s.workerPool.WaitingQueueSize()

	return totalCount, 0, waitingCount // workerCount is not available in official workerpool
}
