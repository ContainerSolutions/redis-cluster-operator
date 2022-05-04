package kubernetes

import (
	"context"
	"github.com/containersolutions/redis-cluster-operator/api/v1alpha1"
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

func GetStatefulSetLabels(cluster v1alpha1.RedisCluster) labels.Set {
	return labels.Set{
		RedisNodeNameStatefulsetLabel: cluster.Name,
	}
}

func GetPodLabels(cluster v1alpha1.RedisCluster) labels.Set {
	return labels.Set{
		RedisNodeNameStatefulsetLabel: cluster.Name,
		RedisNodeComponentLabel:       "redis",
	}
}

func FetchExistingStatefulset(ctx context.Context, kubeClient client.Client, cluster v1alpha1.RedisCluster) (*v1.StatefulSet, error) {
	statefulset := &v1.StatefulSet{}
	err := kubeClient.Get(ctx, types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}, statefulset)
	return statefulset, err
}

func CreateStatefulset(ctx context.Context, kubeClient client.Client, cluster v1alpha1.RedisCluster) (*v1.StatefulSet, error) {
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
			Template: v12.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: GetPodLabels(cluster),
				},
				Spec: v12.PodSpec{
					Containers: []v12.Container{
						{
							Name:  "redis",
							Image: "redis:7.0.0",
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
						},
					},
				},
			},
			ServiceName:     cluster.Name,
			MinReadySeconds: 10,
		},
	}
	err := kubeClient.Create(ctx, statefulset)
	return statefulset, err
}
