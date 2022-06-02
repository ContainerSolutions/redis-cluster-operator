package redis

import (
	"context"
	"errors"
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

func (c *ClusterNodes) ForgetNode(ctx context.Context, forgetNode *Node) error {
	for _, node := range c.Nodes {
		err := node.ClusterForget(ctx, forgetNode.NodeAttributes.ID).Err()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetFailingNodes returns a list of all the nodes marked as failing in the cluster.
// Any nodes marked as failing in `cluster nodes` command will be returned
// We will most likely not be able to connect to these nodes as they would be restarted pods
func (c *ClusterNodes) GetFailingNodes(ctx context.Context) ([]*Node, error) {
	node, err := c.GetCommandingNode(ctx)
	if err != nil {
		return nil, err
	}

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

		// We want to sort the masters by the amount of slots they have.
		// That way when we select removable masters, they are most likely to be empty from slots
		sort.Slice(masters, func(i, j int) bool {
			return len(masters[i].NodeAttributes.slots) > len(masters[j].NodeAttributes.slots)
		})

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

func (c *ClusterNodes) MoveSlot(ctx context.Context, source, destination *Node, slot int) error {
	err := destination.Client.Do(ctx, "cluster", "setslot", slot, "importing", source.NodeAttributes.ID).Err()
	if err != nil {
		return err
	}
	err = source.Client.Do(ctx, "cluster", "setslot", slot, "migrating", destination.NodeAttributes.ID).Err()
	if err != nil {
		return err
	}

	for {
		keys, err := source.ClusterGetKeysInSlot(ctx, slot, 50).Result()
		if err != nil {
			return err
		}
		if len(keys) == 0 {
			break
		}

		// migrate 10.244.1.132 6379 "" 0 5000 KEYS A:163262 A:166510 A:172223 A:177551 A:18733 A:21915 A:247961 A:30954 A:383958 A:392919
		migrateCmd := []interface{}{
			"migrate",
			destination.NodeAttributes.GetHost(),
			destination.NodeAttributes.GetPort(),
			"",
			"0",
			"5000",
			"KEYS",
		}
		for _, key := range keys {
			migrateCmd = append(migrateCmd, key)
		}
		err = source.Client.Do(
			ctx,
			migrateCmd...,
		).Err()
		if err != nil {
			return err
		}
	}
	err = destination.Client.Do(ctx, "cluster", "setslot", slot, "NODE", destination.NodeAttributes.ID).Err()
	if err != nil {
		return err
	}
	err = source.Client.Do(ctx, "cluster", "setslot", slot, "NODE", destination.NodeAttributes.ID).Err()
	if err != nil {
		return err
	}
	for _, node := range c.Nodes {
		if node.NodeAttributes.ID == destination.NodeAttributes.ID {
			continue
		}
		if node.NodeAttributes.ID == source.NodeAttributes.ID {
			continue
		}
		err = source.Client.Do(ctx, "cluster", "setslot", slot, "NODE", destination.NodeAttributes.ID).Err()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ClusterNodes) BalanceSlots(ctx context.Context, cluster *v1alpha1.RedisCluster) error {
	slotMoves := c.CalculateRebalance(ctx, cluster)
	for _, slotMove := range slotMoves {
		for _, slot := range slotMove.Slots {
			err := c.MoveSlot(ctx, slotMove.Source, slotMove.Destination, int(slot))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type slotMoveMap struct {
	Source      *Node
	Destination *Node
	Slots       []int32
}

func (c *ClusterNodes) CalculateRebalance(ctx context.Context, cluster *v1alpha1.RedisCluster) []slotMoveMap {
	// First we sort the nodes by slot count.
	// This allows us to loop through and steal slots from nodes with too many slots,
	// and then when we get to the ones with too few slots, we have a list of "stealable" slots to take from.
	masters := c.Nodes
	sort.Slice(masters, func(i, j int) bool {
		return len(masters[i].NodeAttributes.GetSlots()) > len(masters[j].NodeAttributes.GetSlots())
	})
	var result []slotMoveMap
	stealMap := map[*Node][]int32{}
	for _, node := range c.GetMasters() {
		slots := node.NodeAttributes.GetSlots()
		sort.Slice(slots, func(i, j int) bool {
			return slots[i] < slots[j]
		})
		if len(slots) > int(node.NeedsSlotCount(cluster)) {
			// This node has too many slots
			// We need to steal some slots from it
			stealMap[node] = slots[:len(slots)-int(node.NeedsSlotCount(cluster))]
		}
		if len(slots) <= int(node.NeedsSlotCount(cluster)) {
			// This node has too few slots
			// We need to take slots from the stealable set
			slotsNeeded := int(node.NeedsSlotCount(cluster)) - len(slots)
			for stealNode, stealSlots := range stealMap {
				if slotsNeeded == 0 {
					break
				}
				if len(stealSlots) > slotsNeeded {
					slotsStolen := stealSlots[:slotsNeeded]
					stealMap[stealNode] = stealSlots[slotsNeeded:]
					result = append(result, slotMoveMap{
						Source:      stealNode,
						Destination: node,
						Slots:       slotsStolen,
					})
					break
				}
				if len(stealSlots) <= slotsNeeded {
					slotsNeeded -= len(stealSlots)
					stealMap[stealNode] = []int32{}
					result = append(result, slotMoveMap{
						Source:      stealNode,
						Destination: node,
						Slots:       stealSlots,
					})
				}
			}
		}
	}
	return result
}
