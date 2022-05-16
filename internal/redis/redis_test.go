package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"reflect"
	"testing"
)

// region NewNodeAttributes
func TestNewNodeAttributes(t *testing.T) {
	attributes := NewNodeAttributes("9fd8800b31d569538917c0aaeaa5588e2f9c6edf 10.244.0.218:6379@16379 myself,master - 0 1652373716000 0 connected")
	if attributes.host != "10.244.0.218" || attributes.port != "6379" || attributes.ID != "9fd8800b31d569538917c0aaeaa5588e2f9c6edf" {
		t.Fatalf("Attributes not being correctly extracted from node string")
	}
}

// endregion

// region NodeAttributes
func TestNodeAttributes_HasFlag(t *testing.T) {
	attributes := NewNodeAttributes("9fd8800b31d569538917c0aaeaa5588e2f9c6edf 10.244.0.218:6379@16379 myself,master - 0 1652373716000 0 connected")
	if !attributes.HasFlag("myself") || !attributes.HasFlag("master") {
		t.Fatalf("Flags are not being marked correctly")
	}
}

func TestNodeAttributes_LoadsSlotInformation(t *testing.T) {
	attributes := NewNodeAttributes("103791967781b9db4ae663dd060b51c442bd7105 10.244.0.250:6379@16379 master - 0 1652695701569 5 connected 0-9 11-12 14 16-19")
	expectedSlots := []int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 11, 12, 14, 16, 17, 18, 19}
	if !reflect.DeepEqual(attributes.slots, expectedSlots) {
		t.Fatalf("Expected assigned slot to be %v, got %v", expectedSlots, attributes.slots)
	}
}

// endregion

// region ProcessSlotString
func TestProcessSlotString(t *testing.T) {
	// 0-9 11-12 14 16-19
	got := ProcessSlotStrings([]string{"0-9", "11-12", "14", "16-19"})
	expected := []int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 11, 12, 14, 16, 17, 18, 19}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("Expcted slot list of %v, got %v", expected, got)
	}
}

// endregion

// region NewNode
func TestNewNodehasAttributesAttached(t *testing.T) {
	node, err := NewNode(context.TODO(), &redis.Options{
		Addr: "localhost:6379",
	}, func(opt *redis.Options) *redis.Client {
		db, mock := redismock.NewClientMock()
		mock.ExpectClusterNodes().SetVal(`335e5ceff013eeebdbdb71bb65b4c1aeaf6a06f5 10.244.0.156:6379@16379 master - 0 1652373719041 2 connected
9fd8800b31d569538917c0aaeaa5588e2f9c6edf 10.244.0.218:6379@16379 myself,master - 0 1652373716000 0 connected
8a99a71a38d099de6862284f5aab9329d796c34f 10.244.0.160:6379@16379 master - 0 1652373718026 1 connected
`)
		return db
	})
	if err != nil {
		t.Fatalf("Tried to create new node, but received error %v", err)
	}
	if node.NodeAttributes.ID != "9fd8800b31d569538917c0aaeaa5588e2f9c6edf" {
		t.Fatalf("Wrong attributes attached for node")
	}
}

// endregion

// region GetSelfAttributes
func TestNode_GetSelfAttributes(t *testing.T) {
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
	attributes, err := redisNode.GetSelfAttributes(context.TODO())
	if err != nil {
		t.Fatalf("Got error while trying to read my attributes %v", err)
	}
	if attributes.ID != "9fd8800b31d569538917c0aaeaa5588e2f9c6edf" {
		t.Fatalf("Got info for the wrong node. Expected info for 9fd8800b31d569538917c0aaeaa5588e2f9c6edf, Got info for %s", attributes.ID)
	}
}

// endregion

// region GetFriends
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

// endregion

// region MeetNode
func TestMeetNodeRunsNodeMeetForNewNode(t *testing.T) {
	db, mock := redismock.NewClientMock()
	redisNode := Node{
		Client: db,
		clientBuilder: func(opt *redis.Options) *redis.Client {
			client, _ := redismock.NewClientMock()
			return client
		},
		NodeAttributes: NodeAttributes{
			ID:    "123456789",
			host:  "localhost",
			port:  "6379",
			flags: []string{"master"},
		},
	}
	mock.ExpectClusterMeet("localhost", "6379").SetVal("OK")
	err := redisNode.MeetNode(context.TODO(), &Node{
		Client: db,
		clientBuilder: func(opt *redis.Options) *redis.Client {
			client, _ := redismock.NewClientMock()
			return client
		},
		NodeAttributes: NodeAttributes{
			ID:    "23456789",
			host:  "localhost",
			port:  "6379",
			flags: []string{"master"},
		},
	})
	if err != nil {
		t.Fatalf("Received error while trying to meet nodes %v", err)
	}
	if mock.ExpectationsWereMet() != nil {
		t.Fatalf("Not all of the required Redis commands were run")
	}
}

// endregion

// region IsMaster
func TestNode_IsMasterReturnsTrueIfMaster(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectClusterNodes().SetVal(`335e5ceff013eeebdbdb71bb65b4c1aeaf6a06f5 10.244.0.156:6379@16379 master - 0 1652373719041 2 connected
9fd8800b31d569538917c0aaeaa5588e2f9c6edf 10.244.0.218:6379@16379 myself,master - 0 1652373716000 0 connected
`)
	node, err := NewNode(context.TODO(), &redis.Options{
		Addr: "10.244.0.218:6379",
	}, func(opt *redis.Options) *redis.Client {
		return db
	})
	if err != nil {
		t.Fatalf("Received error while trying to create node %v", err)
	}

	if node.IsMaster() != true {
		t.Fatalf("A master returned false for IsMaster")
	}
}

func TestNode_IsMasterReturnsFalseIfReplica(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectClusterNodes().SetVal(`335e5ceff013eeebdbdb71bb65b4c1aeaf6a06f5 10.244.0.156:6379@16379 master - 0 1652373719041 2 connected
9fd8800b31d569538917c0aaeaa5588e2f9c6edf 10.244.0.218:6379@16379 myself,slave - 0 1652373716000 0 connected
`)
	node, err := NewNode(context.TODO(), &redis.Options{
		Addr: "10.244.0.218:6379",
	}, func(opt *redis.Options) *redis.Client {
		return db
	})
	if err != nil {
		t.Fatalf("Received error while trying to create node %v", err)
	}

	if node.IsMaster() != false {
		t.Fatalf("A replica returned true for IsMaster")
	}
}

// endregion
