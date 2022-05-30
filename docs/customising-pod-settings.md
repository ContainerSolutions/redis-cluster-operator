# Customising pod settings

To allow maximum flexibility for your environment, the Redis Operator allows overriding and customising most settings
on the Redis pods and adding additional containers or settings.

This is done through the `podSpec` field on the RedisCluster CRD. 

The podSpec will be merged with the necessary elements for the operator such as ports and configmaps.

## Examples

Overriding the Redis Image. 

This is especially useful if you have a custom Redis Image, or would like to use a different version

```yaml
apiVersion: cache.container-solutions.com/v1alpha1
kind: RedisCluster
metadata:
  name: rediscluster-sample
spec:
  masters: 3
  replicasPerMaster: 1
  podSpec:
    containers:
      - name: redis
        image: redis:5.0
```

