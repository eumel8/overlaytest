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

const appversion = "1.0.6"

func getKubeconfigPath() string {
	if os.Getenv("KUBECONFIG") != "" {
		return os.Getenv("KUBECONFIG")
	} else if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	} else {
		return ""
	}
}

func createDaemonSetSpec(namespace, app, image string) *apps.DaemonSet {
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
}

func createPingCommand(targetIP string) []string {
	return []string{
		"sh",
		"-c",
		"ping -c 2 " + targetIP + " > /dev/null 2>&1",
	}
}

func validatePodIP(podIP string) bool {
	return net.ParseIP(podIP) != nil
}

func main() {
	// install namespace and app name
	var kubeconfig *string
	namespace := "kube-system"
	app := "overlaytest"
	image := "mtr.devops.telekom.de/mcsps/swiss-army-knife:latest"

	// load kube-config file
	defaultPath := getKubeconfigPath()
	kubeconfig = flag.String("kubeconfig", defaultPath, "(optional) absolute path to the kubeconfig file")
	version := flag.Bool("version", false, "app version")
	reuse := flag.Bool("reuse", false, "reuse existing deployment")

	flag.Parse()

	// print app version and exit
	if *version {
		fmt.Println("version", appversion)
		os.Exit(0)
	}

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
	fmt.Printf("Welcome to the overlaytest.\n\n")

	daemonset := createDaemonSetSpec(namespace, app, image)

	if *reuse != true {
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
				fmt.Printf("Error getting daemonset: %v", err)
				panic(err.Error())
			}
			if obj.Status.NumberReady != 0 {
				fmt.Printf("all pods ready\n")
				break
			}
			time.Sleep(2 * time.Second)
		}
	}

	// load list of PODs
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), meta.ListOptions{LabelSelector: "app=overlaytest"})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d nodes in the cluster\n", len(pods.Items))

	// wait here again if all PODs become an ip-address
	fmt.Printf("checking pod network...\n")
	for _, pod := range pods.Items {
		for {
			podi, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), pod.ObjectMeta.Name, meta.GetOptions{})

			if err != nil {
				panic(err.Error())
			}

			if validatePodIP(podi.Status.PodIP) {
				fmt.Println(podi.ObjectMeta.Name, "ready", podi.Status.PodIP)
				// fmt.Println(podi.ObjectMeta.Name, "ready")
				break
			}
		}
	}
	fmt.Printf("all pods have network\n")

	// loop the pod list for each node for the network test
	fmt.Printf("=> Start network overlay test\n")
	// refresh pod object list
	pods, err = clientset.CoreV1().Pods(namespace).List(context.TODO(), meta.ListOptions{LabelSelector: "app=overlaytest"})
	for _, upod := range pods.Items {
		for _, pod := range pods.Items {
			// fmt.Println("Podname: ", pod.ObjectMeta.Name)
			// fmt.Println("PodIP: ", pod.Status.PodIP)
			// fmt.Println("Nodename: ", pod.Spec.NodeName)
			cmd := createPingCommand(upod.Status.PodIP)
			req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(pod.ObjectMeta.Name).Namespace(namespace).SubResource("exec").VersionedParams(&core.PodExecOptions{
				Command: cmd,
				Stdin:   true,
				Stdout:  true,
				Stderr:  true,
			}, scheme.ParameterCodec)

			exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
			if err != nil {
				fmt.Printf("error while creating Executor: %v\n", err)
			}

			err = exec.Stream(remotecommand.StreamOptions{
				Stdin:  os.Stdin,
				Stdout: os.Stdout,
				Stderr: os.Stderr,
				Tty:    false,
			})
			if err != nil {
				fmt.Printf(upod.Spec.NodeName, " can NOT reach ", pod.Spec.NodeName,"\n")
			} else {
				fmt.Printf(upod.Spec.NodeName, " can reach ", pod.Spec.NodeName,"\n")
			}

		}
	}
	fmt.Printf("=> End network overlay test\n")
	fmt.Printf("Call me again to remove installed cluster resources\n")
}
