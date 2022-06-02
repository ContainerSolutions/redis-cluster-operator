package kubernetes

import (
	"context"
	"github.com/containersolutions/redis-cluster-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FetchExistingService(ctx context.Context, kubeClient client.Client, cluster *v1alpha1.RedisCluster) (*v1.Service, error) {
	service := &v1.Service{}
	err := kubeClient.Get(ctx, types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}, service)
	return service, err
}

func createServiceSpec(cluster *v1alpha1.RedisCluster) *v1.Service {
	service := &v1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "redis",
					Port:       6379,
					TargetPort: intstr.FromInt(6379),
				},
			},
			Selector: GetPodLabels(cluster),
			Type:     "ClusterIP",
		},
	}
	return service
}

func CreateService(ctx context.Context, kubeClient client.Client, cluster *v1alpha1.RedisCluster) (*v1.Service, error) {
	service := createServiceSpec(cluster)
	err := kubeClient.Create(ctx, service)
	return service, err
}
