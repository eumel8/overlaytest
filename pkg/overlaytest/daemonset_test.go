package overlaytest

import (
	"context"
	"testing"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestCreateDaemonSetSpec(t *testing.T) {
	namespace := "test-namespace"
	app := "test-app"
	image := "test-image:latest"

	daemonset := CreateDaemonSetSpec(namespace, app, image)

	t.Run("Metadata", func(t *testing.T) {
		if daemonset.ObjectMeta.Name != app {
			t.Errorf("Expected name %s, got %s", app, daemonset.ObjectMeta.Name)
		}
	})

	t.Run("Selector", func(t *testing.T) {
		if daemonset.Spec.Selector.MatchLabels["app"] != app {
			t.Errorf("Expected selector app label %s, got %s", app, daemonset.Spec.Selector.MatchLabels["app"])
		}
	})

	t.Run("Template labels", func(t *testing.T) {
		if daemonset.Spec.Template.ObjectMeta.Labels["app"] != app {
			t.Errorf("Expected template app label %s, got %s", app, daemonset.Spec.Template.ObjectMeta.Labels["app"])
		}
	})

	t.Run("Container spec", func(t *testing.T) {
		container := daemonset.Spec.Template.Spec.Containers[0]

		if container.Name != app {
			t.Errorf("Expected container name %s, got %s", app, container.Name)
		}

		if container.Image != image {
			t.Errorf("Expected container image %s, got %s", image, container.Image)
		}

		if container.ImagePullPolicy != "IfNotPresent" {
			t.Errorf("Expected ImagePullPolicy IfNotPresent, got %s", container.ImagePullPolicy)
		}

		expectedArgs := []string{"tail -f /dev/null"}
		if len(container.Args) != 1 || container.Args[0] != expectedArgs[0] {
			t.Errorf("Expected args %v, got %v", expectedArgs, container.Args)
		}

		expectedCommand := []string{"sh", "-c"}
		if len(container.Command) != 2 || container.Command[0] != expectedCommand[0] || container.Command[1] != expectedCommand[1] {
			t.Errorf("Expected command %v, got %v", expectedCommand, container.Command)
		}
	})

	t.Run("Security context", func(t *testing.T) {
		container := daemonset.Spec.Template.Spec.Containers[0]
		secCtx := container.SecurityContext

		expectedUser := int64(1000)
		expectedPrivileged := true
		expectedReadonly := true

		if secCtx.RunAsUser == nil || *secCtx.RunAsUser != expectedUser {
			t.Errorf("Expected RunAsUser %d, got %v", expectedUser, secCtx.RunAsUser)
		}

		if secCtx.RunAsGroup == nil || *secCtx.RunAsGroup != expectedUser {
			t.Errorf("Expected RunAsGroup %d, got %v", expectedUser, secCtx.RunAsGroup)
		}

		if secCtx.Privileged == nil || *secCtx.Privileged != expectedPrivileged {
			t.Errorf("Expected Privileged %t, got %v", expectedPrivileged, secCtx.Privileged)
		}

		if secCtx.ReadOnlyRootFilesystem == nil || *secCtx.ReadOnlyRootFilesystem != expectedReadonly {
			t.Errorf("Expected ReadOnlyRootFilesystem %t, got %v", expectedReadonly, secCtx.ReadOnlyRootFilesystem)
		}

		// Test seccomp profile (issue #179)
		if secCtx.SeccompProfile == nil {
			t.Error("Expected SeccompProfile to be set")
		} else if secCtx.SeccompProfile.Type != core.SeccompProfileTypeRuntimeDefault {
			t.Errorf("Expected SeccompProfile type RuntimeDefault, got %s", secCtx.SeccompProfile.Type)
		}
	})

	t.Run("Resource requirements (issue #179)", func(t *testing.T) {
		container := daemonset.Spec.Template.Spec.Containers[0]
		resources := container.Resources

		// Check CPU requests
		cpuRequest := resources.Requests[core.ResourceCPU]
		expectedCPURequest := resource.MustParse("100m")
		if cpuRequest.Cmp(expectedCPURequest) != 0 {
			t.Errorf("Expected CPU request %s, got %s", expectedCPURequest.String(), cpuRequest.String())
		}

		// Check Memory requests
		memRequest := resources.Requests[core.ResourceMemory]
		expectedMemRequest := resource.MustParse("64Mi")
		if memRequest.Cmp(expectedMemRequest) != 0 {
			t.Errorf("Expected Memory request %s, got %s", expectedMemRequest.String(), memRequest.String())
		}

		// Check CPU limits
		cpuLimit := resources.Limits[core.ResourceCPU]
		expectedCPULimit := resource.MustParse("200m")
		if cpuLimit.Cmp(expectedCPULimit) != 0 {
			t.Errorf("Expected CPU limit %s, got %s", expectedCPULimit.String(), cpuLimit.String())
		}

		// Check Memory limits
		memLimit := resources.Limits[core.ResourceMemory]
		expectedMemLimit := resource.MustParse("128Mi")
		if memLimit.Cmp(expectedMemLimit) != 0 {
			t.Errorf("Expected Memory limit %s, got %s", expectedMemLimit.String(), memLimit.String())
		}
	})

	t.Run("Pod spec", func(t *testing.T) {
		podSpec := daemonset.Spec.Template.Spec
		expectedGracePeriod := int64(1)
		expectedFSGroup := int64(1000)

		if podSpec.TerminationGracePeriodSeconds == nil || *podSpec.TerminationGracePeriodSeconds != expectedGracePeriod {
			t.Errorf("Expected TerminationGracePeriodSeconds %d, got %v", expectedGracePeriod, podSpec.TerminationGracePeriodSeconds)
		}

		if len(podSpec.Tolerations) != 1 || podSpec.Tolerations[0].Operator != "Exists" {
			t.Errorf("Expected one toleration with operator 'Exists', got %v", podSpec.Tolerations)
		}

		if podSpec.SecurityContext.FSGroup == nil || *podSpec.SecurityContext.FSGroup != expectedFSGroup {
			t.Errorf("Expected FSGroup %d, got %v", expectedFSGroup, podSpec.SecurityContext.FSGroup)
		}

		// Test pod-level seccomp profile (issue #179)
		if podSpec.SecurityContext.SeccompProfile == nil {
			t.Error("Expected Pod SecurityContext SeccompProfile to be set")
		} else if podSpec.SecurityContext.SeccompProfile.Type != core.SeccompProfileTypeRuntimeDefault {
			t.Errorf("Expected Pod SeccompProfile type RuntimeDefault, got %s", podSpec.SecurityContext.SeccompProfile.Type)
		}
	})
}

func TestDaemonSetSpecConsistency(t *testing.T) {
	namespace1 := "test-ns-1"
	app1 := "test-app-1"
	image1 := "test-image-1"

	namespace2 := "test-ns-2"
	app2 := "test-app-2"
	image2 := "test-image-2"

	ds1 := CreateDaemonSetSpec(namespace1, app1, image1)
	ds2 := CreateDaemonSetSpec(namespace2, app2, image2)

	t.Run("Different parameters produce different specs", func(t *testing.T) {
		if ds1.ObjectMeta.Name == ds2.ObjectMeta.Name {
			t.Error("Expected different DaemonSet names")
		}

		if ds1.Spec.Template.Spec.Containers[0].Image == ds2.Spec.Template.Spec.Containers[0].Image {
			t.Error("Expected different container images")
		}
	})

	t.Run("Security context consistency", func(t *testing.T) {
		sec1 := ds1.Spec.Template.Spec.Containers[0].SecurityContext
		sec2 := ds2.Spec.Template.Spec.Containers[0].SecurityContext

		if *sec1.RunAsUser != *sec2.RunAsUser {
			t.Error("Expected consistent RunAsUser across DaemonSets")
		}

		if *sec1.Privileged != *sec2.Privileged {
			t.Error("Expected consistent Privileged setting across DaemonSets")
		}

		// Verify seccomp profiles are consistent
		if sec1.SeccompProfile.Type != sec2.SeccompProfile.Type {
			t.Error("Expected consistent SeccompProfile type across DaemonSets")
		}
	})

	t.Run("Resource requirements consistency", func(t *testing.T) {
		res1 := ds1.Spec.Template.Spec.Containers[0].Resources
		res2 := ds2.Spec.Template.Spec.Containers[0].Resources

		cpuReq1 := res1.Requests[core.ResourceCPU]
		cpuReq2 := res2.Requests[core.ResourceCPU]
		if cpuReq1.Cmp(cpuReq2) != 0 {
			t.Error("Expected consistent CPU requests across DaemonSets")
		}

		memLimit1 := res1.Limits[core.ResourceMemory]
		memLimit2 := res2.Limits[core.ResourceMemory]
		if memLimit1.Cmp(memLimit2) != 0 {
			t.Error("Expected consistent Memory limits across DaemonSets")
		}
	})
}

func TestCreateOrReuseDaemonSet(t *testing.T) {
	ctx := context.Background()
	namespace := "test-namespace"
	appName := "test-app"

	t.Run("Create new DaemonSet successfully", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		config := &Config{
			Namespace: namespace,
			AppName:   appName,
			Image:     "test-image:latest",
			Reuse:     false,
		}

		err := CreateOrReuseDaemonSet(ctx, clientset, config, false)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify DaemonSet was created
		ds, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, appName, meta.GetOptions{})
		if err != nil {
			t.Errorf("Expected DaemonSet to be created, got error: %v", err)
		}
		if ds.Name != appName {
			t.Errorf("Expected DaemonSet name %s, got %s", appName, ds.Name)
		}
	})

	t.Run("Reuse existing DaemonSet - no error", func(t *testing.T) {
		existingDS := CreateDaemonSetSpec(namespace, appName, "existing-image")
		clientset := fake.NewSimpleClientset(existingDS)

		config := &Config{
			Namespace: namespace,
			AppName:   appName,
			Image:     "new-image:latest",
			Reuse:     true,
		}

		// When reuse=true, function should return without error regardless of DaemonSet existence
		err := CreateOrReuseDaemonSet(ctx, clientset, config, true)
		if err != nil {
			t.Errorf("Expected no error when reusing, got: %v", err)
		}
	})

	t.Run("DaemonSet lifecycle with reactor", func(t *testing.T) {
		// Create a client with a reactor that simulates "already exists" error
		clientset := fake.NewSimpleClientset()

		// Add reactor to simulate AlreadyExists error
		created := false
		clientset.PrependReactor("create", "daemonsets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			if created {
				return true, nil, errors.NewAlreadyExists(core.Resource("daemonsets"), appName)
			}
			created = true
			return false, nil, nil
		})

		config := &Config{
			Namespace: namespace,
			AppName:   appName,
			Image:     "new-image:latest",
			Reuse:     false,
		}

		// First creation should succeed
		err := CreateOrReuseDaemonSet(ctx, clientset, config, false)
		if err != nil {
			t.Errorf("Expected first creation to succeed, got: %v", err)
		}

		// Second creation should detect existing and try to delete
		err = CreateOrReuseDaemonSet(ctx, clientset, config, false)
		if err == nil {
			t.Error("Expected error when DaemonSet already exists")
		}
	})
}

