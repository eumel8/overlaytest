// overlay.go
package main

import (
	"context"
	"flag"
	"fmt"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

const appVersion = "1.0.5"

func parseFlags() (string, bool, bool) {
	var kubeconfig *string
	if home := os.Getenv("KUBECONFIG"); home != "" {
		kubeconfig = flag.String("kubeconfig", home, "Path to the kubeconfig file")
	} else if home, err := os.UserHomeDir(); err == nil {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "Path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "Path to the kubeconfig file")
	}
	version := flag.Bool("version", false, "Display app version")
	reuse := flag.Bool("reuse", false, "Reuse existing deployment")
	flag.Parse()

	return *kubeconfig, *version, *reuse
}

func getKubernetesClient(kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}
	return kubernetes.NewForConfig(config)
}

func createDaemonSet(clientset kubernetes.Interface, namespace, app string) error {
	daemonsetsClient := clientset.AppsV1().DaemonSets(namespace)
	daemonset := &apps.DaemonSet{
		ObjectMeta: meta.ObjectMeta{
			Name: app,
		},
		Spec: apps.DaemonSetSpec{
			Selector: &meta.LabelSelector{
				MatchLabels: map[string]string{"app": app},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Labels: map[string]string{"app": app},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{{
						Name:  app,
						Image: "test-image",
					}},
				},
			},
		},
	}

	_, err := daemonsetsClient.Create(context.TODO(), daemonset, meta.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create DaemonSet: %w", err)
	}
	return nil
}

func main() {
	kubeconfig, version, reuse := parseFlags()

	if version {
		fmt.Println("Version:", appVersion)
		return
	}

	clientset, err := getKubernetesClient(kubeconfig)
	if err != nil {
		fmt.Printf("Error creating Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	if !reuse {
		err := createDaemonSet(clientset, "kube-system", "overlaytest")
		if err != nil {
			fmt.Printf("Error creating DaemonSet: %v\n", err)
			os.Exit(1)
		}
	}
}

