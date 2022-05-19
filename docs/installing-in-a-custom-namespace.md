# Deploying the Operator in a custom namespace

You can create a kustomization file which overrides the namespace, 
and the bundle will automatically update all of the resources with the new namespace

```shell
REDIS_CLUSTER_OPERATOR_HOME=$(mktemp -d)

cat <<EOF >$REDIS_CLUSTER_OPERATOR_HOME/kustomization.yaml
namespace: custom-namespace
bases:
  - https://github.com/ContainerSolutions/redis-cluster-operator/releases/latest/download/bundle.yml
  EOF
```

When you build and apply the kustomization file, it will automatically create a new `custom-namespace` namespace, 
and update the namespace references for all the operator components.
