package redis

import (
	"context"
	"sort"
)

type ClusterNodes struct {
	Nodes []*Node
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
	slotsNeededPerNode := int(TotalRedisSlots / len(c.Nodes)) + 1
	slotsStillToAssign := c.GetMissingSlots()
	for _, node := range c.Nodes {
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
