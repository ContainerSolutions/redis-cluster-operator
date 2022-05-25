# Specifying Redis Configuration

Redis is usually configured through a redis.conf file, which is specified to the server on startup. 

When the Operator creates a cluster, it also creates a configmap which holds the Redis configuration for the nodes.

The configuration for Redis Nodes can be specified through the `config` key in the Redis CRD.

```yaml
apiVersion: cache.container-solutions.com/v1alpha1
kind: RedisCluster
metadata:
  name: rediscluster-sample
spec:
  masters: 3
  replicasPerMaster: 1
  # Config holds all of the settings for redis.conf, and these are propagated to the cluster created.
  # All default settings are overridable.
  # 
  # Default redis.conf:
  # 
  #     port 6379
  #     cluster-enabled yes
  #     cluster-config-file nodes.conf
  #     cluster-node-timeout 5000
  #
  config: |
    maxmemory 200mb
    maxmemory-policy allkeys-lru
```
