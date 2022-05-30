package kubernetes

import (
	"context"
	"github.com/containersolutions/redis-cluster-operator/api/v1alpha1"
	"github.com/containersolutions/redis-cluster-operator/internal/utils"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RedisNodeNameStatefulsetLabel = "cache.container-solutions.com/cluster-name"
	RedisNodeComponentLabel       = "cache.container-solutions.com/cluster-component"
)

func GetStatefulSetLabels(cluster *v1alpha1.RedisCluster) labels.Set {
	return labels.Set{
		RedisNodeNameStatefulsetLabel: cluster.Name,
	}
}

func GetPodLabels(cluster *v1alpha1.RedisCluster) labels.Set {
	return labels.Set{
		RedisNodeNameStatefulsetLabel: cluster.Name,
		RedisNodeComponentLabel:       "redis",
	}
}

func FetchExistingStatefulset(ctx context.Context, kubeClient client.Client, cluster *v1alpha1.RedisCluster) (*v1.StatefulSet, error) {
	statefulset := &v1.StatefulSet{}
	err := kubeClient.Get(ctx, types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}, statefulset)
	return statefulset, err
}

func createStatefulsetSpec(cluster *v1alpha1.RedisCluster) *v1.StatefulSet {
	replicasNeeded := cluster.NodesNeeded()
	statefulset := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    GetStatefulSetLabels(cluster),
		},
		Spec: v1.StatefulSetSpec{
			Replicas: &replicasNeeded,
			Selector: &metav1.LabelSelector{
				MatchLabels: GetPodLabels(cluster),
			},
			PodManagementPolicy: v1.ParallelPodManagement,
			Template: v12.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: GetPodLabels(cluster),
				},
				Spec: v12.PodSpec{
					Volumes: utils.MergeVolumes(
						[]v12.Volume{
							{
								Name: "redis-cluster-config",
								VolumeSource: v12.VolumeSource{
									ConfigMap: &v12.ConfigMapVolumeSource{
										LocalObjectReference: v12.LocalObjectReference{
											Name: getConfigMapName(cluster),
										},
									},
								},
							},
						},
						cluster.Spec.PodSpec.Volumes,
					),
					InitContainers: utils.MergeContainers(
						[]v12.Container{},
						cluster.Spec.PodSpec.InitContainers,
					),
					Containers: utils.MergeContainers(
						[]v12.Container{
							{
								Name:  "redis",
								Image: "redis:7.0.0",
								Command: []string{
									"redis-server",
								},
								Args: []string{
									"/usr/local/etc/redis/redis.conf",
								},
								Ports: []v12.ContainerPort{
									{
										Name:          "redis",
										ContainerPort: 6379,
									},
									{
										Name:          "redis-gossip",
										ContainerPort: 16379,
									},
								},
								LivenessProbe: &v12.Probe{
									ProbeHandler: v12.ProbeHandler{
										Exec: &v12.ExecAction{
											Command: []string{
												"redis-cli",
												"ping",
											},
										},
									},
									InitialDelaySeconds: 10,
									TimeoutSeconds:      5,
									PeriodSeconds:       3,
								},
								ReadinessProbe: &v12.Probe{
									ProbeHandler: v12.ProbeHandler{
										Exec: &v12.ExecAction{
											Command: []string{
												"redis-cli",
												"ping",
											},
										},
									},
									InitialDelaySeconds: 10,
									TimeoutSeconds:      5,
									PeriodSeconds:       3,
								},
								VolumeMounts: []v12.VolumeMount{
									{
										Name:      "redis-cluster-config",
										MountPath: "/usr/local/etc/redis",
									},
								},
							},
						},
						cluster.Spec.PodSpec.Containers,
					),
					EphemeralContainers:           cluster.Spec.PodSpec.EphemeralContainers,
					RestartPolicy:                 cluster.Spec.PodSpec.RestartPolicy,
					TerminationGracePeriodSeconds: cluster.Spec.PodSpec.TerminationGracePeriodSeconds,
					ActiveDeadlineSeconds:         cluster.Spec.PodSpec.ActiveDeadlineSeconds,
					DNSPolicy:                     cluster.Spec.PodSpec.DNSPolicy,
					NodeSelector:                  cluster.Spec.PodSpec.NodeSelector,
					NodeName:                      cluster.Spec.PodSpec.NodeName,
					HostNetwork:                   cluster.Spec.PodSpec.HostNetwork,
					HostPID:                       cluster.Spec.PodSpec.HostPID,
					HostIPC:                       cluster.Spec.PodSpec.HostIPC,
					ShareProcessNamespace:         cluster.Spec.PodSpec.ShareProcessNamespace,
					SecurityContext:               cluster.Spec.PodSpec.SecurityContext,
					ImagePullSecrets:              cluster.Spec.PodSpec.ImagePullSecrets,
					Hostname:                      cluster.Spec.PodSpec.Hostname,
					Subdomain:                     cluster.Spec.PodSpec.Subdomain,
					Affinity:                      cluster.Spec.PodSpec.Affinity,
					SchedulerName:                 cluster.Spec.PodSpec.SchedulerName,
					Tolerations:                   cluster.Spec.PodSpec.Tolerations,
					HostAliases:                   cluster.Spec.PodSpec.HostAliases,
					PriorityClassName:             cluster.Spec.PodSpec.PriorityClassName,
					Priority:                      cluster.Spec.PodSpec.Priority,
					DNSConfig:                     cluster.Spec.PodSpec.DNSConfig,
					ReadinessGates:                cluster.Spec.PodSpec.ReadinessGates,
					RuntimeClassName:              cluster.Spec.PodSpec.RuntimeClassName,
					EnableServiceLinks:            cluster.Spec.PodSpec.EnableServiceLinks,
					PreemptionPolicy:              cluster.Spec.PodSpec.PreemptionPolicy,
					Overhead:                      cluster.Spec.PodSpec.Overhead,
					TopologySpreadConstraints:     cluster.Spec.PodSpec.TopologySpreadConstraints,
					SetHostnameAsFQDN:             cluster.Spec.PodSpec.SetHostnameAsFQDN,
					OS:                            cluster.Spec.PodSpec.OS,
				},
			},
			ServiceName:     cluster.Name,
			MinReadySeconds: 10,
		},
	}
	return statefulset
}

func CreateStatefulset(ctx context.Context, kubeClient client.Client, cluster *v1alpha1.RedisCluster) (*v1.StatefulSet, error) {
	statefulset := createStatefulsetSpec(cluster)
	err := kubeClient.Create(ctx, statefulset)
	return statefulset, err
}
