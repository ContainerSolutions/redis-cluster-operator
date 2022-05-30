/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"github.com/containersolutions/redis-cluster-operator/internal/kubernetes"
	redis_internal "github.com/containersolutions/redis-cluster-operator/internal/redis"
	"github.com/containersolutions/redis-cluster-operator/internal/utils"
	"github.com/go-redis/redis/v8"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "github.com/containersolutions/redis-cluster-operator/api/v1alpha1"
)

// RedisClusterReconciler reconciles a RedisCluster object
type RedisClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cache.container-solutions.com,resources=redisclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.container-solutions.com,resources=redisclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.container-solutions.com,resources=redisclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RedisCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *RedisClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling RedisCluster", "cluster", req.Name, "namespace", req.Namespace)

	redisCluster := &cachev1alpha1.RedisCluster{}
	err := r.Client.Get(ctx, req.NamespacedName, redisCluster)

	if err != nil {
		if errors.IsNotFound(err) {
			// The RedisCluster was probably deleted. Therefore we can skip reconciling, and trust Kubernetes to delete the resources
			logger.Info("RedisCluster not found during reconcile. Probably deleted by user. Exiting early.")
			return ctrl.Result{}, nil
		}
	}

	//region Ensure ConfigMap
	configMap, err := kubernetes.FetchExistingConfigMap(ctx, r.Client, redisCluster)
	if err != nil && !errors.IsNotFound(err) {
		// We've got a legitimate error, we should log the error and exit early
		logger.Error(err, "Could not check whether configmap exists due to error.")
		return ctrl.Result{
			RequeueAfter: 30 * time.Second,
		}, err
	}
	if errors.IsNotFound(err) {
		configMap, err = kubernetes.CreateConfigMap(ctx, r.Client, redisCluster)
		if err != nil {
			logger.Error(err, "Failed to create ConfigMap for RedisCluster")
			return ctrl.Result{
				RequeueAfter: 30 * time.Second,
			}, err
		}

		logger.Info("Created ConfigMap for RedisCluster. Reconciling in 5 seconds.")
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, err
	}
	//endregion

	//region Set ConfigMap owner reference
	err = retry.RetryOnConflict(wait.Backoff{
		Steps:    5,
		Duration: 2 * time.Second,
		Factor:   1.0,
		Jitter:   0.1,
	}, func() error {
		configMap, err = kubernetes.FetchExistingConfigMap(ctx, r.Client, redisCluster)
		if err != nil {
			// At this point we definitely expect the statefulset to exist.
			logger.Error(err, "Cannot find configMap")
			return err
		}
		err = ctrl.SetControllerReference(redisCluster, configMap, r.Scheme)
		if err != nil {
			logger.Error(err, "Could not set owner reference for configMap")
			return err
		}
		err = r.Client.Update(ctx, configMap)
		if err != nil {
			logger.Error(err, "Could not update configmap with owner reference")
		}
		return err
	})
	if err != nil {
		logger.Error(err, "Could not set owner reference for statefulset")
		return ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, err
	}
	//endregion

	//region Ensure Statefulset
	statefulset, err := kubernetes.FetchExistingStatefulset(ctx, r.Client, redisCluster)
	if err != nil && !errors.IsNotFound(err) {
		// We've got a legitimate error, we should log the error and exit early
		logger.Error(err, "Could not check whether statefulset exists due to error.")
		return ctrl.Result{
			RequeueAfter: 30 * time.Second,
		}, err
	}

	if errors.IsNotFound(err) {
		// We need to create the Statefulset
		statefulset, err = kubernetes.CreateStatefulset(ctx, r.Client, redisCluster)
		if err != nil {
			logger.Error(err, "Failed to create Statefulset for RedisCluster")
			return ctrl.Result{
				RequeueAfter: 30 * time.Second,
			}, err
		}

		// We've created the Statefulset, and we can wait a bit before trying to do the rest.
		// We can trigger a new reconcile for this object in about 5 seconds
		logger.Info("Created Statefulset for RedisCluster. Reconciling in 5 seconds.")
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, err
	}
	//endregion

	//region Set Statefsulset owner reference
	err = retry.RetryOnConflict(wait.Backoff{
		Steps:    5,
		Duration: 2 * time.Second,
		Factor:   1.0,
		Jitter:   0.1,
	}, func() error {
		statefulset, err = kubernetes.FetchExistingStatefulset(ctx, r.Client, redisCluster)
		if err != nil {
			// At this point we definitely expect the statefulset to exist.
			logger.Error(err, "Cannot find statefulset")
			return err
		}
		err = ctrl.SetControllerReference(redisCluster, statefulset, r.Scheme)
		if err != nil {
			logger.Error(err, "Could not set owner reference for statefulset")
			return err
		}
		err = r.Client.Update(ctx, statefulset)
		if err != nil {
			logger.Error(err, "Could not update statefulset with owner reference")
		}
		return err
	})
	if err != nil {
		logger.Error(err, "Could not set owner reference for statefulset")
		return ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, err
	}
	//endregion

	if *statefulset.Spec.Replicas < redisCluster.NodesNeeded() {
		// The statefulset has less replicas than are needed for the cluster.
		// This means the user is trying to scale up the cluster, and we need to scale up the statefulset
		// and let th reconciliation take care of stabilising the cluster.
		logger.Info("Scaling up statefulset for Redis Cluster")
		replicas := redisCluster.NodesNeeded()
		statefulset.Spec.Replicas = &replicas
		err = r.Client.Update(ctx, statefulset)
		if err != nil {
			return r.RequeueError(ctx, "Could not update statefulset replicas", err)
		}
		// We've successfully updated the replicas for the statefulset.
		// Now we can wait for the pods to come up and then continue on the
		// normal process for stabilising the Redis Cluster
		logger.Info("Scaling up statefulset for Redis Cluster successful. Reconciling again in 5 seconds.")
		return ctrl.Result{
			RequeueAfter: 5,
		}, nil
	}

	pods, err := kubernetes.FetchRedisPods(ctx, r.Client, redisCluster)
	if err != nil {
		return r.RequeueError(ctx, "Could not fetch pods for redis cluster", err)
	}

	clusterNodes := redis_internal.ClusterNodes{}
	for _, pod := range pods.Items {
		if utils.IsPodReady(&pod) {
			node, err := redis_internal.NewNode(ctx, &redis.Options{
				Addr: pod.Status.PodIP + ":6379",
			}, redis.NewClient)
			if err != nil {
				return r.RequeueError(ctx, "Could not load Redis Client", err)
			}

			// make sure that the node knows about itself
			// This is necessary, as the nodes often startup without being able to retrieve their own IP address
			err = node.Client.ClusterMeet(ctx, pod.Status.PodIP, "6379").Err()
			if err != nil {
				return r.RequeueError(ctx, "Could not let node meet itself", err)
			}
			clusterNodes.Nodes = append(clusterNodes.Nodes, node)
		}
	}

	allPodsReady := len(clusterNodes.Nodes) == int(redisCluster.NodesNeeded())
	if !allPodsReady {
		logger.Info("Not all pods are ready. Reconciling again in 10 seconds")
		return ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	if allPodsReady {
		// region Ensure Cluster Meet

		// todo we should check whether a cluster meet is necessary before just spraying cluster meets.
		// This can also be augmented by doing cluster meet for all ready nodes, and ignoring any none ready ones.
		// If the amount of ready pods is equal to the amount of nodes needed, we probably have some additional nodes we need to remove.
		// We can forget these additional nodes, as they are probably nodes which pods got killed.
		logger.Info("Meeting Redis nodes")
		err = clusterNodes.ClusterMeet(ctx)
		if err != nil {
			return r.RequeueError(ctx, "Could not meet all nodes together", err)
		}
		// We'll wait for 10 seconds to ensure the meet is propagated
		time.Sleep(time.Second * 5)
		// endregion

		logger.Info("Checking Cluster Master Replica Ratio")
		// region Ensure Cluster Replication Ratio
		err = clusterNodes.EnsureClusterReplicationRatio(ctx, redisCluster)
		if err != nil {
			return r.RequeueError(ctx, "Failed to ensure cluster ratio for cluster", err)
		}
		// endregion

		err = clusterNodes.ReloadNodes(ctx)
		if err != nil {
			return r.RequeueError(ctx, "Failed to reload node info for cluster", err)
		}

		// region Assign Slots
		logger.Info("Assigning Missing Slots")
		slotsAssignments := clusterNodes.CalculateSlotAssignment()
		for node, slots := range slotsAssignments {
			if len(slots) == 0 {
				continue
			}
			var slotsInt []int
			for _, slot := range slots {
				slotsInt = append(slotsInt, int(slot))
			}
			err = node.ClusterAddSlots(ctx, slotsInt...).Err()
			if err != nil {
				return r.RequeueError(ctx, "Could not assign node slots", err)
			}
		}
		// endregion

		logger.Info("Forgetting Failed Nodes No Longer Valid")
		failingNodes, err := clusterNodes.GetFailingNodes(ctx)
		if err != nil {
			return r.RequeueError(ctx, "could not fetch failing nodes", err)
		}
		for _, node := range failingNodes {
			err = clusterNodes.ForgetNode(ctx, node)
			if err != nil {
				return r.RequeueError(ctx, fmt.Sprintf("could not forget node %s", node.NodeAttributes.ID), err)
			}
		}
	}

	return ctrl.Result{
		RequeueAfter: 30 * time.Second,
	}, nil
}

func (r *RedisClusterReconciler) RequeueError(ctx context.Context, message string, err error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Error(err, message)
	return ctrl.Result{
		RequeueAfter: 10 * time.Second,
	}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.RedisCluster{}).
		Complete(r)
}
