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
	"time"

	"github.com/akashjain971/flipper/api/flipper.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// FlipperReconciler reconciles a Flipper object
type FlipperReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	*kubernetes.Clientset
}

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;patch
//+kubebuilder:rbac:groups=flipper.io,resources=flippers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=flipper.io,resources=flippers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=flipper.io,resources=flippers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Flipper object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *FlipperReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var flipper v1alpha1.Flipper
	if err := r.Get(ctx, req.NamespacedName, &flipper); err != nil {
		logger.Error(err, "failed to get flipper")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	setFlipperDefaults(&flipper.Spec)

	logger = logger.WithValues("filpper spec", flipper.Spec)

	flipperInterval, err := time.ParseDuration(flipper.Spec.Interval)
	if err != nil {
		logger.Error(err, "failed to parse flipper interval")
		return ctrl.Result{}, err
	}

	reconcileFrequency, err := time.ParseDuration(flipper.Spec.ReconcileFrequency)
	if err != nil {
		logger.Error(err, "failed to parse reconcile fequency")
		return ctrl.Result{}, err
	}

	for _, namespace := range flipper.Spec.Match.Namespaces {
		logger = logger.WithValues("current namespace", namespace)

		deployments, err := r.Clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(flipper.Spec.Match.Labels).String(),
		})

		if err != nil {
			logger.Error(err, "failed to list deployments")
			continue
		}

		if len(deployments.Items) == 0 {
			logger.Info("no deployments in namespace match the flipper labels")
			continue
		}

		for _, deployment := range deployments.Items {
			logger = logger.WithValues("deployment", deployment.Name)

			restartedAt, err := getRestartedAtTime(deployment)
			if err != nil {
				logger.Error(err, "failed to get restartedAt time for deployment")
				continue
			}

			if !isRestartDue(restartedAt, flipperInterval) {
				continue // it is still not time for next restart
			}

			logger.Info("it is time to restart the deployment")

			if err := r.updateRestartedAt(ctx, deployment.Name, deployment.Namespace); err != nil {
				logger.Error(err, "failed to rollout restart deployment")
				continue
			}

			logger.Info("deployment rollout restart initiated")
		}
	}

	return ctrl.Result{RequeueAfter: reconcileFrequency}, nil

}

func (r *FlipperReconciler) updateRestartedAt(ctx context.Context, name, namespace string) error {
	data := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"%s":"%s"}}}}}`, RestartedAtAnnotation, time.Now().UTC().Format(time.RFC3339))
	_, err := r.Clientset.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(data), metav1.PatchOptions{})
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlipperReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Flipper{}).
		Complete(r)
}
