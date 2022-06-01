package redis

import (
	"context"
	"github.com/containersolutions/redis-cluster-operator/api/v1alpha1"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

// region ClusterMeet
func TestClusterMeetMeetsAllNodes(t *testing.T) {
	node1Client, node1Mock := redismock.NewClientMock()
	node1Mock.ExpectClusterNodes().SetVal(`9fd8800b31d569538917c0aaeaa5588e2f9c6edf 10.20.30.40:6379@16379 myself,master - 0 1652373716000 0 connected
8a99a71a38d099de6862284f5aab9329d796c34f 10.20.30.41:6379@16379 master - 0 1652373718026 1 connected
`)
	node1Mock.ExpectClusterMeet("10.20.30.40", "6379").SetVal("OK")
	node1Mock.ExpectClusterMeet("10.20.30.41", "6379").SetVal("OK")

	node1, err := NewNode(context.TODO(), &redis.Options{
		Addr: "10.20.30.40:6379",
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rediscluster",
			Namespace: "default",
		},
	}, func(opt *redis.Options) *redis.Client {
		return node1Client
	})
	if err != nil {
		t.Fatalf("received error while trying to create node %v", err)
	}

	node2Client, node2Mock := redismock.NewClientMock()
	node2Mock.ExpectClusterNodes().SetVal(`9fd8800b31d569538917c0aaeaa5588e2f9c6edf 10.20.30.40:6379@16379 master - 0 1652373716000 0 connected
8a99a71a38d099de6862284f5aab9329d796c34f 10.20.30.41:6379@16379 myself,master - 0 1652373718026 1 connected
`)
	node2Mock.ExpectClusterMeet("10.20.30.40", "6379").SetVal("OK")
	node2Mock.ExpectClusterMeet("10.20.30.41", "6379").SetVal("OK")
	node2, err := NewNode(context.TODO(), &redis.Options{
		Addr: "10.20.30.41:6379",
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rediscluster",
			Namespace: "default",
		},
	}, func(opt *redis.Options) *redis.Client {
		return node2Client
	})
	if err != nil {
		t.Fatalf("received error while trying to create node %v", err)
	}

	clusterNodes := ClusterNodes{
		Nodes: []*Node{
			node1,
			node2,
		},
	}
	err = clusterNodes.ClusterMeet(context.TODO())
	if err != nil {
		t.Fatalf("Receives error when trying to cluster meet %v", err)
	}
	if node1Mock.ExpectationsWereMet() != nil {
		t.Fatalf("Node 1 did not receive all the cluster meet commands it was expected.")
	}
	if node2Mock.ExpectationsWereMet() != nil {
		t.Fatalf("Node 2 did not receive all the cluster meet commands it was expected.")
	}
}

// endregion

// region GetAssignedSlots
func TestGetAssignedSlot(t *testing.T) {
	client, mock := redismock.NewClientMock()
	mock.ExpectClusterNodes().SetVal(`9fd8800b31d569538917c0aaeaa5588e2f9c6edf 10.20.30.40:6379@16379 myself,master - 0 1652373716000 0 connected 0-3 5 7-9
`)
	node, err := NewNode(context.TODO(), &redis.Options{
		Addr: "10.20.30.40:6379",
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rediscluster",
			Namespace: "default",
		},
	}, func(opt *redis.Options) *redis.Client {
		return client
	})
	if err != nil {
		t.Fatalf("received error while trying to create node %v", err)
	}
	clusterNodes := ClusterNodes{
		Nodes: []*Node{node},
	}
	expected := []int32{0, 1, 2, 3, 5, 7, 8, 9}
	if !reflect.DeepEqual(clusterNodes.GetAssignedSlots(), expected) {
		t.Fatalf("Did not get correct list of assigned slots. Expected %v, Got %v", expected, clusterNodes.GetAssignedSlots())
	}
}

// endregion

