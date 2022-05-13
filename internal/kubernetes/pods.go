package kubernetes

import (
	"context"
	"github.com/containersolutions/redis-cluster-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FetchRedisPods(ctx context.Context, kubeClient client.Client, cluster *v1alpha1.RedisCluster) (*v1.PodList, error) {
	pods := &v1.PodList{}
	err := kubeClient.List(
		ctx,
		pods,
		client.MatchingLabelsSelector{Selector: GetPodLabels(cluster).AsSelector()},
		client.InNamespace(cluster.Namespace),
	)
	return pods, err
}
