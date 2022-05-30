package utils

import (
	v1 "k8s.io/api/core/v1"
	"sort"
	"strings"
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
			Name:      "foobar",
			MountPath: "/etc/foo",
		},
	}
	overrideVolumeMounts := []v1.VolumeMount{
		{
			Name:      "foobar",
			MountPath: "/etc/foobar",
		},
		{
			Name:      "new-foo",
			MountPath: "/data",
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

func TestMergeContainers(t *testing.T) {
	// We want to ensure we can add containers, as well as override containers
	originalContainers := []v1.Container{
		{
			Name:  "redis",
			Image: "redis:7",
			Ports: []v1.ContainerPort{
				{
					Name:          "redis",
					ContainerPort: 6379,
				},
			},
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      "redis-cluster-config",
					MountPath: "/usr/local/etc/redis",
				},
			},
		},
	}
	overrideContainers := []v1.Container{
		{
			Name:  "redis",
			Image: "redis:5",
			Ports: []v1.ContainerPort{
				{
					// override one port
					Name:          "redis",
					ContainerPort: 7001,
				},
				{
					// add another port
					Name:          "metrics",
					ContainerPort: 8080,
				},
			},
			VolumeMounts: []v1.VolumeMount{
				{
					// override volume mount
					Name:      "redis-cluster-config",
					MountPath: "/usr/local/etc/foobar",
				},
				{
					// add volume mount
					Name:      "redis-cluster-config-metrics",
					MountPath: "/usr/local/etc/metrics",
				},
			},
		},
		{
			Name: "prometheus-metrics",
			Image: "prometheus-redis:1.0.0",
		},
	}

	merged := MergeContainers(originalContainers, overrideContainers)
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Name < merged[j].Name
	})

	// Assert that the redis container pieces are overridden
	redisContainer := merged[1]
	if redisContainer.Image != "redis:5" {
		t.Fatalf("Redis image not correctly overriden")
	}

	redisPorts := redisContainer.Ports
	sort.SliceStable(redisPorts, func(i, j int) bool {
		return strings.ToLower(merged[i].Name) > strings.ToLower(merged[j].Name)
	})
	if redisPorts[1].ContainerPort != 7001 {
		t.Fatalf("Redis port not correctly overridden")
	}
	if redisPorts[0].ContainerPort != 8080 || redisPorts[0].Name != "metrics" {
		t.Fatalf("Metrics port not correctly added")
	}

	redisVolumeMounts := redisContainer.VolumeMounts
	sort.SliceStable(redisVolumeMounts, func(i, j int) bool {
		return strings.ToLower(merged[i].Name) > strings.ToLower(merged[j].Name)
	})
	if redisVolumeMounts[0].Name != "redis-cluster-config-metrics" || redisVolumeMounts[0].MountPath != "/usr/local/etc/metrics" {
		t.Fatalf("Metrics volume mount not correctly added")
	}
	if redisVolumeMounts[1].MountPath != "/usr/local/etc/foobar" {
		t.Fatalf("Config volume mount not correctly overridden")
	}

	metricsContainer := merged[0]
	if metricsContainer.Name != "prometheus-metrics" || metricsContainer.Image != "prometheus-redis:1.0.0" {
		t.Fatalf("Metrics container not added correctly")
	}
}
