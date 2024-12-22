// overlay_test.go
package main

import (
	"context"
	"testing"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateDaemonSet(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	namespace := "kube-system"
	app := "overlaytest"

	daemonset := &apps.DaemonSet{
		ObjectMeta: meta.ObjectMeta{
			Name: app,
		},
		Spec: apps.DaemonSetSpec{
			Selector: &meta.LabelSelector{
				MatchLabels: map[string]string{
					"app": app,
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Labels: map[string]string{
						"app": app,
					},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  app,
							Image: "test-image",
						},
					},
				},
			},
		},
	}

	_, err := clientset.AppsV1().DaemonSets(namespace).Create(context.TODO(), daemonset, meta.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create DaemonSet: %v", err)
	}

	_, err = clientset.AppsV1().DaemonSets(namespace).Get(context.TODO(), app, meta.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get DaemonSet: %v", err)
	}
}

func TestListPods(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&core.Pod{
			ObjectMeta: meta.ObjectMeta{
				Name:      "test-pod",
				Namespace: "kube-system",
				Labels:    map[string]string{"app": "overlaytest"},
			},
			Status: core.PodStatus{
				PodIP: "192.168.1.1",
			},
		},
	)

	pods, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), meta.ListOptions{
		LabelSelector: "app=overlaytest",
	})
	if err != nil {
		t.Fatalf("Failed to list Pods: %v", err)
	}

	if len(pods.Items) != 1 {
		t.Fatalf("Expected 1 pod, got %d", len(pods.Items))
	}
}
