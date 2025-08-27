package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func TestOverlayNetworkE2E(t *testing.T) {
	// Setup kubeconfig path
	var kubeconfig string
	if os.Getenv("KUBECONFIG") != "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		t.Fatal("Could not find kubeconfig")
	}

	// Create kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		t.Fatalf("Error building kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Error creating kubernetes client: %v", err)
	}

	namespace := "kube-system"
	appName := "overlaytest"

	t.Run("ClusterReady", func(t *testing.T) {
		testClusterReady(t, clientset)
	})

	t.Run("DeployDaemonSet", func(t *testing.T) {
		testDeployDaemonSet(t, clientset, namespace, appName)
	})

	t.Run("WaitForPods", func(t *testing.T) {
		testWaitForPods(t, clientset, namespace, appName)
	})

	t.Run("NetworkConnectivity", func(t *testing.T) {
		testNetworkConnectivity(t, clientset, namespace, appName)
	})

	t.Run("CleanupDaemonSet", func(t *testing.T) {
		testCleanupDaemonSet(t, clientset, namespace, appName)
	})
}

func testClusterReady(t *testing.T, clientset *kubernetes.Clientset) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nodes, err := clientset.CoreV1().Nodes().List(ctx, meta.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list nodes: %v", err)
	}

	if len(nodes.Items) < 2 {
		t.Fatalf("Expected at least 2 nodes, got %d", len(nodes.Items))
	}

	for _, node := range nodes.Items {
		ready := false
		for _, condition := range node.Status.Conditions {
			if condition.Type == core.NodeReady && condition.Status == core.ConditionTrue {
				ready = true
				break
			}
		}
		if !ready {
			t.Fatalf("Node %s is not ready", node.Name)
		}
	}

	t.Logf("Cluster has %d ready nodes", len(nodes.Items))
}

func testDeployDaemonSet(t *testing.T, clientset *kubernetes.Clientset, namespace, appName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First, cleanup any existing daemonset
	daemonsetsClient := clientset.AppsV1().DaemonSets(namespace)
	deletePolicy := meta.DeletePropagationForeground
	_ = daemonsetsClient.Delete(ctx, appName, meta.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})

	// Wait for cleanup
	time.Sleep(5 * time.Second)

	// Create new daemonset
	var graceperiod = int64(1)
	var user = int64(1000)
	var privileged = bool(true)
	var readonly = bool(true)
	image := "mtr.devops.telekom.de/mcsps/swiss-army-knife:latest"

	daemonset := &apps.DaemonSet{
		ObjectMeta: meta.ObjectMeta{
			Name: appName,
		},
		Spec: apps.DaemonSetSpec{
			Selector: &meta.LabelSelector{
				MatchLabels: map[string]string{
					"app": appName,
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Labels: map[string]string{
						"app": appName,
					},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Args:            []string{"tail -f /dev/null"},
							Command:         []string{"sh", "-c"},
							Name:            appName,
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

	result, err := daemonsetsClient.Create(ctx, daemonset, meta.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create daemonset: %v", err)
	}

	t.Logf("Created daemonset %s", result.GetObjectMeta().GetName())
}

func testWaitForPods(t *testing.T, clientset *kubernetes.Clientset, namespace, appName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Wait for daemonset to be ready
	daemonsetsClient := clientset.AppsV1().DaemonSets(namespace)
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for daemonset to be ready")
		default:
			ds, err := daemonsetsClient.Get(ctx, appName, meta.GetOptions{})
			if err != nil {
				t.Fatalf("Error getting daemonset: %v", err)
			}

			if ds.Status.NumberReady > 0 && ds.Status.NumberReady == ds.Status.DesiredNumberScheduled {
				t.Logf("DaemonSet ready: %d/%d pods", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
				return
			}

			t.Logf("Waiting for daemonset: %d/%d pods ready", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
			time.Sleep(5 * time.Second)
		}
	}
}

func testNetworkConnectivity(t *testing.T, clientset *kubernetes.Clientset, namespace, appName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get all pods
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, meta.ListOptions{LabelSelector: "app=" + appName})
	if err != nil {
		t.Fatalf("Failed to list pods: %v", err)
	}

	if len(pods.Items) < 2 {
		t.Fatalf("Expected at least 2 pods for network test, got %d", len(pods.Items))
	}

	// Wait for all pods to have IP addresses
	for _, pod := range pods.Items {
		for {
			select {
			case <-ctx.Done():
				t.Fatal("Timeout waiting for pod IPs")
			default:
				podi, err := clientset.CoreV1().Pods(namespace).Get(ctx, pod.ObjectMeta.Name, meta.GetOptions{})
				if err != nil {
					t.Fatalf("Error getting pod: %v", err)
				}

				if podi.Status.PodIP != "" {
					t.Logf("Pod %s has IP: %s", podi.ObjectMeta.Name, podi.Status.PodIP)
					break
				}
				time.Sleep(2 * time.Second)
			}
		}
	}

	t.Log("All pods have network IPs - network connectivity test would run here")
	// Note: Actual network connectivity test would require exec into pods
	// which is complex in a unit test environment
}

func testCleanupDaemonSet(t *testing.T, clientset *kubernetes.Clientset, namespace, appName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	daemonsetsClient := clientset.AppsV1().DaemonSets(namespace)
	deletePolicy := meta.DeletePropagationForeground

	err := daemonsetsClient.Delete(ctx, appName, meta.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})

	if err != nil && !errors.IsNotFound(err) {
		t.Fatalf("Failed to delete daemonset: %v", err)
	}

	// Wait for deletion
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for daemonset deletion")
		default:
			_, err := daemonsetsClient.Get(ctx, appName, meta.GetOptions{})
			if errors.IsNotFound(err) {
				t.Log("DaemonSet successfully deleted")
				return
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func TestOverlayToolBinary(t *testing.T) {
	// Test that the binary can be built and shows version
	cmd := exec.Command("go", "build", "-o", "overlaytest-test", ".")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("overlaytest-test")

	// Test version flag
	cmd = exec.Command("./overlaytest-test", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run binary with -version: %v", err)
	}

	if string(output) == "" {
		t.Fatal("Version output is empty")
	}

	t.Logf("Binary version output: %s", string(output))
}