package overlaytest

import (
	"context"
	"fmt"
	"net"
	"os"

	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// CreatePingCommand creates a ping command for the target IP
func CreatePingCommand(targetIP string) []string {
	return []string{
		"sh",
		"-c",
		"ping -c 2 " + targetIP + " > /dev/null 2>&1",
	}
}

// ValidatePodIP checks if the provided IP address is valid
func ValidatePodIP(podIP string) bool {
	return net.ParseIP(podIP) != nil
}

// RunNetworkTest executes network overlay tests between all pods
func RunNetworkTest(ctx context.Context, clientset kubernetes.Interface, config *rest.Config, namespace string) error {
	// Refresh pod object list
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, meta.ListOptions{LabelSelector: "app=overlaytest"})
	if err != nil {
		return err
	}

	for _, upod := range pods.Items {
		for _, pod := range pods.Items {
			cmd := CreatePingCommand(upod.Status.PodIP)
			req := clientset.CoreV1().RESTClient().Post().
				Resource("pods").
				Name(pod.ObjectMeta.Name).
				Namespace(namespace).
				SubResource("exec").
				VersionedParams(&core.PodExecOptions{
					Command: cmd,
					Stdin:   true,
					Stdout:  true,
					Stderr:  true,
				}, scheme.ParameterCodec)

			exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
			if err != nil {
				fmt.Printf("error while creating Executor: %v\n", err)
				continue
			}

			err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
				Stdin:  os.Stdin,
				Stdout: os.Stdout,
				Stderr: os.Stderr,
				Tty:    false,
			})
			if err != nil {
				fmt.Printf("%s can NOT reach %s\n", upod.Spec.NodeName, pod.Spec.NodeName)
			} else {
				fmt.Printf("%s can reach %s\n", upod.Spec.NodeName, pod.Spec.NodeName)
			}
		}
	}
	return nil
}
