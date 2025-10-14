package controller

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type Controller struct {
	clientset       kubernetes.Interface
	informerFactory kubeinformers.SharedInformerFactory
	podInformer     cache.SharedIndexInformer
	podLister       v1.PodLister
	podSynced       cache.InformerSynced
	podSet          *PodSet
}

func NewController(clientset kubernetes.Interface, resync time.Duration, podSet *PodSet) *Controller {
	factory := kubeinformers.NewSharedInformerFactory(clientset, resync)
	podInformer := factory.Core().V1().Pods().Informer()

	c := &Controller{
		clientset:       clientset,
		informerFactory: factory,
		podInformer:     podInformer,
		podLister:       factory.Core().V1().Pods().Lister(),
		podSynced:       podInformer.HasSynced,
		podSet:          podSet,
	}

	handler, err := podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onPodAdd,
		UpdateFunc: c.onPodUpdate,
		DeleteFunc: c.onPodDelete,
	})
	if err != nil {
		klog.Errorf("Failed to add event handler: %v", err)
	}
	_ = handler

	return c
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	klog.Info("Starting controller informers...")

	// Start the informer factory
	c.informerFactory.Start(stopCh)

	// Wait for all informers to sync
	if !cache.WaitForCacheSync(stopCh, c.podSynced) {
		klog.Fatalf("Failed to sync pod informer")
	}

	klog.Info("All informers synced. Controller is running.")
	<-stopCh
}

func (c *Controller) onPodAdd(obj interface{}) {
	pod := obj.(*corev1.Pod)
	c.podSet.AddOrUpdate(pod)
}

func (c *Controller) onPodUpdate(oldObj, newObj interface{}) {
	pod := newObj.(*corev1.Pod)
	c.podSet.AddOrUpdate(pod)
}

func (c *Controller) onPodDelete(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		deletedObj := obj.(cache.DeletedFinalStateUnknown)
		pod = deletedObj.Obj.(*corev1.Pod)
		klog.Infof("Received delete event for pod %s/%s (from DeletedFinalStateUnknown)", pod.Namespace, pod.Name)
	} else {
		klog.Infof("Received delete event for pod %s/%s", pod.Namespace, pod.Name)
	}

	// If PodIP is empty, use namespace and name to delete
	if pod.Status.PodIP == "" {
		klog.Infof("PodIP is empty for deleted pod %s/%s, using namespace/name to delete", pod.Namespace, pod.Name)
		c.podSet.DeleteByNamespaceAndName(pod.Namespace, pod.Name)
	} else {
		c.podSet.Delete(pod)
	}
}
