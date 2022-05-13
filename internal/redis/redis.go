package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"strings"
)

type NodeAttributes struct {
	ID string
}

type Node struct {
	*redis.Client
	NodeAttributes NodeAttributes
	clientBuilder func(opt *redis.Options) *redis.Client
}

// GetFriends returns a list of all the other Redis nodes that this node knows about
func (n *Node) GetFriends(ctx context.Context) ([]*Node, error) {
	var result []*Node
	nodes, err := n.ClusterNodes(ctx).Result()
	if err != nil {
		return result, err
	}
	for _, friendString := range strings.Split(nodes, "\n") {
		if strings.TrimSpace(friendString) == "" {
			continue
		}
		// Output Format: <0:id> <1:ip:port@cport> <2:flags> <3:master> <4:ping-sent> <5:pong-recv> <6:config-epoch> <7:link-state> <slot> <slot> ... <slot>
		friendFields := strings.Split(friendString, " ")

		if strings.Contains(friendFields[2], "myself") {
			// We only want to return nodes which are friends not ourself
			continue
		}
		address := strings.Split(friendFields[1], "@")[0]
		result = append(result, &Node{
			n.clientBuilder(&redis.Options{
				Addr:               address,
			}),
			NodeAttributes{
				ID: friendFields[0],
			},
			n.clientBuilder,
		})
	}
	return result, err
}

// MeetNode let's the node recognise and connect to another Redis Node
func (n *Node) MeetNode(ctx context.Context, node *Node) error {
	parts := strings.Split(node.Client.Options().Addr, ":")
	host := parts[0]
	port := parts[1]
	err := n.ClusterMeet(ctx, host, port).Err()
	return err
}
