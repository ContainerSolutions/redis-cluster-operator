package kubernetes

import (
	"context"
	cachev1alpha1 "github.com/containersolutions/redis-cluster-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// region FetchExistingConfigMap
func TestFindExistingService(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)
	clientBuilder := fake.NewClientBuilder()

	clientBuilder.WithObjects(&v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	})

	client := clientBuilder.Build()

	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	service, err := FetchExistingService(context.TODO(), client, cluster)
	if err != nil {
		t.Fatalf("Expected Service to be found, but received an error %v", err)
	}
	if service.Name != "redis-cluster" || service.Namespace != "default" {
		t.Fatalf("Expected correct Service to be found, but received an unexpected one Name: %s Namespace: %s", service.Name, service.Namespace)
	}
}

func TestFindExistingServiceFetchesCorrectService(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)
	clientBuilder := fake.NewClientBuilder()

	clientBuilder.WithObjects(&v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}, &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-foo",
			Namespace: "default",
		},
	}, &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "foobar",
		},
	})

	client := clientBuilder.Build()

	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	service, err := FetchExistingService(context.TODO(), client, cluster)
	if err != nil {
		t.Fatalf("Expected Service to be found, but received an error %v", err)
	}
	if service.Name != "redis-cluster" || service.Namespace != "default" {
		t.Fatalf("Expected correct Service to be found, but received an unexpected one Name: %s Namespace: %s", service.Name, service.Namespace)
	}
}

func TestFindExistingServiceReturnsNotFoundErrorIfNotExists(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)
	clientBuilder := fake.NewClientBuilder()
	client := clientBuilder.Build()

	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	_, err := FetchExistingService(context.TODO(), client, cluster)
	if err == nil {
		t.Fatalf("Expected not found error but did not receive any error")
	}
	if !errors.IsNotFound(err) {
		t.Fatalf("Expected not found error but received unexpected error %v", err)
	}
}
// endregion

func TestCreateServiceSpec(t *testing.T) {
	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "redis-cluster",
			Namespace: "default",
		},
	}
	service := createServiceSpec(cluster)
	if service.Spec.Ports[0].Port != 6379 && service.Spec.Ports[0].TargetPort != intstr.FromInt(6379) {
		t.Fatalf("Redis port is not active on the service")
	}
	if !reflect.DeepEqual(labels.Set(service.Spec.Selector), GetPodLabels(cluster)) {
		t.Fatalf("Service selector does not match pods labels")
	}
}