package kubernetes

import (
	"context"
	cachev1alpha1 "github.com/containersolutions/redis-cluster-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestFetchExistingStatefulSetReturnsErrorIfNotFound(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.GroupVersion)
	clientBuilder := fake.NewClientBuilder()
	client := clientBuilder.Build()

	cluster := cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	_, err := FetchExistingStatefulSet(context.TODO(), client, cluster)
	if err == nil {
		t.Fatalf("Expected Statefulset to not be found, but received no error")
	}
	if !errors.IsNotFound(err) {
		t.Fatalf("Expected Statefulset to not be found, but received unknown error %v", err)
	}
}

func TestFetchExistingStatefulSetReturnsStatefulsetIfFound(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.GroupVersion)
	clientBuilder := fake.NewClientBuilder()

	clientBuilder.WithObjects(&v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       "redis-cluster",
			Namespace:                  "default",
		},
	})

	client := clientBuilder.Build()

	cluster := cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	statefulset, err := FetchExistingStatefulSet(context.TODO(), client, cluster)
	if err != nil {
		t.Fatalf("Expected Statefulset to be found, but received an error %v", err)
	}
	if statefulset.Name != "redis-cluster" {
		t.Fatalf("Expected correct Statefulset to be found, but received an unexpected one %s", statefulset.Name)
	}
}

func TestFetchExistingStatefulSetReturnsCorrectStatefulsetIfMany(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.GroupVersion)
	clientBuilder := fake.NewClientBuilder()

	clientBuilder.WithObjects(&v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       "redis-cluster",
			Namespace:                  "default",
		},
	}, &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       "redis-cluster-foobar",
			Namespace:                  "default",
		},
	}, &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       "redis-cluster",
			Namespace:                  "foobar",
		},
	})

	client := clientBuilder.Build()

	cluster := cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	statefulset, err := FetchExistingStatefulSet(context.TODO(), client, cluster)
	if err != nil {
		t.Fatalf("Expected Statefulset to be found, but received an error %v", err)
	}
	if statefulset.Name != "redis-cluster" || statefulset.Namespace != "default" {
		t.Fatalf("Expected correct Statefulset to be found, but received an unexpected one Name: %s Namespace: %s", statefulset.Name, statefulset.Namespace)
	}
}
