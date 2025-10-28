package controller

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/stretchr/testify/assert"
)

func TestNewController(t *testing.T) {
	// Create a fake clientset
	clientset := fake.NewSimpleClientset()
	podSet := NewPodSet()

	controller := NewController(clientset, time.Minute, podSet)

	assert.NotNil(t, controller)
	assert.NotNil(t, controller.clientset)
	assert.NotNil(t, controller.podSet)
}

func TestPodSetOperations(t *testing.T) {
	podSet := NewPodSet()

	// Test adding a pod
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-pod",
			Namespace:   "default",
			Annotations: map[string]string{"endpoint-health-checker.io/enabled": "true"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "192.168.1.100",
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	podSet.AddOrUpdate(testPod)
	count, _ := podSet.GetStats()
	assert.Equal(t, 1, count)

	// Test updating a pod
	testPod.Status.PodIP = "192.168.1.101"
	podSet.AddOrUpdate(testPod)
	count, _ = podSet.GetStats()
	assert.Equal(t, 2, count)

	// Test deleting a pod
	podSet.Delete(testPod)
	count, _ = podSet.GetStats()
	assert.Equal(t, 1, count)

	// Test deleting by namespace and name
	podSet.DeleteByNamespaceAndName("default", "test-pod")
	count, _ = podSet.GetStats()
	assert.Equal(t, 0, count)
}

func TestControllerPodEvents(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	podSet := NewPodSet()
	controller := NewController(clientset, time.Minute, podSet)

	// Test pod add event
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-pod",
			Namespace:   "default",
			Annotations: map[string]string{"endpoint-health-checker.io/enabled": "true"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "192.168.1.100",
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	controller.onPodAdd(testPod)
	count, _ := podSet.GetStats()
	assert.Equal(t, 1, count)

	// Test pod update event
	updatedPod := testPod.DeepCopy()
	updatedPod.Labels = map[string]string{"updated": "true"}
	controller.onPodUpdate(nil, updatedPod)
	count, _ = podSet.GetStats()
	assert.Equal(t, 1, count)

	// Test pod delete event
	controller.onPodDelete(testPod)
	count, _ = podSet.GetStats()
	assert.Equal(t, 0, count)
}

func TestControllerWithDeletedFinalStateUnknown(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	podSet := NewPodSet()
	controller := NewController(clientset, time.Minute, podSet)

	// Add a pod first
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-pod",
			Namespace:   "default",
			Annotations: map[string]string{"endpoint-health-checker.io/enabled": "true"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "192.168.1.100",
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	controller.onPodAdd(testPod)
	count, _ := podSet.GetStats()
	assert.Equal(t, 1, count)

	// Test delete event with DeletedFinalStateUnknown
	deletedState := cache.DeletedFinalStateUnknown{
		Key: "default/test-pod",
		Obj: testPod,
	}

	controller.onPodDelete(deletedState)
	count, _ = podSet.GetStats()
	assert.Equal(t, 0, count)
}

func TestControllerPodWithEmptyIP(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	podSet := NewPodSet()
	controller := NewController(clientset, time.Minute, podSet)

	// Test pod with empty IP
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-pod-empty-ip",
			Namespace:   "default",
			Annotations: map[string]string{"endpoint-health-checker.io/enabled": "true"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "",
		},
	}

	controller.onPodAdd(testPod)
	count, _ := podSet.GetStats()
	assert.Equal(t, 0, count) // Pod with empty IP should not be added

	// Delete pod with empty IP - should use namespace/name
	controller.onPodDelete(testPod)
	count, _ = podSet.GetStats()
	assert.Equal(t, 0, count)
}

func TestContextCancellation(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	podSet := NewPodSet()

	// Create a test pod
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-pod",
			Namespace:   "default",
			Annotations: map[string]string{"endpoint-health-checker.io/enabled": "true"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.FromInt(8080),
							},
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "192.168.1.100",
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	// Add pod to podSet
	podSet.AddOrUpdate(testPod)

	// Create scheduler
	scheduler := NewScheduler(clientset, podSet)
	healthChecker := NewHealthChecker()
	healthChecker.SetHealthCheckInterval(100 * time.Millisecond)
	healthChecker.SetHealthCheckTimeout(50 * time.Millisecond)
	scheduler.SetConfig(healthChecker)

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Start health check workers in background
	go scheduler.StartHealthCheckWorkers(ctx)

	// Wait a bit for workers to start
	time.Sleep(200 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait a bit for workers to stop
	time.Sleep(100 * time.Millisecond)

	// Context cancellation test completed successfully
	// The scheduler should have stopped gracefully
}

func TestContextTimeout(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	podSet := NewPodSet()

	// Create scheduler
	scheduler := NewScheduler(clientset, podSet)
	healthChecker := NewHealthChecker()
	healthChecker.SetHealthCheckInterval(100 * time.Millisecond)
	scheduler.SetConfig(healthChecker)

	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Start workers in background
	go scheduler.StartHealthCheckWorkers(ctx)

	// Wait for context timeout
	time.Sleep(300 * time.Millisecond)

	// Verify context has timed out
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
}

func TestPodReadinessCheck(t *testing.T) {
	podSet := NewPodSet()

	// Test pod without Ready condition should not be added
	notReadyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "not-ready-pod",
			Namespace:   "default",
			Annotations: map[string]string{"endpoint-health-checker.io/enabled": "true"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "192.168.1.200",
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}

	podSet.AddOrUpdate(notReadyPod)
	count, _ := podSet.GetStats()
	assert.Equal(t, 0, count, "Pod without Ready=True should not be added")

	// Test pod with Ready condition should be added
	readyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "ready-pod",
			Namespace:   "default",
			Annotations: map[string]string{"endpoint-health-checker.io/enabled": "true"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "192.168.1.201",
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	podSet.AddOrUpdate(readyPod)
	count, _ = podSet.GetStats()
	assert.Equal(t, 1, count, "Pod with Ready=True should be added")

	// Test pod without any conditions should not be added
	noConditionsPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "no-conditions-pod",
			Namespace:   "default",
			Annotations: map[string]string{"endpoint-health-checker.io/enabled": "true"},
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodRunning,
			PodIP:      "192.168.1.202",
			Conditions: []corev1.PodCondition{},
		},
	}

	podSet.AddOrUpdate(noConditionsPod)
	count, _ = podSet.GetStats()
	assert.Equal(t, 1, count, "Pod without conditions should not be added, count should remain 1")
}
