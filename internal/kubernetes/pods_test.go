package kubernetes

import (
	"context"
	cachev1alpha1 "github.com/containersolutions/redis-cluster-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// region FetchRedisPods
func TestFetchRedisPodsFetchesAllRedisPods(t *testing.T) {
	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.GroupVersion)
	clientBuilder := fake.NewClientBuilder()
	clientBuilder.WithObjects(cluster)
	clientBuilder.WithObjects(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-123",
			Namespace: "default",
			Labels:    GetPodLabels(cluster),
		},
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-456",
			Namespace: "default",
			Labels:    GetPodLabels(cluster),
		},
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-789",
			Namespace: "default",
			Labels:    GetPodLabels(cluster),
		},
	})
	client := clientBuilder.Build()
	pods, err := FetchRedisPods(context.TODO(), client, cluster)
	if err != nil {
		t.Fatalf("Received error while trying to getch pods %v", err)
	}
	if len(pods.Items) != 3 {
		t.Fatalf("Received wrong amount of pods. Expected 3, Got %d", len(pods.Items))
	}
}

func TestFetchRedisPodsFetchesAllRedisPodsExcludingOtherPods(t *testing.T) {
	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.GroupVersion)
	clientBuilder := fake.NewClientBuilder()
	clientBuilder.WithObjects(cluster)
	clientBuilder.WithObjects(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-123",
			Namespace: "default",
			Labels:    GetPodLabels(cluster),
		},
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-456",
			Namespace: "foobar",
			Labels:    GetPodLabels(cluster),
		},
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-789",
			Namespace: "default",
			Labels: labels.Set{
				RedisNodeNameStatefulsetLabel: cluster.Name + "-foo",
				RedisNodeComponentLabel:       "redis",
			},
		},
	})
	client := clientBuilder.Build()
	pods, err := FetchRedisPods(context.TODO(), client, cluster)
	if err != nil {
		t.Fatalf("Received error while trying to getch pods %v", err)
	}
	if len(pods.Items) != 1 {
		t.Fatalf("Received wrong amount of pods. Expected 1, Got %d", len(pods.Items))
	}
	if pods.Items[0].Name != "redis-cluster-123" {
		t.Fatalf("Received wrong pods. Expected redis-cluster-123, Got %s", pods.Items[0].Name)
	}
}

// endregion
