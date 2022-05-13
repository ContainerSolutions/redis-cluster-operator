package redis

import "context"

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
