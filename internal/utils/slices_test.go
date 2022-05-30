package utils

import (
	v1 "k8s.io/api/core/v1"
	"sort"
	"testing"
)

func TestMergeContainerPorts(t *testing.T) {
	originalPorts := []v1.ContainerPort{
		{
			Name:          "foobar",
			HostPort:      6379,
			ContainerPort: 6379,
			Protocol:      v1.ProtocolTCP,
		},
	}
	overrides := []v1.ContainerPort{
		{
			Name:     "foobar",
			HostPort: 7001,
			Protocol: v1.ProtocolUDP,
			HostIP:   "10.10.10.10",
		},
		{
			Name:     "new-foo",
			HostPort: 8080,
		},
	}
	merged := MergeContainerPorts(originalPorts, overrides)
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Name < merged[j].Name
	})
	// Assert correct overriding of values
	if merged[0].HostPort != 7001 {
		t.Fatalf("Host port was not overridden correctly")
	}
	if merged[0].Protocol != v1.ProtocolUDP {
		t.Fatalf("Protocol was not overridden correctly")
	}
	// Assert original field stay the same
	if merged[0].ContainerPort != 6379 {
		t.Fatalf("ContainerPort was changed unexpectedly")
	}
	// Assert new field are added correctly
	if merged[0].HostIP != "10.10.10.10" {
		t.Fatalf("HostIP was not added")
	}

	// Assert new port added
	if merged[1].Name != "new-foo" || merged[1].HostPort != 8080 {
		t.Fatalf("New port was not added")
	}
}

func TestMergeVolumeMounts(t *testing.T) {
	originalVolumeMounts := []v1.VolumeMount{
		{
			Name:             "foobar",
			MountPath:        "/etc/foo",
		},
	}
	overrideVolumeMounts :=  []v1.VolumeMount{
		{
			Name:             "foobar",
			MountPath:        "/etc/foobar",
		},
		{
			Name:             "new-foo",
			MountPath:        "/data",
		},
	}
	merged := MergeVolumeMounts(originalVolumeMounts, overrideVolumeMounts)
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Name < merged[j].Name
	})
	if merged[0].MountPath != "/etc/foobar" {
		t.Fatalf("MountPath not overridden correctly")
	}
	if merged[1].MountPath != "/data" || merged[1].Name != "new-foo" {
		t.Fatalf("New Volume Mount not added correctly")
	}
}
