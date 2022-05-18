package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
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
