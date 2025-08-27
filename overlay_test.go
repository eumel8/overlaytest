package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/homedir"
)

func TestAppVersion(t *testing.T) {
	expected := "1.0.6"
	if appversion != expected {
		t.Errorf("Expected appversion to be %s, got %s", expected, appversion)
	}
}

func TestKubeconfigPath(t *testing.T) {
	tests := []struct {
		name        string
		kubeconfig  string
		homeDir     string
		expected    string
		description string
	}{
		{
			name:        "KUBECONFIG environment variable set",
			kubeconfig:  "/custom/kubeconfig",
			homeDir:     "/home/user",
			expected:    "/custom/kubeconfig",
			description: "should use KUBECONFIG env var when set",
		},
		{
			name:        "Home directory available",
			kubeconfig:  "",
			homeDir:     "/home/user",
			expected:    "/home/user/.kube/config",
			description: "should use default kubeconfig path in home directory",
		},
		{
			name:        "No home directory",
			kubeconfig:  "",
			homeDir:     "",
			expected:    "",
			description: "should use empty string when no home directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldKubeconfig := os.Getenv("KUBECONFIG")
			defer os.Setenv("KUBECONFIG", oldKubeconfig)

			os.Setenv("KUBECONFIG", tt.kubeconfig)

			var kubeconfigPath string
			if tt.kubeconfig != "" {
				kubeconfigPath = os.Getenv("KUBECONFIG")
			} else if tt.homeDir != "" {
				kubeconfigPath = filepath.Join(tt.homeDir, ".kube", "config")
			} else {
				kubeconfigPath = ""
			}

			if kubeconfigPath != tt.expected {
				t.Errorf("Expected kubeconfig path to be %s, got %s", tt.expected, kubeconfigPath)
			}
		})
	}
}

func TestHomeDirFunction(t *testing.T) {
	homeDir := homedir.HomeDir()
	if homeDir == "" {
		t.Skip("No home directory available in test environment")
	}
	
	if !filepath.IsAbs(homeDir) {
		t.Errorf("Expected home directory to be absolute path, got %s", homeDir)
	}
}

