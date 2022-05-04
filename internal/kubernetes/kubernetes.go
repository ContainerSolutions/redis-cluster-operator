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
	RedisNodeComponentLabel = "cache.container-solutions.com/cluster-component"
)

func GetStatefulSetLabels(cluster v1alpha1.RedisCluster) labels.Set {
	return labels.Set{
		RedisNodeNameStatefulsetLabel: cluster.Name,
	}
}

func GetPodLabels(cluster v1alpha1.RedisCluster) labels.Set {
	return labels.Set{
		RedisNodeNameStatefulsetLabel: cluster.Name,
		RedisNodeComponentLabel: "redis",
	}
}

func FetchExistingStatefulSet(ctx context.Context, client client.Client, cluster v1alpha1.RedisCluster) (*v1.StatefulSet, error) {
	statefulset := &v1.StatefulSet{}
	err := client.Get(ctx, types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}, statefulset)
	return statefulset, err
}

func CreateStatefulSet(ctx context.Context, client client.Client, cluster v1alpha1.RedisCluster) (*v1.StatefulSet, error) {
	replicasNeeded := cluster.NodesNeeded()
	statefulset := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       cluster.Name,
			Namespace:               cluster.Namespace,
			Labels: GetStatefulSetLabels(cluster),
		},
		Spec:       v1.StatefulSetSpec{
			Replicas:                             &replicasNeeded,
			Selector:                             &metav1.LabelSelector{
				MatchLabels:      GetPodLabels(cluster),
			},
			Template:                             v12.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:                     GetPodLabels(cluster),
				},
				Spec: v12.PodSpec{
					Containers: []v12.Container{
						{
							Name:            "redis",
							Image:           "",  // TODO
							ImagePullPolicy: "",  // TODO
							Command:         nil, // TODO
							Args:            nil, // TODO
							WorkingDir:      "",  // TODO
							Ports:           nil, // TODO
							EnvFrom:         nil, // TODO
							Env:             nil, // TODO
							LivenessProbe:   nil, // TODO
							ReadinessProbe:  nil, // TODO
							StartupProbe:    nil, // TODO
						},
					},
				},
			},
			ServiceName:                          cluster.Name,
			MinReadySeconds:                      10,
		},
	}
	err := client.Get(ctx, types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}, statefulset)
	return statefulset, err
}