// region GetMissingSlots
func TestGetMissingSlots(t *testing.T) {
	client, mock := redismock.NewClientMock()
	mock.ExpectClusterNodes().SetVal(`9fd8800b31d569538917c0aaeaa5588e2f9c6edf 10.20.30.40:6379@16379 myself,master - 0 1652373716000 0 connected 0-10000 10005 10011-16379
`)
	node, err := NewNode(context.TODO(), &redis.Options{
		Addr: "10.20.30.40:6379",
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rediscluster",
			Namespace: "default",
		},
	}, func(opt *redis.Options) *redis.Client {
		return client
	})
	if err != nil {
		t.Fatalf("received error while trying to create node %v", err)
	}
	clusterNodes := ClusterNodes{
		Nodes: []*Node{node},
	}
	expected := []int32{10001, 10002, 10003, 10004, 10006, 10007, 10008, 10009, 10010, 16380, 16381, 16382, 16383}
	if !reflect.DeepEqual(clusterNodes.GetMissingSlots(), expected) {
		t.Fatalf("Did not get correct list of missing slots. Expected %v, Got %v", expected, clusterNodes.GetMissingSlots())
	}
}

// endregion

func TestCalculateSlotAssignmentWorksForMastersOnly(t *testing.T) {
	var nodes []*Node
	mocks := map[string]*redismock.ClientMock{}
	for i := 0; i <= 3; i++ {
		node, err := NewNode(context.TODO(), &redis.Options{
			Addr: "10.20.30.40:6379",
		}, &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rediscluster",
				Namespace: "default",
			},
		}, func(opt *redis.Options) *redis.Client {
			client, mock := redismock.NewClientMock()
			switch i {
			case 0:
				mock.ExpectClusterNodes().SetVal(`5dbeafc760e4ec355f007b2ce10c690a56306dc8 10.20.30.40:6379@16379 myself,master - 0 1653479781000 16 connected 8202-16383
4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad 10.20.30.41:6379@16379 master - 0 1653479781000 16 connected 0-8180 
0465e428668773fc3bbeb02150bbd4324e409fe0 10.20.30.42:6379@16379 slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781544 16 connected
85613000e76a00c2da80e9eae0f2fed6bc857605 10.20.30.43:6379@16379 slave 4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad 0 1653479781000 16 connected
`)
				mocks["5dbeafc760e4ec355f007b2ce10c690a56306dc8"] = &mock
			case 1:
				// Early return for this node
				mock.ExpectClusterNodes().SetVal(`5dbeafc760e4ec355f007b2ce10c690a56306dc8 10.20.30.40:6379@16379 master - 0 1653479781000 16 connected 8202-16383
4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad 10.20.30.41:6379@16379 myself,master - 0 1653479781000 16 connected 0-8180 
0465e428668773fc3bbeb02150bbd4324e409fe0 10.20.30.42:6379@16379 slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781544 16 connected
85613000e76a00c2da80e9eae0f2fed6bc857605 10.20.30.43:6379@16379 slave 4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad 0 1653479781000 16 connected
`)
				mock.ExpectClusterResetSoft().SetVal("OK")
				mocks["4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad"] = &mock
			case 2:
				mock.ExpectClusterNodes().SetVal(`5dbeafc760e4ec355f007b2ce10c690a56306dc8 10.20.30.40:6379@16379 master - 0 1653479781000 16 connected 8202-16383
4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad 10.20.30.41:6379@16379 master - 0 1653479781000 16 connected 0-8180 
0465e428668773fc3bbeb02150bbd4324e409fe0 10.20.30.42:6379@16379 myself,slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781544 16 connected
85613000e76a00c2da80e9eae0f2fed6bc857605 10.20.30.43:6379@16379 slave 4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad 0 1653479781000 16 connected
`)
				mocks["0465e428668773fc3bbeb02150bbd4324e409fe0"] = &mock
			case 3:
				mock.ExpectClusterNodes().SetVal(`5dbeafc760e4ec355f007b2ce10c690a56306dc8 10.20.30.40:6379@16379 master - 0 1653479781000 16 connected 8202-16383
4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad 10.20.30.41:6379@16379 master - 0 1653479781000 16 connected 0-8180 
0465e428668773fc3bbeb02150bbd4324e409fe0 10.20.30.42:6379@16379 slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781544 16 connected
85613000e76a00c2da80e9eae0f2fed6bc857605 10.20.30.43:6379@16379 myself,slave 4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad 0 1653479781000 16 connected
`)
				mocks["85613000e76a00c2da80e9eae0f2fed6bc857605"] = &mock
			}
			return client
		})
		if err != nil {
			t.Fatalf("Got error whil trying to create node. %v", err)
		}
		nodes = append(nodes, node)
	}

	clusterNodes := ClusterNodes{
		Nodes: nodes,
	}
	got := clusterNodes.CalculateSlotAssignment()

	var gotNodeIds []string
	var gotSlots [][]int32
	for node, slots := range got {
		gotNodeIds = append(gotNodeIds, node.NodeAttributes.ID)
		gotSlots = append(gotSlots, slots)
	}
	if !reflect.DeepEqual(gotNodeIds, []string{"5dbeafc760e4ec355f007b2ce10c690a56306dc8", "4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad"}) &&
		!reflect.DeepEqual(gotNodeIds, []string{"4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad", "5dbeafc760e4ec355f007b2ce10c690a56306dc8"}) {
		t.Fatalf("Slot Assignment Calculation did not return correct nodes. Expected %v Got %v", []string{"5dbeafc760e4ec355f007b2ce10c690a56306dc8", "4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad"}, gotNodeIds)
	}

	if !reflect.DeepEqual(gotSlots, [][]int32{{8181, 8182, 8183, 8184, 8185, 8186, 8187, 8188, 8189, 8190, 8191}, {8192, 8193, 8194, 8195, 8196, 8197, 8198, 8199, 8200, 8201}}) &&
		!reflect.DeepEqual(gotSlots, [][]int32{{8192, 8193, 8194, 8195, 8196, 8197, 8198, 8199, 8200, 8201}, {8181, 8182, 8183, 8184, 8185, 8186, 8187, 8188, 8189, 8190, 8191}}) {
		t.Fatalf("Slot Assignment Calculation did not return correct slot assignments. Expected %v Got %v", [][]int32{{8192, 8193, 8194, 8195, 8196, 8197, 8198, 8199, 8200, 8201}, {8181, 8182, 8183, 8184, 8185, 8186, 8187, 8188, 8189, 8190, 8191}}, gotSlots)
	}
}

