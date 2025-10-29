package controller

import (
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type PodInfo struct {
	Namespace        string
	Name             string
	IP               string
	Ports            []int32
	IsBeingChecked   bool  // Mark whether it's being health checked
	LastHealthStatus *bool // Record last health check status, nil means unknown
}

type PodSet struct {
	mu   sync.RWMutex
	pods map[string]*PodInfo // key: podIP
}

func NewPodSet() *PodSet {
	return &PodSet{pods: make(map[string]*PodInfo)}
}

func (ps *PodSet) AddOrUpdate(pod *corev1.Pod) {
	if pod.Status.Phase != corev1.PodRunning || pod.Status.PodIP == "" {
		klog.V(4).Infof("Skipping pod %s/%s: Phase=%s, PodIP=%s",
			pod.Namespace, pod.Name, pod.Status.Phase, pod.Status.PodIP)
		return
	}

	if !shouldCheckPod(pod) {
		klog.V(4).Infof("Skipping pod %s/%s: health check not enabled via annotation",
			pod.Namespace, pod.Name)
		return
	}

	if !isPodReady(pod) {
		klog.V(4).Infof("Skipping pod %s/%s: waiting for initial readiness probe to pass",
			pod.Namespace, pod.Name)
		return
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.pods[pod.Status.PodIP] = &PodInfo{
		Namespace: pod.Namespace,
		Name:      pod.Name,
		IP:        pod.Status.PodIP,
		Ports:     getProbePorts(pod),
	}

	klog.Infof("Added pod %s/%s (IP: %s) to PodSet, total: %d",
		pod.Namespace, pod.Name, pod.Status.PodIP, len(ps.pods))
}

func (ps *PodSet) Delete(pod *corev1.Pod) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Check if PodIP is empty
	if pod.Status.PodIP == "" {
		klog.V(4).Infof("Pod %s/%s has empty PodIP, cannot delete from PodSet", pod.Namespace, pod.Name)
		return
	}

	// Check if Pod exists in PodSet
	if _, exists := ps.pods[pod.Status.PodIP]; !exists {
		klog.V(4).Infof("Pod %s/%s with IP %s not found in PodSet", pod.Namespace, pod.Name, pod.Status.PodIP)
		return
	}

	delete(ps.pods, pod.Status.PodIP)
	klog.Infof("Deleted pod %s/%s with IP %s from PodSet", pod.Namespace, pod.Name, pod.Status.PodIP)
}

// DeleteByNamespaceAndName deletes Pod by namespace and name, used when PodIP is empty
func (ps *PodSet) DeleteByNamespaceAndName(namespace, name string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Iterate through all pods to find matching pod
	for ip, podInfo := range ps.pods {
		if podInfo.Namespace == namespace && podInfo.Name == name {
			klog.Infof("Deleted pod %s/%s with IP %s from PodSet", namespace, name, ip)
			delete(ps.pods, ip)
			return
		}
	}

	klog.V(4).Infof("Pod %s/%s not found in PodSet", namespace, name)
}

// GetStats gets PodSet statistics
func (ps *PodSet) GetStats() (int, map[string]int) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	namespaceCount := make(map[string]int)
	for _, pod := range ps.pods {
		namespaceCount[pod.Namespace]++
	}

	return len(ps.pods), namespaceCount
}

// SetBeingChecked sets Pod's being checked status
func (ps *PodSet) SetBeingChecked(podIP string, isBeingChecked bool) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if pod, exists := ps.pods[podIP]; exists {
		pod.IsBeingChecked = isBeingChecked
		klog.V(4).Infof("Set pod %s/%s (IP: %s) IsBeingChecked to %v",
			pod.Namespace, pod.Name, podIP, isBeingChecked)
		return true
	}
	klog.Warningf("Pod with IP %s not found when setting IsBeingChecked", podIP)
	return false
}

// GetAvailablePods gets all unchecked Pod list
func (ps *PodSet) GetAvailablePods() []*PodInfo {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var result []*PodInfo
	for _, pod := range ps.pods {
		if !pod.IsBeingChecked {
			result = append(result, pod)
		}
	}
	klog.V(4).Infof("GetAvailablePods() returning %d available pods", len(result))
	return result
}

func getProbePorts(pod *corev1.Pod) []int32 {
	ports := make(map[int32]struct{})
	for _, c := range pod.Spec.Containers {
		for _, probe := range []*corev1.Probe{c.LivenessProbe, c.ReadinessProbe} {
			if probe == nil {
				continue
			}
			if probe.TCPSocket != nil {
				ports[probe.TCPSocket.Port.IntVal] = struct{}{}
			}
			if probe.HTTPGet != nil {
				ports[probe.HTTPGet.Port.IntVal] = struct{}{}
			}
			if probe.GRPC != nil {
				ports[probe.GRPC.Port] = struct{}{}
			}
		}
	}
	var result []int32
	for p := range ports {
		result = append(result, p)
	}
	return result
}

func shouldCheckPod(pod *corev1.Pod) bool {
	const annotationKey = "endpoint-health-checker.io/enabled"
	const readinessGateType = "endpointHealthCheckSuccess"

	if pod.Annotations != nil {
		if value, exists := pod.Annotations[annotationKey]; exists {
			return value == "true"
		}
	}

	// legacy way for backward compatibility
	for _, gate := range pod.Spec.ReadinessGates {
		if string(gate.ConditionType) == readinessGateType {
			return true
		}
	}

	return false
}

// isPodReady checks if Pod has passed kubelet's readiness probe
func isPodReady(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

func (p *PodInfo) GetNamespace() string            { return p.Namespace }
func (p *PodInfo) GetName() string                 { return p.Name }
func (p *PodInfo) GetIP() string                   { return p.IP }
func (p *PodInfo) GetPorts() []int32               { return p.Ports }
func (p *PodInfo) SetIsBeingChecked(checked bool)  { p.IsBeingChecked = checked }
func (p *PodInfo) GetLastHealthStatus() *bool      { return p.LastHealthStatus }
func (p *PodInfo) SetLastHealthStatus(status bool) { p.LastHealthStatus = &status }