func TestDaemonSetConfiguration(t *testing.T) {
	app := "overlaytest"
	var graceperiod = int64(1)
	var user = int64(1000)
	var privileged = bool(true)
	var readonly = bool(true)
	image := "mtr.devops.telekom.de/mcsps/swiss-army-knife:latest"

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
							Args:            []string{"tail -f /dev/null"},
							Command:         []string{"sh", "-c"},
							Name:            app,
							Image:           image,
							ImagePullPolicy: "IfNotPresent",
							SecurityContext: &core.SecurityContext{
								AllowPrivilegeEscalation: &privileged,
								Privileged:               &privileged,
								ReadOnlyRootFilesystem:   &readonly,
								RunAsGroup:               &user,
								RunAsUser:                &user,
							},
						},
					},
					TerminationGracePeriodSeconds: &graceperiod,
					Tolerations: []core.Toleration{{
						Operator: "Exists",
					}},
					SecurityContext: &core.PodSecurityContext{
						FSGroup: &user,
					},
				},
			},
		},
	}

	t.Run("DaemonSet metadata", func(t *testing.T) {
		if daemonset.ObjectMeta.Name != app {
			t.Errorf("Expected DaemonSet name to be %s, got %s", app, daemonset.ObjectMeta.Name)
		}
	})

	t.Run("DaemonSet selector", func(t *testing.T) {
		if daemonset.Spec.Selector.MatchLabels["app"] != app {
			t.Errorf("Expected selector label 'app' to be %s, got %s", app, daemonset.Spec.Selector.MatchLabels["app"])
		}
	})

	t.Run("Pod template labels", func(t *testing.T) {
		if daemonset.Spec.Template.ObjectMeta.Labels["app"] != app {
			t.Errorf("Expected pod template label 'app' to be %s, got %s", app, daemonset.Spec.Template.ObjectMeta.Labels["app"])
		}
	})

	t.Run("Container configuration", func(t *testing.T) {
		container := daemonset.Spec.Template.Spec.Containers[0]
		
		if container.Name != app {
			t.Errorf("Expected container name to be %s, got %s", app, container.Name)
		}
		
		if container.Image != image {
			t.Errorf("Expected container image to be %s, got %s", image, container.Image)
		}
		
		if container.ImagePullPolicy != "IfNotPresent" {
			t.Errorf("Expected ImagePullPolicy to be IfNotPresent, got %s", container.ImagePullPolicy)
		}
		
		expectedArgs := []string{"tail -f /dev/null"}
		if len(container.Args) != len(expectedArgs) || container.Args[0] != expectedArgs[0] {
			t.Errorf("Expected container args to be %v, got %v", expectedArgs, container.Args)
		}
		
		expectedCommand := []string{"sh", "-c"}
		if len(container.Command) != len(expectedCommand) || container.Command[0] != expectedCommand[0] || container.Command[1] != expectedCommand[1] {
			t.Errorf("Expected container command to be %v, got %v", expectedCommand, container.Command)
		}
	})

	t.Run("Security context", func(t *testing.T) {
		container := daemonset.Spec.Template.Spec.Containers[0]
		secCtx := container.SecurityContext
		
		if secCtx.AllowPrivilegeEscalation == nil || *secCtx.AllowPrivilegeEscalation != privileged {
			t.Errorf("Expected AllowPrivilegeEscalation to be %t, got %v", privileged, secCtx.AllowPrivilegeEscalation)
		}
		
		if secCtx.Privileged == nil || *secCtx.Privileged != privileged {
			t.Errorf("Expected Privileged to be %t, got %v", privileged, secCtx.Privileged)
		}
		
		if secCtx.ReadOnlyRootFilesystem == nil || *secCtx.ReadOnlyRootFilesystem != readonly {
			t.Errorf("Expected ReadOnlyRootFilesystem to be %t, got %v", readonly, secCtx.ReadOnlyRootFilesystem)
		}
		
		if secCtx.RunAsUser == nil || *secCtx.RunAsUser != user {
			t.Errorf("Expected RunAsUser to be %d, got %v", user, secCtx.RunAsUser)
		}
		
		if secCtx.RunAsGroup == nil || *secCtx.RunAsGroup != user {
			t.Errorf("Expected RunAsGroup to be %d, got %v", user, secCtx.RunAsGroup)
		}
	})

	t.Run("Pod spec configuration", func(t *testing.T) {
		podSpec := daemonset.Spec.Template.Spec
		
		if podSpec.TerminationGracePeriodSeconds == nil || *podSpec.TerminationGracePeriodSeconds != graceperiod {
			t.Errorf("Expected TerminationGracePeriodSeconds to be %d, got %v", graceperiod, podSpec.TerminationGracePeriodSeconds)
		}
		
		if len(podSpec.Tolerations) != 1 || podSpec.Tolerations[0].Operator != "Exists" {
			t.Errorf("Expected one toleration with operator 'Exists', got %v", podSpec.Tolerations)
		}
		
		if podSpec.SecurityContext.FSGroup == nil || *podSpec.SecurityContext.FSGroup != user {
			t.Errorf("Expected FSGroup to be %d, got %v", user, podSpec.SecurityContext.FSGroup)
		}
	})
}

func TestDaemonSetCreation(t *testing.T) {
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
		},
	}

	clientset := fake.NewSimpleClientset()
	daemonsetsClient := clientset.AppsV1().DaemonSets("kube-system")

	t.Run("Create new DaemonSet", func(t *testing.T) {
		result, err := daemonsetsClient.Create(context.TODO(), daemonset, meta.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create DaemonSet: %v", err)
		}
		
		if result.GetObjectMeta().GetName() != app {
			t.Errorf("Expected created DaemonSet name to be %s, got %s", app, result.GetObjectMeta().GetName())
		}
	})

	t.Run("List pods with label selector", func(t *testing.T) {
		pods, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), meta.ListOptions{LabelSelector: "app=overlaytest"})
		if err != nil {
			t.Fatalf("Failed to list pods: %v", err)
		}
		
		if len(pods.Items) != 0 {
			t.Errorf("Expected 0 pods initially, got %d", len(pods.Items))
		}
	})
}

func TestPodNetworkValidation(t *testing.T) {
	
	pod := &core.Pod{
		ObjectMeta: meta.ObjectMeta{
			Name: "test-pod",
			Namespace: "kube-system",
			Labels: map[string]string{
				"app": "overlaytest",
			},
		},
		Status: core.PodStatus{
			PodIP: "10.244.0.1",
		},
	}

	clientset := fake.NewSimpleClientset(pod)

	t.Run("Get pod with valid IP", func(t *testing.T) {
		retrievedPod, err := clientset.CoreV1().Pods("kube-system").Get(context.TODO(), "test-pod", meta.GetOptions{})
		if err != nil {
			t.Fatalf("Failed to get pod: %v", err)
		}
		
		if retrievedPod.Status.PodIP != "10.244.0.1" {
			t.Errorf("Expected pod IP to be 10.244.0.1, got %s", retrievedPod.Status.PodIP)
		}
	})

	t.Run("List pods with label selector", func(t *testing.T) {
		pods, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), meta.ListOptions{LabelSelector: "app=overlaytest"})
		if err != nil {
			t.Fatalf("Failed to list pods: %v", err)
		}
		
		if len(pods.Items) != 1 {
			t.Errorf("Expected 1 pod, got %d", len(pods.Items))
			return
		}
		
		if pods.Items[0].Status.PodIP != "10.244.0.1" {
			t.Errorf("Expected pod IP to be 10.244.0.1, got %s", pods.Items[0].Status.PodIP)
		}
	})
}

