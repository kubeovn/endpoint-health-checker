package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	goping "github.com/prometheus-community/pro-bing"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type HealthCheckPodInfo interface {
	GetNamespace() string
	GetName() string
	GetIP() string
	GetPorts() []int32
	SetIsBeingChecked(checked bool)
	GetLastHealthStatus() *bool
	SetLastHealthStatus(status bool)
}

// HealthCheckConfig health check configuration
type HealthCheckConfig struct {
	RetryCount   int           // Retry count
	ProbeTimeout time.Duration // Single probe timeout
}

// HealthChecker handles health check configuration and execution
type HealthChecker struct {
	healthCheckInterval time.Duration
	healthCheckTimeout  time.Duration
	workerCount         int
	retryCount          int
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		healthCheckInterval: 1 * time.Second,
		healthCheckTimeout:  1 * time.Second,
		workerCount:         10,
		retryCount:          3,
	}
}

// SetHealthCheckInterval sets health check interval
func (hc *HealthChecker) SetHealthCheckInterval(interval time.Duration) {
	hc.healthCheckInterval = interval
}

// SetHealthCheckTimeout sets health check timeout
func (hc *HealthChecker) SetHealthCheckTimeout(timeout time.Duration) {
	hc.healthCheckTimeout = timeout
}

// SetWorkerCount sets health check worker count
func (hc *HealthChecker) SetWorkerCount(count int) {
	if count > 0 {
		hc.workerCount = count
	}
}

// SetRetryCount sets health check retry count
func (hc *HealthChecker) SetRetryCount(count int) {
	if count > 0 {
		hc.retryCount = count
	}
}

// GetHealthCheckInterval gets health check interval
func (hc *HealthChecker) GetHealthCheckInterval() time.Duration {
	return hc.healthCheckInterval
}

// GetHealthCheckTimeout gets health check timeout
func (hc *HealthChecker) GetHealthCheckTimeout() time.Duration {
	return hc.healthCheckTimeout
}

// GetWorkerCount gets health check worker count
func (hc *HealthChecker) GetWorkerCount() int {
	return hc.workerCount
}

// GetRetryCount gets health check retry count
func (hc *HealthChecker) GetRetryCount() int {
	return hc.retryCount
}

// CheckPod performs health check on a pod
func (hc *HealthChecker) CheckPod(clientset kubernetes.Interface, pod HealthCheckPodInfo) error {
	// Perform health check
	healthy := hc.performHealthCheck(pod)

	// Update pod status if changed
	if err := hc.updatePodStatusIfChanged(clientset, pod, healthy); err != nil {
		return err
	}

	// Update cached health status
	pod.SetLastHealthStatus(healthy)

	// Health check completed, reset IsBeingChecked flag
	pod.SetIsBeingChecked(false)

	return nil
}

// performHealthCheck performs the actual health check on a pod
func (hc *HealthChecker) performHealthCheck(pod HealthCheckPodInfo) bool {
	config := &HealthCheckConfig{
		RetryCount:   hc.retryCount,
		ProbeTimeout: hc.healthCheckTimeout,
	}

	if len(pod.GetPorts()) > 0 {
		return hc.checkPorts(pod, config)
	} else {
		return hc.checkICMP(pod, config)
	}
}

// checkPorts performs TCP health check on all ports
func (hc *HealthChecker) checkPorts(pod HealthCheckPodInfo, config *HealthCheckConfig) bool {
	healthy := true
	for _, port := range pod.GetPorts() {
		addr := net.JoinHostPort(pod.GetIP(), fmt.Sprintf("%d", port))
		if err := tcpProbeWithRetry(addr, config); err != nil {
			healthy = false
			// Extract actual retry count from error message
			klog.Errorf("Pod %s/%s probe port %d failed: %v",
				pod.GetNamespace(), pod.GetName(), port, err)
		} else {
			klog.V(4).Infof("Pod %s/%s probe port %d success", pod.GetNamespace(), pod.GetName(), port)
		}
	}
	return healthy
}

