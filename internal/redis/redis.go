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

func (node *Node) GetFriends(ctx context.Context) ([]*Node, error) {
	var result []*Node
	nodes, err := node.ClusterNodes(ctx).Result()
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
			node.clientBuilder(&redis.Options{
				Addr:               address,
			}),
			NodeAttributes{
				ID: friendFields[0],
			},
			node.clientBuilder,
		})
	}
	return result, err
}
