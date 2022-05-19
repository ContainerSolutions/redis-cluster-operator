# Developing on the Redis Cluster Operator

When working on the Redis Cluster Operator there are a few things to be aware of.

## Makefile

The Makefile in the root of this project documents and scripts all the normal development processes related to the 
development of the operator.

To see all the available commands, run `make help`
```shell
$ make help 

Usage:
  make <target>

General
  help             Display this help.

Development
  manifests        Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
  generate         Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
  fmt              Run go fmt against code.
  vet              Run go vet against code.
  test             Run tests.

Build
  build            Build manager binary.
  run              Run a controller from your host.
  docker-build     Build docker image with the manager.
  docker-push      Push docker image with the manager.

Deployment
  install          Install CRDs into the K8s cluster specified in ~/.kube/config.
  uninstall        Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
  deploy           Deploy controller to the K8s cluster specified in ~/.kube/config.
  undeploy         Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
  controller-gen   Download controller-gen locally if necessary.
  kustomize        Download kustomize locally if necessary.
  envtest          Download envtest-setup locally if necessary.
  bundle           Generate bundle manifests and metadata, then validate generated files.
  bundle-build     Build the bundle image.
  bundle-push      Push the bundle image.
  opm              Download opm locally if necessary.
  catalog-build    Build a catalog image.
  catalog-push     Push a catalog image.

Development
  all-dev          Install and run development mode
  install-dev      Install development components
  uninstall-dev    Uninstall development components
  upload-dev       Upload application into development pod
  run-dev          Run application in development pod
```

The most important parts of the Makefile for local development is the last section under `Development`.

## Running the Operator

When running the Operator, the code base needs to be uploaded to a development pod, and then run through `kubetcl exec`.

[Here's why](#why-does-the-operator-run-in-cluster-for-local-development)

TL;DR;

```shell
$ make install-dev # Install development components, CRDs and RBAC. Rerun on changes to CRD / RBAC.
$ make run-dev # Run the operator. Restart on code changes.
```

All the commands have been scripted and are easily runnable through `make`

```shell
$ make install-dev # Install the development deployment, as well as the CRDs and RBAC necessary
$ make upload-dev # Upload the current code to the development pod
$ make run-dev # Run the code inside the pod. This action will run upload-dev automatically before running.
$ make all-dev # Run all of the above
$ make uninstall-dev # Uninstall the development components
```

Whenever you make changes to the CRDs or the RBAC for the operator, you need to rerun `make install-dev` so they 
can be propagated to the Kubernetes cluster.

## Running the tests

```shell
$ make test
```

# FAQ 

## Why does the operator run in-cluster for local development?

When developing the operator, and you would like to run it for testing, the operator needs to run inside the Kubernetes 
cluster. 

This is due to the nature of Redis communication. 

Each Redis node needs to be individually contactable by the Operator, so we can direct the correct commands to the 
correct nodes. 

When running on a local machine, there is no easily configurable way without additional tooling to expose each node 
separately.

For this reason the local development commands such as `run-dev` do not run locally, but they rsync the latest code to 
a development deployment, and then run the operator within that. 

It feels very quick and feels pretty much the same as local development, and this process should not prohibit you from 
running the Operator easily and cleanly.