func TestWaitForDaemonSetReady(t *testing.T) {
	ctx := context.Background()
	namespace := "test-namespace"
	appName := "test-app"

	t.Run("DaemonSet becomes ready immediately", func(t *testing.T) {
		ds := CreateDaemonSetSpec(namespace, appName, "test-image")
		ds.Status.NumberReady = 3

		// Need to create the DaemonSet in the fake client first
		clientset := fake.NewSimpleClientset()
		_, err := clientset.AppsV1().DaemonSets(namespace).Create(ctx, ds, meta.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create test DaemonSet: %v", err)
		}

		// Update status separately as fake client doesn't automatically update status
		ds.Status.NumberReady = 3
		_, err = clientset.AppsV1().DaemonSets(namespace).UpdateStatus(ctx, ds, meta.UpdateOptions{})
		if err != nil {
			t.Fatalf("Failed to update status: %v", err)
		}

		err = WaitForDaemonSetReady(ctx, clientset, namespace, appName)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("DaemonSet not found", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()

		err := WaitForDaemonSetReady(ctx, clientset, namespace, "nonexistent")
		if err == nil {
			t.Error("Expected error when DaemonSet not found")
		}
	})

	t.Run("Wait function structure", func(t *testing.T) {
		// Test that the function exists and has correct signature
		// Full async behavior testing is difficult with fake client
		ds := CreateDaemonSetSpec(namespace, appName, "test-image")
		ds.Status.NumberReady = 1

		clientset := fake.NewSimpleClientset()
		clientset.AppsV1().DaemonSets(namespace).Create(ctx, ds, meta.CreateOptions{})
		ds.Status.NumberReady = 1
		clientset.AppsV1().DaemonSets(namespace).UpdateStatus(ctx, ds, meta.UpdateOptions{})

		// Should return immediately since NumberReady > 0
		err := WaitForDaemonSetReady(ctx, clientset, namespace, appName)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func TestGetOverlayTestPods(t *testing.T) {
	ctx := context.Background()
	namespace := "test-namespace"

	t.Run("Get pods successfully", func(t *testing.T) {
		pod1 := &core.Pod{
			ObjectMeta: meta.ObjectMeta{
				Name:      "overlaytest-pod1",
				Namespace: namespace,
				Labels:    map[string]string{"app": "overlaytest"},
			},
		}
		pod2 := &core.Pod{
			ObjectMeta: meta.ObjectMeta{
				Name:      "overlaytest-pod2",
				Namespace: namespace,
				Labels:    map[string]string{"app": "overlaytest"},
			},
		}

		clientset := fake.NewSimpleClientset(pod1, pod2)

		pods, err := GetOverlayTestPods(ctx, clientset, namespace)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(pods.Items) != 2 {
			t.Errorf("Expected 2 pods, got %d", len(pods.Items))
		}
	})

	t.Run("No pods found", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()

		pods, err := GetOverlayTestPods(ctx, clientset, namespace)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(pods.Items) != 0 {
			t.Errorf("Expected 0 pods, got %d", len(pods.Items))
		}
	})

	t.Run("Only labeled pods returned", func(t *testing.T) {
		overlayPod := &core.Pod{
			ObjectMeta: meta.ObjectMeta{
				Name:      "overlaytest-pod",
				Namespace: namespace,
				Labels:    map[string]string{"app": "overlaytest"},
			},
		}
		otherPod := &core.Pod{
			ObjectMeta: meta.ObjectMeta{
				Name:      "other-pod",
				Namespace: namespace,
				Labels:    map[string]string{"app": "other"},
			},
		}

		clientset := fake.NewSimpleClientset(overlayPod, otherPod)

		pods, err := GetOverlayTestPods(ctx, clientset, namespace)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(pods.Items) != 1 {
			t.Errorf("Expected 1 pod with overlaytest label, got %d", len(pods.Items))
		}
	})
}

func TestWaitForPodNetwork(t *testing.T) {
	ctx := context.Background()
	namespace := "test-namespace"

	t.Run("Pods with valid IPs", func(t *testing.T) {
		pod := core.Pod{
			ObjectMeta: meta.ObjectMeta{
				Name:      "test-pod",
				Namespace: namespace,
			},
			Status: core.PodStatus{
				PodIP: "10.244.0.1",
			},
		}

		clientset := fake.NewSimpleClientset(&pod)

		err := WaitForPodNetwork(ctx, clientset, namespace, []core.Pod{pod})
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("Empty pod list", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()

		err := WaitForPodNetwork(ctx, clientset, namespace, []core.Pod{})
		if err != nil {
			t.Errorf("Expected no error for empty pod list, got: %v", err)
		}
	})

	t.Run("Pod not found", func(t *testing.T) {
		pod := core.Pod{
			ObjectMeta: meta.ObjectMeta{
				Name:      "nonexistent-pod",
				Namespace: namespace,
			},
		}

		clientset := fake.NewSimpleClientset()

		err := WaitForPodNetwork(ctx, clientset, namespace, []core.Pod{pod})
		if err == nil {
			t.Error("Expected error when pod not found")
		}
	})
}

func TestDaemonSetEdgeCases(t *testing.T) {
	t.Run("CreateDaemonSetSpec with empty parameters", func(t *testing.T) {
		ds := CreateDaemonSetSpec("", "", "")

		if ds.ObjectMeta.Name != "" {
			t.Error("Expected empty name")
		}

		container := ds.Spec.Template.Spec.Containers[0]
		if container.Image != "" {
			t.Error("Expected empty image")
		}
	})

	t.Run("CreateDaemonSetSpec with special characters", func(t *testing.T) {
		namespace := "test-ns-!@#"
		app := "app-$%^"
		image := "registry.io/image:tag-123"

		ds := CreateDaemonSetSpec(namespace, app, image)

		if ds.ObjectMeta.Name != app {
			t.Errorf("Expected name to preserve special characters, got %s", ds.ObjectMeta.Name)
		}

		if ds.Spec.Template.Spec.Containers[0].Image != image {
			t.Error("Expected image to preserve registry and tag")
		}
	})

	t.Run("CreateDaemonSetSpec with very long names", func(t *testing.T) {
		longName := "very-long-daemonset-name-that-exceeds-typical-kubernetes-naming-conventions-and-limits"
		ds := CreateDaemonSetSpec("default", longName, "image")

		// Kubernetes should handle validation, but we ensure it's set
		if ds.ObjectMeta.Name != longName {
			t.Error("Expected long name to be preserved")
		}
	})
}
