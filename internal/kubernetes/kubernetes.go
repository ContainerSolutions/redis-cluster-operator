package kubernetes

import (
	"context"
	"github.com/containersolutions/redis-cluster-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FetchExistingStatefulSet(ctx context.Context, client client.Client, cluster v1alpha1.RedisCluster) (*v1.StatefulSet, error) {
	statefulset := &v1.StatefulSet{}
	err := client.Get(ctx, types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}, statefulset)
	return statefulset, err
}
