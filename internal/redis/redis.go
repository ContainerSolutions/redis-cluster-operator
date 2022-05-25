package redis

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"strconv"
	"strings"
)

const (
	TotalRedisSlots = 16384
)

func ProcessSlotStrings(slotStrings []string) []int32 {
	var result []int32
	for _, slotString := range slotStrings {
		slotParts := strings.Split(slotString, "-")
		if len(slotParts) == 2 {
			slotStart, _ := strconv.Atoi(slotParts[0])
			slotEnd, _ := strconv.Atoi(slotParts[1])
			for slot := slotStart; slot <= slotEnd; slot++ {
				result = append(result, int32(slot))
			}
		}
		if len(slotParts) == 1 {
			slot, _ := strconv.Atoi(slotParts[0])
			result = append(result, int32(slot))
		}
	}
	return result
}

// NodeAttributes represents the data returned from the CLUSTER NODES commands.
// The format returned from Redis contains the fields, split by spaces
//
// <id> <ip:port@cport> <flags> <master> <ping-sent> <pong-recv> <config-epoch> <link-state> <slot> <slot> ... <slot>
//
// <id> represents the ID of the node in UUID format
// <ip:port@cport> part has the IP and port of the redis server, with the gossip port after @
// <flags> is a string of flags separated by comma (,). Useful flags include master|slave|myself. Myself is the indicator that this line is for the calling node
// <master> represents the node ID that is being replicated, if the node is a slave. if it is not replicating anything it will be replaced by a dash (-)
// <slot>... represents slot ranges assigned to this node. The format is ranges, or single numbers. 0-4 represents all slots from 0 to 4. 8 represents the single slot 8
type NodeAttributes struct {
	ID    string
	host  string
	port  string
	flags []string
	slots []int32
}

func NewNodeAttributes(nodeString string) NodeAttributes {
	// Output Format: <0:id> <1:ip:port@cport> <2:flags> <3:master> <4:ping-sent> <5:pong-recv> <6:config-epoch> <7:link-state> <slot> <slot> ... <slot>
	friendFields := strings.Split(nodeString, " ")
	address := strings.Split(friendFields[1], "@")[0]
	addressParts := strings.Split(address, ":")
	return NodeAttributes{
		ID:    friendFields[0],
		host:  addressParts[0],
		port:  addressParts[1],
		flags: strings.Split(friendFields[2], ","),
		slots: ProcessSlotStrings(friendFields[8:]),
	}
}

func (n *NodeAttributes) HasFlag(flag string) bool {
	for _, _flag := range n.flags {
		if _flag == flag {
			return true
		}
	}
	return false
}

// Node represents a single Redis Node with a client, and a client builder.
// The client builder is necessary in case we are getting nodes from this node, for example when we load friends.
// We need a clientBuilder, so we can create the same base client for nodes fetched through this node,
// for example getting all of the attched nodes.
// This is especially useful for testing, as we need to pass in a mocked constructor for child clients.
type Node struct {
	*redis.Client
	NodeAttributes NodeAttributes
	clientBuilder  func(opt *redis.Options) *redis.Client
}

func NewNode(ctx context.Context, opt *redis.Options, clientBuilder func(opt *redis.Options) *redis.Client) (*Node, error) {
	redisClient := clientBuilder(opt)
	node := &Node{
		Client:         redisClient,
		NodeAttributes: NodeAttributes{},
		clientBuilder:  clientBuilder,
	}
	attributes, err := node.GetSelfAttributes(ctx)
	if err != nil {
		return nil, err
	}
	node.NodeAttributes = attributes
	node.NodeAttributes.host = strings.Split(opt.Addr, ":")[0]
	node.NodeAttributes.port = strings.Split(opt.Addr, ":")[1]
	return node, nil
}

func (n *Node) ReloadNodeInfo(ctx context.Context) error {
	oldAttributes := n.NodeAttributes
	attributes, err := n.GetSelfAttributes(ctx)
	if err != nil {
		return err
	}
	n.NodeAttributes = attributes
	n.NodeAttributes.host = oldAttributes.host
	n.NodeAttributes.port = oldAttributes.port
	return nil
}

func (n *Node) GetSelfAttributes(ctx context.Context) (NodeAttributes, error) {
	nodes, err := n.ClusterNodes(ctx).Result()
	if err != nil {
		return NodeAttributes{}, err
	}
	for _, friendString := range strings.Split(nodes, "\n") {
		if strings.TrimSpace(friendString) == "" {
			continue
		}
		nodeAttributes := NewNodeAttributes(friendString)
		if !nodeAttributes.HasFlag("myself") {
			continue
		}
		return nodeAttributes, nil
	}
	return NodeAttributes{}, errors.New("could not find myself in nodes list")
}

func (n *Node) IsMaster() bool {
	return n.NodeAttributes.HasFlag("master")
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
		nodeAttributes := NewNodeAttributes(friendString)
		if nodeAttributes.HasFlag("myself") {
			// We only want to return nodes which are friends not ourself
			continue
		}
		result = append(result, &Node{
			n.clientBuilder(&redis.Options{
				Addr: nodeAttributes.host + ":" + nodeAttributes.port,
			}),
			nodeAttributes,
			n.clientBuilder,
		})
	}
	return result, err
}

// MeetNode let's the node recognise and connect to another Redis Node
func (n *Node) MeetNode(ctx context.Context, node *Node) error {
	//parts := strings.Split(node.Client.Options().Addr, ":")
	//host := parts[0]
	//port := parts[1]
	err := n.ClusterMeet(ctx, node.NodeAttributes.host, node.NodeAttributes.port).Err()
	return err
}
