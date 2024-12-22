// overlay_test.go
package main

import (
	"context"
	"testing"

	//apps "k8s.io/api/apps/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)


func TestCreateDaemonSet(t *testing.T) {
	clientset := fake.NewSimpleClientset() // Fake clientset
	err := createDaemonSet(clientset, "kube-system", "overlaytest") // Use fake clientset
	if err != nil {
		t.Fatalf("Error creating DaemonSet: %v", err)
	}

	_, err = clientset.AppsV1().DaemonSets("kube-system").Get(context.TODO(), "overlaytest", meta.GetOptions{}) // Use meta.GetOptions
	if err != nil {
		t.Fatalf("DaemonSet not found: %v", err)
	}
}

func TestParseFlags(t *testing.T) {
	// Simulate flag input
	kubeconfig, version, reuse := parseFlags()
	if kubeconfig == "" {
		t.Errorf("Expected default kubeconfig, got empty string")
	}
	if version {
		t.Errorf("Expected version false, got true")
	}
	if reuse {
		t.Errorf("Expected reuse false, got true")
	}
}