func TestClusterNodes_GetMasters(t *testing.T) {
	var nodes []*Node
	for i := 0; i <= 1; i++ {
		node, err := NewNode(context.TODO(), &redis.Options{
			Addr: "10.20.30.40:6379",
		}, &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rediscluster",
				Namespace: "default",
			},
		}, func(opt *redis.Options) *redis.Client {
			client, mock := redismock.NewClientMock()
			if i == 0 {
				mock.ExpectClusterNodes().SetVal(`5dbeafc760e4ec355f007b2ce10c690a56306dc8 10.244.0.225:6379@16379 myself,master - 0 1653476460000 9 connected
85613000e76a00c2da80e9eae0f2fed6bc857605 10.244.0.240:6379@16379 slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653476461000 9 connected
`)
			} else {
				mock.ExpectClusterNodes().SetVal(`5dbeafc760e4ec355f007b2ce10c690a56306dc8 10.244.0.225:6379@16379 master - 0 1653476460000 9 connected
85613000e76a00c2da80e9eae0f2fed6bc857605 10.244.0.240:6379@16379 myself,slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653476461000 9 connected
`)
			}
			return client
		})
		if err != nil {
			t.Fatalf("Got error while creating nodes. %v", err)
		}
		nodes = append(nodes, node)
	}

	clusterNodes := ClusterNodes{
		Nodes: nodes,
	}
	masters := clusterNodes.GetMasters()
	if len(masters) != 1 {
		t.Fatalf("Incorrect number of masters returned")
	}
	if masters[0].NodeAttributes.ID != "5dbeafc760e4ec355f007b2ce10c690a56306dc8" {
		t.Fatalf("Incorrect master list returned")
	}
}