// checkICMP performs ICMP health check
func (hc *HealthChecker) checkICMP(pod HealthCheckPodInfo, config *HealthCheckConfig) bool {
	if err := icmpProbeWithRetry(pod.GetIP(), config); err != nil {
		// Extract actual retry count from error message
		klog.Errorf("Pod %s/%s ICMP probe failed: %v",
			pod.GetNamespace(), pod.GetName(), err)
		return false
	} else {
		klog.V(4).Infof("Pod %s/%s ICMP probe success", pod.GetNamespace(), pod.GetName())
		return true
	}
}

// updatePodStatusIfChanged updates pod ready status only if health status changed
func (hc *HealthChecker) updatePodStatusIfChanged(clientset kubernetes.Interface, pod HealthCheckPodInfo, healthy bool) error {
	// Check if health status has changed
	lastStatus := pod.GetLastHealthStatus()
	statusChanged := lastStatus == nil || *lastStatus != healthy

	if !statusChanged {
		klog.V(4).Infof("Pod %s/%s: Health status unchanged (%v), skipping API call",
			pod.GetNamespace(), pod.GetName(), healthy)
		return nil
	}

	// Get pod from Kubernetes API
	k8sPod, err := clientset.CoreV1().Pods(pod.GetNamespace()).Get(context.Background(), pod.GetName(), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("Pod %s/%s not found in Kubernetes, should be removed from PodSet",
				pod.GetNamespace(), pod.GetName())
			return err // Return original NotFound error directly
		}
		return fmt.Errorf("failed to get pod %s/%s: %w", pod.GetNamespace(), pod.GetName(), err)
	}

	if err := updatePodReadyWithPod(clientset, k8sPod, healthy); err != nil {
		klog.Errorf("update pod %s/%s ready failed: %v", pod.GetNamespace(), pod.GetName(), err)
		return err
	}

	return nil
}

// tcpProbeWithRetry TCP probe with retry mechanism
func tcpProbeWithRetry(addr string, config *HealthCheckConfig) error {
	var lastErr error

	for i := 0; i <= config.RetryCount; i++ {
		start := time.Now()
		if err := tcpProbe(addr, config.ProbeTimeout); err != nil {
			lastErr = err
			if i < config.RetryCount {
				elapsed := time.Since(start)
				remaining := config.ProbeTimeout - elapsed
				if remaining > 0 {
					klog.V(4).Infof("TCP probe attempt %d/%d failed for %s: %v, waiting %v before retry...",
						i+1, config.RetryCount+1, addr, err, remaining)
					time.Sleep(remaining)
				} else {
					klog.V(4).Infof("TCP probe attempt %d/%d failed for %s: %v, retrying immediately...",
						i+1, config.RetryCount+1, addr, err)
				}
				continue
			}
		} else {
			// Return immediately on success, no more retries
			if i > 0 {
				klog.V(4).Infof("TCP probe succeeded on attempt %d/%d for %s",
					i+1, config.RetryCount+1, addr)
			}
			return nil
		}
	}

	return fmt.Errorf("TCP probe failed after %d attempts: %w", config.RetryCount+1, lastErr)
}

// icmpProbeWithRetry ICMP probe with retry mechanism
func icmpProbeWithRetry(ip string, config *HealthCheckConfig) error {
	var lastErr error

	for i := 0; i <= config.RetryCount; i++ {
		if err := icmpProbe(ip, 1, config.ProbeTimeout); err != nil {
			lastErr = err
			if i < config.RetryCount {
				klog.V(4).Infof("ICMP probe attempt %d/%d failed for %s: %v, retrying...",
					i+1, config.RetryCount+1, ip, err)
				continue
			}
		} else {
			// Return immediately on success, no more retries
			if i > 0 {
				klog.V(4).Infof("ICMP probe succeeded on attempt %d/%d for %s",
					i+1, config.RetryCount+1, ip)
			}
			return nil
		}
	}

	return fmt.Errorf("ICMP probe failed after %d attempts: %w", config.RetryCount+1, lastErr)
}

