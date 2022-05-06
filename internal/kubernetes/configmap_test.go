package kubernetes

import (
	"context"
	cachev1alpha1 "github.com/containersolutions/redis-cluster-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"
)

//region getDefaultRedisConfig
func TestGetDefaultRedisConfig(t *testing.T) {
	defaultConfig := getDefaultRedisConfig()

	// Cluster needs to be enabled
	if defaultConfig["cluster-enabled"] != "yes" {
		t.Fatalf("The default redis config does not enable cluster mode")
	}
	// Port needs to be set to 6379
	if defaultConfig["port"] != "6379" {
		t.Fatalf("The default redis config port is not 6379")
	}
}

//endregion

// region FindExistingConfigMap
func TestFindExistingConfigMapFetchesConfigMap(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)
	clientBuilder := fake.NewClientBuilder()

	clientBuilder.WithObjects(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-config",
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

	configMap, err := FindExistingConfigMap(context.TODO(), client, cluster)
	if err != nil {
		t.Fatalf("Expected ConfigMap to be found, but received an error %v", err)
	}
	if configMap.Name != "redis-cluster-config" || configMap.Namespace != "default" {
		t.Fatalf("Expected correct ConfigMap to be found, but received an unexpected one Name: %s Namespace: %s", configMap.Name, configMap.Namespace)
	}
}

func TestFindExistingConfigMapFetchesCorrectConfigMap(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)
	clientBuilder := fake.NewClientBuilder()

	clientBuilder.WithObjects(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-config",
			Namespace: "default",
		},
	}, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-foo-config",
			Namespace: "default",
		},
	}, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster-config",
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

	configMap, err := FindExistingConfigMap(context.TODO(), client, cluster)
	if err != nil {
		t.Fatalf("Expected ConfigMap to be found, but received an error %v", err)
	}
	if configMap.Name != "redis-cluster-config" || configMap.Namespace != "default" {
		t.Fatalf("Expected correct ConfigMap to be found, but received an unexpected one Name: %s Namespace: %s", configMap.Name, configMap.Namespace)
	}
}

func TestFindExistingConfigMapReturnsNotFoundErrorIfNotExists(t *testing.T) {
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

	_, err := FindExistingConfigMap(context.TODO(), client, cluster)
	if err == nil {
		t.Fatalf("Expected not found error but did not receive any error")
	}
	if !errors.IsNotFound(err) {
		t.Fatalf("Expected not found error but received unexpected error %v", err)
	}
}

// endregion

//region getRedisConfigAsMultilineYaml
func TestGetRedisConfigAsMultilineYaml(t *testing.T) {
	got := getRedisConfigAsMultilineYaml(map[string]string{
		"cluster-enabled": "true",
		"port":            "6379",
		"maxmemory":       "100mb",
	})
	expected := `port 6379
maxmemory 100mb
cluster-enabled true
`
	// We want to test that all the lines are contained from expected to got result.
	// The order of the string elements i s not guaranteed, so we do it lines by line
	if len(strings.Split(got, "\n")) != len(strings.Split(expected, "\n")) {
		t.Fatalf(`The amount of lines in the multiline string does not match what we expected. Expected %d Got %d`, len(strings.Split(expected, "\n")), len(strings.Split(got, "\n")))
	}
	for _, expectedLine := range strings.Split(expected, "\n") {
		if !strings.Contains(got, expectedLine) {
			t.Fatalf("setting not found in multiline. Expected %s to be in %s", expectedLine, got)
		}
	}
}

//endregion

//region getAppliedRedisConfig
func TestGetAppliedRedisConfigProcessesAdditionalConfigPassedIntoRedisClusterWithDefault(t *testing.T) {
	redisConfig := `
maxmemory 128mb
maxmemory-samples 5
`
	redisCluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           3,
			ReplicasPerMaster: 1,
			Config:            redisConfig,
		},
	}
	gotAppliedConfig := getAppliedRedisConfig(redisCluster)
	expectedAppliedConfig := map[string]string{
		"maxmemory":            "128mb",
		"maxmemory-samples":    "5",
		"cluster-config-file":  "nodes.conf",
		"cluster-enabled":      "yes",
		"cluster-node-timeout": "5000",
		"port":                 "6379",
	}
	if !reflect.DeepEqual(gotAppliedConfig, expectedAppliedConfig) {
		t.Fatalf(`Applied configuration is not what we expect it to be based on inputs.
Expected: %v
Got: %v
`, expectedAppliedConfig, gotAppliedConfig)
	}
}

//endregion

//region getRedisConfigFromMultilineYaml
func TestGetRedisConfigFromMultilineYaml(t *testing.T) {
	redisConfigString := `maxmemory 128mb
maxmemory-samples 5
port 7000
`
	gotRedisConfig := getRedisConfigFromMultilineYaml(redisConfigString)
	expectedRedisConfig := map[string]string{
		"maxmemory":         "128mb",
		"maxmemory-samples": "5",
		"port":              "7000",
	}
	if !reflect.DeepEqual(gotRedisConfig, expectedRedisConfig) {
		t.Fatalf(`Expected String config to be processed correctly. 
Expected %v
Got %v`, expectedRedisConfig, gotRedisConfig)
	}
}

//endregion

//region createConfigMapSpec
func TestCreateConfigMapSpecShouldHaveRedisConfKey(t *testing.T) {
	cluster := &cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
	}
	configMap := createConfigMapSpec(cluster)
	if configMap.Name != "redis-cluster-config" || configMap.Namespace != "default" {
		t.Fatalf("ConfigMap generated with incorrect name or namespace. Name %s Namespace %s", configMap.Name, configMap.Namespace)
	}
	if _, ok := configMap.Data["redis.conf"]; !ok {
		t.Fatalf("The redis.conf key does not exist on the generated configmap")
	}
}

//endregion

//region CreateConfigMap
func TestCreateConfigMap(t *testing.T) {
	redisCluster := cachev1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: cachev1alpha1.RedisClusterSpec{
			Masters:           3,
			ReplicasPerMaster: 1,
		},
	}
	s := scheme.Scheme
	_ = cachev1alpha1.AddToScheme(s)
	clientBuilder := fake.NewClientBuilder()
	clientBuilder.WithObjects(&redisCluster)
	client := clientBuilder.Build()
	_, err := CreateConfigMap(context.TODO(), client, &redisCluster)
	if err != nil {
		t.Fatalf("Received an error while trying to create Redis configmap")
	}

	// Assert that the kubeClient contains the new configMap
	configMap := &v1.ConfigMap{}
	err = client.Get(context.TODO(), types.NamespacedName{
		Namespace: redisCluster.Namespace,
		Name:      getConfigMapName(&redisCluster),
	}, configMap)
	if err != nil {
		t.Fatalf("Received an error while trying to assert created configmap %v", err)
	}
}

//endregion
