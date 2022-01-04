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
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/homedir"
	"net"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// install namespace and app name
	var kubeconfig *string
	namespace := "kube-system"
	app := "overlaytest"
	var graceperiod = int64(1)
	var user = int64(1000)
	var privledged = bool(true)
	var readonly = bool(true)
	image := "mtr.external.otc.telekomcloud.com/mcsps/swiss-army-knife:latest"

	// load kube-config file
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// how this will work with env var?
	// *argKubecfgFile = os.Getenv("KUBECONFIG")

	// prepare the DaemonSet resource
	daemonsetsClient := clientset.AppsV1().DaemonSets(namespace)
	fmt.Println("Welcome to the overlaytest.\n\n")

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
								AllowPrivilegeEscalation: &privledged,
								Privileged:               &privledged,
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

	// create DaemonSet, delete it if exists
	fmt.Println("Creating daemonset...")
	result, err := daemonsetsClient.Create(context.TODO(), daemonset, meta.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		fmt.Println("daemonset already exists, deleting ... & exit")
		deletePolicy := meta.DeletePropagationForeground
		if err := daemonsetsClient.Delete(context.TODO(), app, meta.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}); err != nil {
			panic(err)
		}
		os.Exit(1)
	} else if err != nil {
		panic(err)
	}
	fmt.Printf("Created daemonset %q.\n", result.GetObjectMeta().GetName())

	// wait here if all PODs are ready
	for {
		obj, err := clientset.AppsV1().DaemonSets(namespace).Get(context.TODO(), "overlaytest", meta.GetOptions{})
		if err != nil {
			fmt.Println("Error getting daemonset: %v", err)
			panic(err.Error())
		}
		if obj.Status.NumberReady != 0 {
			fmt.Println("all pods ready")
			break
		}
		time.Sleep(2 * time.Second)
	}

	// load list of PODs
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), meta.ListOptions{LabelSelector: "app=overlaytest"})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d nodes in the cluster\n", len(pods.Items))

	// wait here again if all PODs become an ip-address
	fmt.Println("checking pod network...")
	for _, pod := range pods.Items {
		for {
			podi, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), pod.ObjectMeta.Name, meta.GetOptions{})

			if err != nil {
				panic(err.Error())
			}

			if net.ParseIP(podi.Status.PodIP) != nil {
				fmt.Println(podi.ObjectMeta.Name, "ready", podi.Status.PodIP)
				// fmt.Println(podi.ObjectMeta.Name, "ready")
				break
			}
		}
	}
	fmt.Println("all pods have network\n")

	// loop the pod list for each node for the network test
	fmt.Println("=> Start network overlay test\n")
	// refresh pod object list
	pods, err = clientset.CoreV1().Pods(namespace).List(context.TODO(), meta.ListOptions{LabelSelector: "app=overlaytest"})
	for _, upod := range pods.Items {
		for _, pod := range pods.Items {
			// fmt.Println("Podname: ", pod.ObjectMeta.Name)
			// fmt.Println("PodIP: ", pod.Status.PodIP)
			// fmt.Println("Nodename: ", pod.Spec.NodeName)
			cmd := []string{
				"sh",
				"-c",
				"ping -c 2 " + upod.Status.PodIP + " > /dev/null 2>&1",
			}
			req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(pod.ObjectMeta.Name).Namespace(namespace).SubResource("exec").VersionedParams(&core.PodExecOptions{
				Command: cmd,
				Stdin:   true,
				Stdout:  true,
				Stderr:  true,
			}, scheme.ParameterCodec)

			exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
			if err != nil {
				fmt.Println("error while creating Executor: %v", err)
			}

			err = exec.Stream(remotecommand.StreamOptions{
				Stdin:  os.Stdin,
				Stdout: os.Stdout,
				Stderr: os.Stderr,
				Tty:    false,
			})
			if err != nil {
				fmt.Println(upod.Spec.NodeName, " can NOT reach ", pod.Spec.NodeName)
			} else {
				fmt.Println(upod.Spec.NodeName, " can reach ", pod.Spec.NodeName)
			}

		}
	}
	fmt.Println("=> End network overlay test\n")
	fmt.Println("Call me again to remove installed cluster resources\n")
}