func TestClusterNodes_GetReplicas(t *testing.T) {
	var nodes []*Node
	for i := 0; i <= 1; i++ {
		node, err := NewNode(context.TODO(), &redis.Options{
			Addr: "10.20.30.40:6379",
		}, &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rediscluster",
				Namespace: "default",
			},
		}, func(opt *redis.Options) *redis.Client {
			client, mock := redismock.NewClientMock()
			if i == 0 {
				mock.ExpectClusterNodes().SetVal(`5dbeafc760e4ec355f007b2ce10c690a56306dc8 10.244.0.225:6379@16379 myself,master - 0 1653476460000 9 connected
85613000e76a00c2da80e9eae0f2fed6bc857605 10.244.0.240:6379@16379 slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653476461000 9 connected
`)
			} else {
				mock.ExpectClusterNodes().SetVal(`5dbeafc760e4ec355f007b2ce10c690a56306dc8 10.244.0.225:6379@16379 master - 0 1653476460000 9 connected
85613000e76a00c2da80e9eae0f2fed6bc857605 10.244.0.240:6379@16379 myself,slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653476461000 9 connected
`)
			}
			return client
		})
		if err != nil {
			t.Fatalf("Got error while creating nodes. %v", err)
		}
		nodes = append(nodes, node)
	}

	clusterNodes := ClusterNodes{
		Nodes: nodes,
	}
	masters := clusterNodes.GetReplicas()
	if len(masters) != 1 {
		t.Fatalf("Incorrect number of replicas returned")
	}
	if masters[0].NodeAttributes.ID != "85613000e76a00c2da80e9eae0f2fed6bc857605" {
		t.Fatalf("Incorrect replica list returned")
	}
}

func TestClusterNodes_EnsureClusterReplicationRatioIfTooManyMasters(t *testing.T) {
	// We are testing here that a cluster is replicated in the way we specified.
	node1, err := NewNode(context.TODO(), &redis.Options{
		Addr: "10.20.30.40:6379",
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rediscluster",
			Namespace: "default",
		},
	}, func(opt *redis.Options) *redis.Client {
		client, mock := redismock.NewClientMock()
		mock.ExpectClusterNodes().SetVal(`9fd8800b31d569538917c0aaeaa5588e2f9c6edf 10.20.30.40:6379@16379 myself,master - 0 1652373716000 0 connected
9fd8800b31d569538917c0aaeaa5588e2f9c6edg 10.20.30.41:6379@16379 master - 0 1652373716000 0 connected
`)
		return client
	})
	if err != nil {
		t.Fatalf("received error while trying to create node %v", err)
	}

	replicaClient, replicaMock := redismock.NewClientMock()
	replicaMock.ExpectClusterNodes().SetVal(`9fd8800b31d569538917c0aaeaa5588e2f9c6edf 10.20.30.40:6379@16379 master - 0 1652373716000 0 connected
9fd8800b31d569538917c0aaeaa5588e2f9c6edg 10.20.30.41:6379@16379 myself,master - 0 1652373716000 0 connected
`)
	replicaMock.ExpectClusterReplicate("9fd8800b31d569538917c0aaeaa5588e2f9c6edf").SetVal("OK")

	node2, err := NewNode(context.TODO(), &redis.Options{
		Addr: "10.20.30.41:6379",
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rediscluster",
			Namespace: "default",
		},
	}, func(opt *redis.Options) *redis.Client {
		return replicaClient
	})
	if err != nil {
		t.Fatalf("received error while trying to create node %v", err)
	}

	clusterNodes := ClusterNodes{
		Nodes: []*Node{
			node1,
			node2,
		},
	}
	err = clusterNodes.EnsureClusterReplicationRatio(context.TODO(), &v1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: v1alpha1.RedisClusterSpec{
			Masters:           1,
			ReplicasPerMaster: 1,
		},
	})

	if err != nil {
		t.Fatalf("Did not expect error %v", err)
	}

	if err = replicaMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Expected node to become replica, but didn't. Err: %v", err)
	}
}

