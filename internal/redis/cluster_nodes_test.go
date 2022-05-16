package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
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

