package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"testing"
)

func TestRedisNodeGetFriendsReturnsKnowsNodes(t *testing.T) {
	db, mock := redismock.NewClientMock()
	redisNode := Node{
		Client: db,
		clientBuilder: func(opt *redis.Options) *redis.Client {
			client, _ := redismock.NewClientMock()
			return client
		},
	}
	mock.ExpectClusterNodes().SetVal(`335e5ceff013eeebdbdb71bb65b4c1aeaf6a06f5 10.244.0.156:6379@16379 master - 0 1652373719041 2 connected
9fd8800b31d569538917c0aaeaa5588e2f9c6edf 10.244.0.218:6379@16379 myself,master - 0 1652373716000 0 connected
8a99a71a38d099de6862284f5aab9329d796c34f 10.244.0.160:6379@16379 master - 0 1652373718026 1 connected
`)

	nodes, err := redisNode.GetFriends(context.TODO())
	if err != nil {
		t.Fatalf("Got error when trying to get node friends %v", err)
	}
	for _, node := range nodes {
		if node.NodeAttributes.ID != "8a99a71a38d099de6862284f5aab9329d796c34f" && node.NodeAttributes.ID != "335e5ceff013eeebdbdb71bb65b4c1aeaf6a06f5" {
			t.Fatalf("Wrong node returned in friend list. Got %s", node.NodeAttributes.ID)
		}
	}
	if len(nodes) != 2 {
		t.Fatalf("Did not receive the right amount of friends for node")
	}
}

func TestRedisNodeGetFriendsReturnsEmptySliceIfNotFriends(t *testing.T) {
	db, mock := redismock.NewClientMock()
	redisNode := Node{
		Client: db,
		clientBuilder: func(opt *redis.Options) *redis.Client {
			client, _ := redismock.NewClientMock()
			return client
		},
	}
	mock.ExpectClusterNodes().SetVal(`335e5ceff013eeebdbdb71bb65b4c1aeaf6a06f5 10.244.0.156:6379@16379 myself,master - 0 1652373719041 2 connected
`)

	nodes, err := redisNode.GetFriends(context.TODO())
	if err != nil {
		t.Fatalf("Got error when trying to get node friends %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("Did not receive the right amount of friends for node")
	}
}
