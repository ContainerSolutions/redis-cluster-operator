package v1alpha1

import (
	"bytes"
	"k8s.io/apimachinery/pkg/util/yaml"
	"testing"
)

func TestRedisCluster_NodesNeeded(t *testing.T) {
	testMap := map[string]struct {
		cluster       RedisCluster
		expectedNodes int32
	}{
		"3M0R": {
			cluster: RedisCluster{
				Spec: RedisClusterSpec{
					Masters:           3,
					ReplicasPerMaster: 0,
				},
			},
			expectedNodes: 3,
		},
		"3M1R": {
			cluster: RedisCluster{
				Spec: RedisClusterSpec{
					Masters:           3,
					ReplicasPerMaster: 1,
				},
			},
			expectedNodes: 6,
		},
		"3M2R": {
			cluster: RedisCluster{
				Spec: RedisClusterSpec{
					Masters:           3,
					ReplicasPerMaster: 2,
				},
			},
			expectedNodes: 9,
		},
		"5M0R": {
			cluster: RedisCluster{
				Spec: RedisClusterSpec{
					Masters:           5,
					ReplicasPerMaster: 0,
				},
			},
			expectedNodes: 5,
		},
		"5M1R": {
			cluster: RedisCluster{
				Spec: RedisClusterSpec{
					Masters:           5,
					ReplicasPerMaster: 1,
				},
			},
			expectedNodes: 10,
		},
		"5M2R": {
			cluster: RedisCluster{
				Spec: RedisClusterSpec{
					Masters:           5,
					ReplicasPerMaster: 2,
				},
			},
			expectedNodes: 15,
		},
	}
	for name, testCase := range testMap {
		got := testCase.cluster.NodesNeeded()
		if got != testCase.expectedNodes {
			t.Fatalf("Inccorect amount of nodes received to fullfill cluster. Expected %d, got %d for testcase %s", testCase.expectedNodes, got, name)
		}
	}
}

func TestRedisConfigIsProcessedCorrectly(t *testing.T) {
	redisClusterYaml := `---
apiVersion: cache.container-solutions.com/v1alpha1
kind: RedisCluster
metadata:
  name: redis-cluster
spec:
  masters: 3
  replicasPerMaster: 0
  config: |
    maxmemory 128mb
    port 6379
`
	redisCluster := &RedisCluster{}
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(redisClusterYaml)), 1000)
	err := decoder.Decode(&redisCluster)
	if err != nil {
		t.Fatalf("Failed to load yaml into RedisCluster type")
	}
	expectedConfig := `maxmemory 128mb
port 6379
`
	if redisCluster.Spec.Config != expectedConfig {
		t.Fatalf(`RedisCluster is not processing config correctly.
# Expected:
%v
# Got: 
%v`, expectedConfig, redisCluster.Spec.Config)
	}
}
