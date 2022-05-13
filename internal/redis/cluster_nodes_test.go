package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"testing"
)

func TestClusterMeetMeetsAllNodes(t *testing.T) {
	node1, err := NewNode(context.TODO(), &redis.Options{
		Addr: "10.20.30.40:6379",
	}, func(opt *redis.Options) *redis.Client {
		db, mock := redismock.NewClientMock()
		mock.ExpectClusterNodes().SetVal(`9fd8800b31d569538917c0aaeaa5588e2f9c6edf 10.20.30.40:6379@16379 myself,master - 0 1652373716000 0 connected
8a99a71a38d099de6862284f5aab9329d796c34f 10.20.30.41:6379@16379 master - 0 1652373718026 1 connected
`)
		mock.ExpectClusterMeet("10.20.30.40", "6379").SetVal("OK")
		mock.ExpectClusterMeet("10.20.30.41", "6379").SetVal("OK")
		return db
	})
	if err != nil {
		t.Fatalf("received error while trying to create node %v", err)
	}
	node2, err := NewNode(context.TODO(), &redis.Options{
		Addr: "10.20.30.40:6379",
	}, func(opt *redis.Options) *redis.Client {
		db, mock := redismock.NewClientMock()
		mock.ExpectClusterNodes().SetVal(`9fd8800b31d569538917c0aaeaa5588e2f9c6edf 10.20.30.40:6379@16379 master - 0 1652373716000 0 connected
8a99a71a38d099de6862284f5aab9329d796c34f 10.20.30.41:6379@16379 myself,master - 0 1652373718026 1 connected
`)
		mock.ExpectClusterMeet("10.20.30.40", "6379").SetVal("OK")
		mock.ExpectClusterMeet("10.20.30.41", "6379").SetVal("OK")
		return db
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
}
