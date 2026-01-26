/*
  The Overlay Network Test installs a DaemonSet in the target cluster
  and send 2 ping to each node to check if the Overlay Network is in
  a workable state. The state will be print out.
  Requires a working .kube/config file or a param -kubeconfig with a
  working kube-config file.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/eumel8/overlaytest/pkg/overlaytest"
)

func main() {
	// Parse flags
	config := overlaytest.DefaultConfig()
	defaultPath := overlaytest.GetKubeconfigPath()

	kubeconfig := flag.String("kubeconfig", defaultPath, "(optional) absolute path to the kubeconfig file")
	version := flag.Bool("version", false, "app version")
	reuse := flag.Bool("reuse", false, "reuse existing deployment")

	flag.Parse()

	// Handle version flag
	if *version {
		fmt.Println("version", overlaytest.GetVersion())
		os.Exit(0)
	}

	// Update config
	config.Kubeconfig = *kubeconfig
	config.Reuse = *reuse

	// Run the overlay test
	ctx := context.Background()
	if err := runOverlayTest(ctx, config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runOverlayTest(ctx context.Context, config *overlaytest.Config) error {
	// Create Kubernetes client
	clientset, restConfig, err := overlaytest.NewKubernetesClient(config.Kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	fmt.Printf("Welcome to the overlaytest.\n\n")

	// Create or reuse DaemonSet
	if err := overlaytest.CreateOrReuseDaemonSet(ctx, clientset, config, config.Reuse); err != nil {
		return err
	}

	// Wait for DaemonSet ready
	if !config.Reuse {
		if err := overlaytest.WaitForDaemonSetReady(ctx, clientset, config.Namespace, config.AppName); err != nil {
			return err
		}
	}

	// Get pods
	pods, err := overlaytest.GetOverlayTestPods(ctx, clientset, config.Namespace)
	if err != nil {
		return err
	}
	fmt.Printf("There are %d nodes in the cluster\n", len(pods.Items))

	// Wait for pod network
	if err := overlaytest.WaitForPodNetwork(ctx, clientset, config.Namespace, pods.Items); err != nil {
		return err
	}

	// Run network test
	fmt.Printf("\n=> Start network overlay test\n")
	if err := overlaytest.RunNetworkTest(ctx, clientset, restConfig, config.Namespace); err != nil {
		return err
	}
	fmt.Printf("=> End network overlay test\n")

	fmt.Printf("\nCall me again to remove installed cluster resources\n")
	return nil
}
