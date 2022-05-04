# Redis Cluster Operator

The Redis Cluster Operator runs Redis Clusters on Kubernetes.

We've found many operators which either use the redis-cli directly, which makes it hard to customise
behaviour, or do not support a full productionised suite of features.

The aim for this operator is to run productionised clusters with most necessary features, 
as well as providing additionals such as RunBooks to help debug issues with Redis Clusters 
when running them with this Operator, and ready made load tests to test your Redis Clusters with real traffic.

## Features this operator aims to cover
- [ ] Cluster Creation and Management
- [ ] Support for replicated clusters (Master-Replica splits)
- [ ] 0 Downtime scaling
- [ ] 0 Downtime upgrades
- [ ] Persistent clusters (Supported through Kubernetes PVC management)
- [ ] Backup & Restore capability for persistent clusters
- [ ] Documentation on observability for clusters
- [ ] Runbooks for common debugging issues and resolutions
- [ ] Ready made k6s load tests to load Redis Clusters