func TestClusterNodes_EnsureClusterReplicationRatioIfTooFewMasters(t *testing.T) {
	var nodes []*Node
	mocks := map[string]*redismock.ClientMock{}
	for i := 0; i <= 3; i++ {
		node, err := NewNode(context.TODO(), &redis.Options{
			Addr: "10.20.30.40:6379",
		}, &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rediscluster",
				Namespace: "default",
			},
		}, func(opt *redis.Options) *redis.Client {
			client, mock := redismock.NewClientMock()
			switch i {
			case 0:
				mock.ExpectClusterNodes().SetVal(`5dbeafc760e4ec355f007b2ce10c690a56306dc8 10.20.30.40:6379@16379 myself,master - 0 1653479781000 16 connected
4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad 10.20.30.41:6379@16379 slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781745 16 connected
0465e428668773fc3bbeb02150bbd4324e409fe0 10.20.30.42:6379@16379 slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781544 16 connected
85613000e76a00c2da80e9eae0f2fed6bc857605 10.20.30.43:6379@16379 slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781000 16 connected
`)
				mocks["5dbeafc760e4ec355f007b2ce10c690a56306dc8"] = &mock
			case 1:
				// Early return for this node
				mock.ExpectClusterNodes().SetVal(`5dbeafc760e4ec355f007b2ce10c690a56306dc8 10.20.30.40:6379@16379 master - 0 1653479781000 16 connected
4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad 10.20.30.41:6379@16379 myself,slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781745 16 connected
0465e428668773fc3bbeb02150bbd4324e409fe0 10.20.30.42:6379@16379 slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781544 16 connected
85613000e76a00c2da80e9eae0f2fed6bc857605 10.20.30.43:6379@16379 slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781000 16 connected
`)
				mock.ExpectClusterResetSoft().SetVal("OK")
				mocks["4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad"] = &mock
			case 2:
				mock.ExpectClusterNodes().SetVal(`5dbeafc760e4ec355f007b2ce10c690a56306dc8 10.20.30.40:6379@16379 master - 0 1653479781000 16 connected
4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad 10.20.30.41:6379@16379 slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781745 16 connected
0465e428668773fc3bbeb02150bbd4324e409fe0 10.20.30.42:6379@16379 myself,slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781544 16 connected
85613000e76a00c2da80e9eae0f2fed6bc857605 10.20.30.43:6379@16379 slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781000 16 connected
`)
				mocks["0465e428668773fc3bbeb02150bbd4324e409fe0"] = &mock
			case 3:
				mock.ExpectClusterNodes().SetVal(`5dbeafc760e4ec355f007b2ce10c690a56306dc8 10.20.30.40:6379@16379 master - 0 1653479781000 16 connected
4e70ffa7e012ecec890b25f52fbc3d2e8edd89ad 10.20.30.41:6379@16379 slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781745 16 connected
0465e428668773fc3bbeb02150bbd4324e409fe0 10.20.30.42:6379@16379 slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781544 16 connected
85613000e76a00c2da80e9eae0f2fed6bc857605 10.20.30.43:6379@16379 myself,slave 5dbeafc760e4ec355f007b2ce10c690a56306dc8 0 1653479781000 16 connected
`)
				mocks["85613000e76a00c2da80e9eae0f2fed6bc857605"] = &mock
			}
			return client
		})
		if err != nil {
			t.Fatalf("Got error whil trying to create node. %v", err)
		}
		nodes = append(nodes, node)
	}

	clusterNodes := ClusterNodes{
		Nodes: nodes,
	}
	err := clusterNodes.EnsureClusterReplicationRatio(context.TODO(), &v1alpha1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-cluster",
			Namespace: "default",
		},
		Spec: v1alpha1.RedisClusterSpec{
			Masters:           2,
			ReplicasPerMaster: 1,
		},
	})

	if err != nil {
		t.Fatalf("Did not expect error %v", err)
	}

	for node, mock := range mocks {
		realMock := *mock
		if err = realMock.ExpectationsWereMet(); err != nil {
			t.Fatalf("Expected node to become replica, but didn't. Node %s. Err: %v", node, err)
		}
	}
}

func TestClusterNodes_GetFailingNodes(t *testing.T) {
	var nodes []*Node
	mocks := map[string]*redismock.ClientMock{}
	for i := 0; i <= 0; i++ {
		node, err := NewNode(context.TODO(), &redis.Options{
			Addr: "10.20.30.40:6379",
		}, &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rediscluster",
				Namespace: "default",
			},
		}, func(opt *redis.Options) *redis.Client {
			client, mock := redismock.NewClientMock()
			switch i {
			case 0:
				mock.MatchExpectationsInOrder(false)
				mock.ExpectPing().SetVal("PONG")
				clusterNodeString := `c9d83f035342c51c8d23b32339f37656becd14c9 10.20.30.40:6379@16379 myself,master - 0 1653647426553 3 connected 0-5461
1a4c602fc868c69b74fc13f9b0410a20241c7197 10.20.30.41:6379@16379 master,fail - 1653646405584 1653646403000 4 connected
`
				// Cluster nodes will be called twice. Once for creating the nodes, the next for getting friends.
				mock.ExpectClusterNodes().SetVal(clusterNodeString)
				mock.ExpectClusterNodes().SetVal(clusterNodeString)
				mocks["c9d83f035342c51c8d23b32339f37656becd14c9"] = &mock
			}
			return client
		})
		if err != nil {
			t.Fatalf("Got error whil trying to create node. %v", err)
		}
		nodes = append(nodes, node)
	}
	clusterNodes := ClusterNodes{
		Nodes: nodes,
	}
	failingNodes, err := clusterNodes.GetFailingNodes(context.TODO())
	if err != nil {
		t.Fatalf("Failed to get failing nodes. %v", err)
	}
	if len(failingNodes) != 1 {
		t.Fatalf("incorrect amount of failing nodes returned")
	}
	if failingNodes[0].NodeAttributes.ID != "1a4c602fc868c69b74fc13f9b0410a20241c7197" {
		t.Fatalf("Incorrect node returned for failing nodes. Expected 1a4c602fc868c69b74fc13f9b0410a20241c7197. Got %s", failingNodes[0].NodeAttributes.ID)
	}
	for node, mock := range mocks {
		realMock := *mock
		if err = realMock.ExpectationsWereMet(); err != nil {
			t.Fatalf("Not all expectations from redis were met. Node %s. Err: %v", node, err)
		}
	}
}

