package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/containersolutions/redis-cluster-operator/api/v1alpha1"
	"math"
	"sort"
)

type ClusterNodes struct {
	Nodes []*Node
}

func (c *ClusterNodes) ReloadNodes(ctx context.Context) error {
	for _, node := range c.Nodes {
		err := node.ReloadNodeInfo(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ClusterNodes) GetCommandingNode(ctx context.Context) (*Node, error) {
	for _, node := range c.Nodes {
		if node.IsMaster() {
			err := node.Ping(ctx).Err()
			return node, err
		}
	}
	return nil, errors.New("no commanding nodes found")
}

// GetFailingNodes returns a list of all the nodes marked as failing in the cluster.
// Any nodes marked as failing in `cluster nodes` command will be returned
// We will most likely not be able to connect to these nodes as they would be restarted pods
func (c *ClusterNodes) GetFailingNodes(ctx context.Context) ([]*Node, error) {
	node, err := c.GetCommandingNode(ctx)
	if err != nil {
		return nil, err
	}

	fmt.Println(node.NodeAttributes.ID)

	friends, err := node.GetFriends(ctx)

	if err != nil {
		return nil, err
	}

	var result []*Node

	for _, friend := range friends {
		if friend.NodeAttributes.HasFlag("fail") {
			result = append(result, friend)
		}
	}

	return result, nil
}

func (c *ClusterNodes) ClusterMeet(ctx context.Context) error {
	for _, node := range c.Nodes {
		for _, joinNode := range c.Nodes {
			err := node.MeetNode(ctx, joinNode)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *ClusterNodes) GetAssignedSlots() []int32 {
	var result []int32
	for _, node := range c.Nodes {
		result = append(result, node.NodeAttributes.slots...)
	}
	return result
}

func (c *ClusterNodes) GetMissingSlots() []int32 {
	var result []int32
	// For optimal processing, we create a map first of all the slots,
	// then delete the assigned slots from the map,
	// then return the map as a slice
	slotMap := map[int32]interface{}{}
	for slot := int32(0); slot < TotalRedisSlots; slot++ {
		slotMap[slot] = true
	}
	for _, slot := range c.GetAssignedSlots() {
		delete(slotMap, slot)
	}
	for slot, _ := range slotMap {
		result = append(result, slot)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}

func (c *ClusterNodes) CalculateSlotAssignment() map[*Node][]int32 {
	slotAssignment := map[*Node][]int32{}

	// We add one for the remainder, as 16834 does not go even into an uneven amount of nodes.
	// By adding one to each node, we don't need a check after to see whether there are unassigned slots left,
	// as assignable slots will be less than slotsNeededPerNode * len(nodes).
	slotsNeededPerNode := int(TotalRedisSlots/len(c.GetMasters())) + 1
	slotsStillToAssign := c.GetMissingSlots()
	for _, node := range c.GetMasters() {
		if len(node.NodeAttributes.slots) < slotsNeededPerNode {
			// This node needs some slots to fill it's quota of slots.
			// We can cut some slots from the allocatable slots, if there are enough
			slotsNeededForNode := slotsNeededPerNode - len(node.NodeAttributes.slots)
			var slotsTake []int32
			if len(slotsStillToAssign) <= slotsNeededForNode {
				slotsTake = slotsStillToAssign
				slotsStillToAssign = []int32{}
			}
			if len(slotsStillToAssign) > slotsNeededForNode {
				slotsTake = slotsStillToAssign[:slotsNeededForNode]
				slotsStillToAssign = slotsStillToAssign[slotsNeededForNode:]
			}
			slotAssignment[node] = slotsTake
		}
	}
	return slotAssignment
}

func (c *ClusterNodes) GetMasters() []*Node {
	var masters []*Node
	for _, node := range c.Nodes {
		if node.IsMaster() {
			masters = append(masters, node)
		}
	}
	return masters
}

func (c *ClusterNodes) GetReplicas() []*Node {
	var replicas []*Node
	for _, node := range c.Nodes {
		if !node.IsMaster() {
			replicas = append(replicas, node)
		}
	}
	return replicas
}

func (c *ClusterNodes) EnsureClusterReplicationRatio(ctx context.Context, cluster *v1alpha1.RedisCluster) error {
	masters := c.GetMasters()

	if len(masters) == int(cluster.Spec.Masters) {
		// There are the appropriate amount of masters and replicas
		return nil
	}

	// If we have too many masters, we can failover the extra masters to replicate the original masters
	if len(masters) > int(cluster.Spec.Masters) {
		// todo select keepable masters as masters with most slots attached
		var keepMasters []*Node
		var removeMasters []*Node
		// We have too many masters and need to fail over some
		keepMasters = masters[:cluster.Spec.Masters]
		removeMasters = masters[cluster.Spec.Masters:]

		// todo we need to replicate the masters which have the least amount of replicas, rather than just the matching index
		for k, removeMaster := range removeMasters {
			selectedMaster := keepMasters[int(math.Mod(float64(k), float64(cluster.Spec.Masters)))]

			// todo we need to make sure that there are no slots in the master before replicating it.
			err := removeMaster.ClusterReplicate(ctx, selectedMaster.NodeAttributes.ID).Err()
			if err != nil {
				return err
			}
		}
		return nil
	}

	if len(masters) < int(cluster.Spec.Masters) {
		mastersNeeded := int(cluster.Spec.Masters) - len(masters)
		replicas := c.GetReplicas()
		replicaNeedsReset := replicas[:mastersNeeded]
		for _, replica := range replicaNeedsReset {
			err := replica.ClusterResetSoft(ctx).Err()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
