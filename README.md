# Redis Cluster Operator

The Redis Cluster Operator runs Redis Clusters on Kubernetes.

We've found many operators which either use the redis-cli directly, which makes it hard to customise
behaviour, or do not support a full productionised suite of features.

The aim for this operator is to run productionised clusters with most necessary features, 
as well as providing additions such as RunBooks to help debug issues with Redis Clusters 
when running them with this Operator, and ready-made load tests to test your Redis Clusters with real traffic.

* [Features](#features-this-operator-supports)
* [Installing The Operator](#installing-the-operator)
  * [Bundled Cluster Wide](#bundled-cluster-wide)
  * [Bundled Namespaced](#bundled-namespaced)
  * [OLM bundle](#olm-bundle)
* [Creating your first Redis Cluster](#creating-your-first-redis-cluster)
* [Contributing](./CONTRIBUTING.md)

## Features this operator supports
- [ ] Cluster Creation and Management
- [ ] Support for replicated clusters (Master-Replica splits)
- [ ] 0 Downtime scaling
- [ ] 0 Downtime upgrades
- [ ] Persistent clusters (Supported through Kubernetes PVC management)
- [ ] Backup & Restore capability for persistent clusters
- [ ] Documentation on observability for clusters
- [ ] Runbooks for common debugging issues and resolutions
- [ ] Ready-made k6s load tests to load Redis Clusters

## Installing the Operator

### bundled cluster-wide

The operator gets bundled for every release together with all of it's crds, rbac, and deployment.

The origin bundle works in cluster mode, and will manage all RedisClusters created in all namespaces. 

To install or upgrade the operator 
```shell
kubectl apply -f https://github.com/ContainerSolutions/redis-cluster-operator/releases/latest/download/bundle.yml
```

This will install the Operator in a new namespace `redis-cluster-operator`. 

You can also [install the operator in a custom namespace](./docs/installing-in-a-custom-namespace.md).

### bundled namespaced

The operator currently works in cluster-wide mode, but namespaced mode will be supported in future.

We know it's quite important for redundancy, reducing single-point of failures, 
as well as tenanted models, or excluding namespaces from the operator.

Namespaced mode will be supported in the future.

### OLM bundle

> OLM bundling support is a work in progress. 
> There are remnants of OLM due to the initial Operator SDK installation, 
> but we have not specifically tested and looked at it in depth.

## Creating your first Redis Cluster

To create your first Redis cluster, you'll need a CRD.

```yaml
apiVersion: cache.container-solutions.com/v1alpha1
kind: RedisCluster
metadata:
  name: rediscluster-product-api
spec:
  masters: 3
  replicasPerMaster: 1
```

Once applied, the Operator will create all the necessary nodes, and set up the cluster ready for use.