func TestClusterNodes_ForgetNode(t *testing.T) {
	var nodes []*Node
	mocks := map[string]*redismock.ClientMock{}
	for i := 0; i <= 1; i++ {
		node, err := NewNode(context.TODO(), &redis.Options{
			Addr: "10.20.30.40:6379",
		}, &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rediscluster",
				Namespace: "default",
			},
		}, func(opt *redis.Options) *redis.Client {
			client, mock := redismock.NewClientMock()
			switch i {
			case 0:
				mock.MatchExpectationsInOrder(false)
				clusterNodeString := `1cbbfae6453680475e523e4d28438b1c1acf8cd3 10.20.30.40:6379@16379 myself,master - 0 1653647424000 2 connected 5462-10923
c9d83f035342c51c8d23b32339f37656becd14c9 10.20.30.41:6379@16379 master - 0 1653647426553 3 connected 0-5461
1a4c602fc868c69b74fc13f9b0410a20241c7197 10.20.30.42:6379@16379 master,fail - 1653646405584 1653646403000 4 connected`
				// Cluster nodes will be called twice. Once for creating the nodes, the next for getting friends.
				mock.ExpectClusterNodes().SetVal(clusterNodeString)
				mock.ExpectClusterForget("1a4c602fc868c69b74fc13f9b0410a20241c7197").SetVal("OK")
				mocks["1cbbfae6453680475e523e4d28438b1c1acf8cd3"] = &mock
			case 1:
				mock.MatchExpectationsInOrder(false)
				clusterNodeString := `1cbbfae6453680475e523e4d28438b1c1acf8cd3 10.20.30.40:6379@16379 master - 0 1653647424000 2 connected 5462-10923
c9d83f035342c51c8d23b32339f37656becd14c9 10.20.30.41:6379@16379 myself,master - 0 1653647426553 3 connected 0-5461
1a4c602fc868c69b74fc13f9b0410a20241c7197 10.20.30.42:6379@16379 master,fail - 1653646405584 1653646403000 4 connected`
				// Cluster nodes will be called twice. Once for creating the nodes, the next for getting friends.
				mock.ExpectClusterNodes().SetVal(clusterNodeString)
				mock.ExpectClusterForget("1a4c602fc868c69b74fc13f9b0410a20241c7197").SetVal("OK")
				mocks["c9d83f035342c51c8d23b32339f37656becd14c9"] = &mock
			}
			return client
		})
		if err != nil {
			t.Fatalf("Got error whil trying to create node. %v", err)
		}
		nodes = append(nodes, node)
	}
	clusterNodes := ClusterNodes{
		Nodes: nodes,
	}
	removeAbleNode := &Node{
		NodeAttributes: NodeAttributes{
			ID:   "1a4c602fc868c69b74fc13f9b0410a20241c7197",
			host: "10.20.30.42",
			port: "6379",
		},
	}
	err := clusterNodes.ForgetNode(context.TODO(), removeAbleNode)
	if err != nil {
		t.Fatalf("Failed to forget node. %v", err)
	}
	for node, mock := range mocks {
		realMock := *mock
		if err = realMock.ExpectationsWereMet(); err != nil {
			t.Fatalf("Not all expectations from redis were met. Node %s. Err: %v", node, err)
		}
	}
}
