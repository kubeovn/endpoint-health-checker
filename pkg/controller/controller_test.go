package controller

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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