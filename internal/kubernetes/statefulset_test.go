package kubernetes

import (
	"context"
	cachev1alpha1 "github.com/containersolutions/redis-cluster-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	_, err := FetchExistingStatefulset(context.TODO(), client, cluster)
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

	statefulset, err := FetchExistingStatefulset(context.TODO(), client, cluster)
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
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}, &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-foobar",
			Namespace: "default",
		},
	}, &v1.StatefulSet{
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

	statefulset, err := FetchExistingStatefulset(context.TODO(), client, cluster)
	if err != nil {
		t.Fatalf("Expected Statefulset to be found, but received an error %v", err)
	}
	if statefulset.Name != "redis-cluster" || statefulset.Namespace != "default" {
		t.Fatalf("Expected correct Statefulset to be found, but received an unexpected one Name: %s Namespace: %s", statefulset.Name, statefulset.Namespace)
	}
}

func TestCreateStatefulset_CanCreateStatefulset(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.GroupVersion)
	clientBuilder := fake.NewClientBuilder()
	client := clientBuilder.Build()

	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}

	_, err := CreateStatefulset(context.TODO(), client, cluster)
	if err != nil {
		t.Fatalf("Expected Statefulset to be created sucessfully, but received an error %v", err)
	}

	statefulset := &v1.StatefulSet{}
	err = client.Get(context.TODO(), types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}, statefulset)
	if err != nil {
		if errors.IsNotFound(err) {
			t.Fatalf("Expected statefulset to be in client, but it does not exist")
		} else {
			t.Fatalf("Got an error while trying to fetch the created statefulset")
		}
	}
}

func TestCreateStatefulset_ThrowsErrorIfStatefulsetAlreadyExists(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.GroupVersion)
	clientBuilder := fake.NewClientBuilder()

	clientBuilder.WithObjects(&v1.StatefulSet{
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

	_, err := CreateStatefulset(context.TODO(), client, cluster)
	if err == nil {
		t.Fatalf("Expected an error while trying to create Statefulset but didn't receive one")
	}
}

func TestCreateStatefulset_MountsConfigMapAsVolumeCorrectly(t *testing.T) {
	// Register operator types with the runtime scheme.
	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}
	statefulset := createStatefulsetSpec(cluster)

	configVolume := v13.Volume{}
	mounted := false
	for _, volume := range statefulset.Spec.Template.Spec.Volumes {
		if volume.Name == "redis-cluster-config" {
			configVolume = volume
			mounted = true
			break
		}
	}
	if !mounted {
		// The volume could not be found
		t.Fatalf("No configMap mounted into redis pods")
	}
	if configVolume.VolumeSource.ConfigMap.LocalObjectReference.Name != "redis-cluster-config" {
		t.Fatalf("Configmap mounted incorrectly to Redis pods")
	}

	redisContainer := v13.Container{}
	for _, container := range statefulset.Spec.Template.Spec.Containers {
		if container.Name == "redis" {
			redisContainer = container
		}
	}
	configMount := v13.VolumeMount{}
	mounted = false
	for _, mount := range redisContainer.VolumeMounts {
		if mount.Name == "redis-cluster-config" {
			configMount = mount
			mounted = true
			break
		}
	}
	if !mounted {
		// The volume could not be found
		t.Fatalf("Configmap not mounted into redis pods")
	}
	if configMount.MountPath != "/usr/local/etc/redis" {
		t.Fatalf("Configmap mounted on wrong directory in redis pod")
	}
}
