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
	"github.com/containersolutions/redis-cluster-operator/internal/kubernetes"
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

	// At this point we have a valid RedisCluster.
	// A Redis cluster needs a StatefulSet to run in.
	// We'll check for an existing Statefulset. If it doesn't exist we'll create one.
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
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, err
	}

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
		return ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, err
	}

	//fmt.Println(statefulset)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.RedisCluster{}).
		Complete(r)
}