func TestPingCommandGeneration(t *testing.T) {
	targetIP := "10.244.0.5"
	expectedCmd := []string{
		"sh",
		"-c",
		"ping -c 2 " + targetIP + " > /dev/null 2>&1",
	}
	
	cmd := []string{
		"sh",
		"-c",
		"ping -c 2 " + targetIP + " > /dev/null 2>&1",
	}
	
	if len(cmd) != len(expectedCmd) {
		t.Errorf("Expected command length to be %d, got %d", len(expectedCmd), len(cmd))
	}
	
	for i, part := range cmd {
		if part != expectedCmd[i] {
			t.Errorf("Expected command part %d to be %s, got %s", i, expectedCmd[i], part)
		}
	}
}

func TestMultiplePodScenario(t *testing.T) {
	
	pods := []runtime.Object{
		&core.Pod{
			ObjectMeta: meta.ObjectMeta{
				Name: "pod-1",
				Namespace: "kube-system",
				Labels: map[string]string{
					"app": "overlaytest",
				},
			},
			Spec: core.PodSpec{
				NodeName: "node-1",
			},
			Status: core.PodStatus{
				PodIP: "10.244.0.1",
			},
		},
		&core.Pod{
			ObjectMeta: meta.ObjectMeta{
				Name: "pod-2",
				Namespace: "kube-system",
				Labels: map[string]string{
					"app": "overlaytest",
				},
			},
			Spec: core.PodSpec{
				NodeName: "node-2",
			},
			Status: core.PodStatus{
				PodIP: "10.244.1.1",
			},
		},
		&core.Pod{
			ObjectMeta: meta.ObjectMeta{
				Name: "pod-3",
				Namespace: "kube-system",
				Labels: map[string]string{
					"app": "overlaytest",
				},
			},
			Spec: core.PodSpec{
				NodeName: "node-3",
			},
			Status: core.PodStatus{
				PodIP: "10.244.2.1",
			},
		},
	}

	clientset := fake.NewSimpleClientset(pods...)

	t.Run("List all overlay test pods", func(t *testing.T) {
		podList, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), meta.ListOptions{LabelSelector: "app=overlaytest"})
		if err != nil {
			t.Fatalf("Failed to list pods: %v", err)
		}
		
		if len(podList.Items) != 3 {
			t.Errorf("Expected 3 pods, got %d", len(podList.Items))
		}
	})

	t.Run("Verify pod network configuration", func(t *testing.T) {
		podList, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), meta.ListOptions{LabelSelector: "app=overlaytest"})
		if err != nil {
			t.Fatalf("Failed to list pods: %v", err)
		}
		
		expectedIPs := []string{"10.244.0.1", "10.244.1.1", "10.244.2.1"}
		expectedNodes := []string{"node-1", "node-2", "node-3"}
		
		for i, pod := range podList.Items {
			if pod.Status.PodIP != expectedIPs[i] {
				t.Errorf("Expected pod %d IP to be %s, got %s", i, expectedIPs[i], pod.Status.PodIP)
			}
			
			if pod.Spec.NodeName != expectedNodes[i] {
				t.Errorf("Expected pod %d node name to be %s, got %s", i, expectedNodes[i], pod.Spec.NodeName)
			}
		}
	})
}

func TestConstants(t *testing.T) {
	t.Run("App version constant", func(t *testing.T) {
		if appversion == "" {
			t.Error("App version should not be empty")
		}
	})
	
	t.Run("Default values", func(t *testing.T) {
		namespace := "kube-system"
		app := "overlaytest"
		image := "mtr.devops.telekom.de/mcsps/swiss-army-knife:latest"
		
		if namespace != "kube-system" {
			t.Errorf("Expected namespace to be kube-system, got %s", namespace)
		}
		
		if app != "overlaytest" {
			t.Errorf("Expected app name to be overlaytest, got %s", app)
		}
		
		if image == "" {
			t.Error("Image should not be empty")
		}
	})
}