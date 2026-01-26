package overlaytest

import (
	"testing"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
