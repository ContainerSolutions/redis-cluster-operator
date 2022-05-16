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
