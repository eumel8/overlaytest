package overlaytest

import (
	"context"
	"fmt"
	"time"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CreateDaemonSetSpec creates a DaemonSet specification
func CreateDaemonSetSpec(namespace, app, image string) *apps.DaemonSet {
	var graceperiod = int64(1)
	var user = int64(1000)
	var privledged = bool(true)
	var readonly = bool(true)

	return &apps.DaemonSet{
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
								AllowPrivilegeEscalation: &privledged,
								Privileged:               &privledged,
								ReadOnlyRootFilesystem:   &readonly,
								RunAsGroup:               &user,
								RunAsUser:                &user,
								SeccompProfile: &core.SeccompProfile{
									Type: core.SeccompProfileTypeRuntimeDefault,
								},
							},
							Resources: core.ResourceRequirements{
								Requests: core.ResourceList{
									core.ResourceCPU:    resource.MustParse("100m"),
									core.ResourceMemory: resource.MustParse("64Mi"),
								},
								Limits: core.ResourceList{
									core.ResourceCPU:    resource.MustParse("200m"),
									core.ResourceMemory: resource.MustParse("128Mi"),
								},
							},
						},
					},
					TerminationGracePeriodSeconds: &graceperiod,
					Tolerations: []core.Toleration{{
						Operator: "Exists",
					}},
					SecurityContext: &core.PodSecurityContext{
						FSGroup: &user,
						SeccompProfile: &core.SeccompProfile{
							Type: core.SeccompProfileTypeRuntimeDefault,
						},
					},
				},
			},
		},
	}
}

// CreateOrReuseDaemonSet creates a new DaemonSet or exits if it exists (when not reusing)
func CreateOrReuseDaemonSet(ctx context.Context, clientset kubernetes.Interface, config *Config, reuse bool) error {
	daemonsetsClient := clientset.AppsV1().DaemonSets(config.Namespace)
	daemonset := CreateDaemonSetSpec(config.Namespace, config.AppName, config.Image)

	if !reuse {
		fmt.Println("Creating daemonset...")
		result, err := daemonsetsClient.Create(ctx, daemonset, meta.CreateOptions{})
		if errors.IsAlreadyExists(err) {
			fmt.Println("daemonset already exists, deleting ... & exit")
			deletePolicy := meta.DeletePropagationForeground
			if err := daemonsetsClient.Delete(ctx, config.AppName, meta.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			}); err != nil {
				return err
			}
			return fmt.Errorf("daemonset already existed, deleted it - please run again")
		} else if err != nil {
			return err
		}
		fmt.Printf("Created daemonset %q.\n", result.GetObjectMeta().GetName())
	}
	return nil
}

// WaitForDaemonSetReady waits for all DaemonSet pods to be ready
func WaitForDaemonSetReady(ctx context.Context, clientset kubernetes.Interface, namespace, name string) error {
	for {
		obj, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, meta.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting daemonset: %w", err)
		}
		if obj.Status.NumberReady != 0 {
			fmt.Printf("all pods ready\n")
			break
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}

// GetOverlayTestPods returns all pods with the overlaytest label
func GetOverlayTestPods(ctx context.Context, clientset kubernetes.Interface, namespace string) (*core.PodList, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, meta.ListOptions{LabelSelector: "app=overlaytest"})
	if err != nil {
		return nil, err
	}
	return pods, nil
}

// WaitForPodNetwork waits for all pods to have valid IP addresses
func WaitForPodNetwork(ctx context.Context, clientset kubernetes.Interface, namespace string, pods []core.Pod) error {
	fmt.Printf("checking pod network...\n")
	for _, pod := range pods {
		for {
			podi, err := clientset.CoreV1().Pods(namespace).Get(ctx, pod.ObjectMeta.Name, meta.GetOptions{})
			if err != nil {
				return err
			}

			if ValidatePodIP(podi.Status.PodIP) {
				fmt.Println(podi.ObjectMeta.Name, "ready", podi.Status.PodIP)
				break
			}
		}
	}
	fmt.Printf("all pods have network\n")
	return nil
}
