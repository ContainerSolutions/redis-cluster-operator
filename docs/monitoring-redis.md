# Monitoring Redis Clusters

Monitoring has purposefully not been built into the Operator to allow flexibility in monitoring choices.

Below are some guidelines on monitoring Redis Clusters through different tools.

## Prometheus Operator

If you are using the Prometheus Operator to monitor your appliances, monitoring is quite simple. 

First you'll need to add the Redis Exporter to export metrics.

You can do this in the RedisCluster CRD.

```yaml
apiVersion: cache.container-solutions.com/v1alpha1
kind: RedisCluster
metadata:
  name: rediscluster-sample
spec:
  masters: 3
  replicasPerMaster: 1
  config: |
    maxmemory 200mb
    maxmemory-policy allkeys-lru
  podSpec:
    containers:
    # Add additional container to monitor each Redis Node
    - name: redis-exporter
      image: oliver006/redis_exporter:latest
      ports:
        - name: metrics
          containerPort: 9121
      env:
        - name: REDIS_ADDR
          value: 'redis://localhost:6379'
```

The Redis Operator will now create an additional container for each pod that runs the exporter.

Next you'll need a `PodMonitor` and a `Prometheus` instance to scrape the Redis metrics.

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  labels:
    app: rediscluster-sample
  name: rediscluster-sample-monitor
spec:
  podMetricsEndpoints:
  - interval: 5s
    port: metrics
  selector:
    matchLabels:
      # We need to specify which cluster, and which component to monitor.
      # All pods have these labels for a Redis Cluster
      cache.container-solutions.com/cluster-component: redis
      cache.container-solutions.com/cluster-name: rediscluster-sample
---
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
metadata:
  name: prometheus
spec:
  podMonitorSelector:
    matchLabels:
      app: rediscluster-sample
```

You should now receive Redis information in the Prometheus instance. 

[This dashboard](https://grafana.com/grafana/dashboards/763) works well for Grafana when using the Redis Exporter.