func tcpProbe(addr string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

func icmpProbe(ip string, count int, timeout time.Duration) error {
	pinger, err := goping.NewPinger(ip)
	if err != nil {
		return err
	}
	pinger.Count = count
	pinger.Timeout = timeout
	pinger.SetPrivileged(true)

	err = pinger.Run()
	if err != nil {
		return err
	}

	// Check ping results, ensure there are successful responses
	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		return fmt.Errorf("ICMP probe failed: no response from %s", ip)
	}

	return nil
}
func updatePodReadyWithPod(clientset kubernetes.Interface, pod *corev1.Pod, success bool) error {
	klog.V(4).Infof("Updating pod status: namespace=%s, name=%s, success=%v", pod.Namespace, pod.Name, success)

	hasReadinessGate := hasReadinessGate(pod)

	if hasReadinessGate {
		status := corev1.ConditionTrue
		if !success {
			status = corev1.ConditionFalse
		}
		klog.Infof("Pod %s/%s: Setting readinessGate condition to %v", pod.Namespace, pod.Name, status)
		updateReadinessGateCondition(&pod.Status.Conditions, status)
	}

	if !success {
		klog.Infof("Pod %s/%s: Setting Ready condition to False due to health check failure", pod.Namespace, pod.Name)
		updateReadyCondition(&pod.Status.Conditions, corev1.ConditionFalse)
	} else if !hasReadinessGate {
		// If health check passed and no readinessGate, no need to update anything
		return nil
	}

	// Apply the patch
	patch := map[string]interface{}{
		"status": map[string]interface{}{
			"conditions": pod.Status.Conditions,
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}

	_, err = clientset.CoreV1().Pods(pod.Namespace).Patch(
		context.Background(),
		pod.Name,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
		"status",
	)

	if err != nil {
		return fmt.Errorf("failed to patch pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}

	klog.Infof("Pod %s/%s: Successfully updated pod conditions", pod.Namespace, pod.Name)
	return nil
}

// updateReadyCondition updates the Ready condition status
func updateReadyCondition(conditions *[]corev1.PodCondition, status corev1.ConditionStatus) {
	now := metav1.Now()

	// Update existing Ready condition
	for i, cond := range *conditions {
		if cond.Type == corev1.PodReady {
			(*conditions)[i].Status = status
			(*conditions)[i].LastProbeTime = now
			(*conditions)[i].LastTransitionTime = now
			return
		}
	}

	// Append new Ready condition if not found
	*conditions = append(*conditions, corev1.PodCondition{
		Type:               corev1.PodReady,
		Status:             status,
		LastProbeTime:      now,
		LastTransitionTime: now,
	})
}

// hasReadinessGate checks if pod has readinessGate configured
func hasReadinessGate(pod *corev1.Pod) bool {
	const readinessGateType = "endpointHealthCheckSuccess"

	for _, gate := range pod.Spec.ReadinessGates {
		if string(gate.ConditionType) == readinessGateType {
			return true
		}
	}
	return false
}

// updateReadinessGateCondition updates the readinessGate condition status
func updateReadinessGateCondition(conditions *[]corev1.PodCondition, status corev1.ConditionStatus) {
	const readinessGateType = "endpointHealthCheckSuccess"
	now := metav1.Now()

	// Update existing readinessGate condition
	for i, cond := range *conditions {
		if cond.Type == corev1.PodConditionType(readinessGateType) {
			(*conditions)[i].Status = status
			(*conditions)[i].LastProbeTime = now
			(*conditions)[i].LastTransitionTime = now
			return
		}
	}

	// Append new readinessGate condition if not found
	*conditions = append(*conditions, corev1.PodCondition{
		Type:               corev1.PodConditionType(readinessGateType),
		Status:             status,
		LastProbeTime:      now,
		LastTransitionTime: now,
	})
}
