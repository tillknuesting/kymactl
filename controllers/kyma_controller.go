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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	inventoryv1alpha1 "github.com/kyma-incubator/kymactl/api/v1alpha1"
)

// KymaReconciler reconciles a Kyma object
type KymaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func IgnoreAlreadyExists(err error) error {
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func IgnoreStatusUpdateConflict(err error) error {
	if apierrors.IsConflict(err) {
		return nil
	}
	return err
}

//+kubebuilder:rbac:groups=inventory.kyma-project.io,resources=kymas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inventory.kyma-project.io,resources=kymas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inventory.kyma-project.io,resources=kymas/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Kyma object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile

func (r *KymaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.V(2).Info("Kyma reconciliation happened")

	var components inventoryv1alpha1.HelmComponentList
	if err := r.List(ctx, &components, client.InNamespace(req.Namespace), client.MatchingFields{componentOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child components")
		return ctrl.Result{}, err
	}

	var kyma inventoryv1alpha1.Kyma
	if err := r.Get(ctx, req.NamespacedName, &kyma); err != nil {
		if apierrors.IsNotFound(err) {
			// Kyma not found - delete all components
			for _, c := range components.Items {
				r.Delete(ctx, &c)
			}
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	constructComponentForKyma := func(kyma *inventoryv1alpha1.Kyma, module inventoryv1alpha1.ComponentSpec) (*inventoryv1alpha1.HelmComponent, error) {
		name := fmt.Sprintf("%s-%s", kyma.Name, module.Name)

		component := &inventoryv1alpha1.HelmComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: kyma.Namespace,
			},
			Spec: inventoryv1alpha1.HelmComponentSpec{ComponentName: module.Name},
		}

		if err := ctrl.SetControllerReference(kyma, component, r.Scheme); err != nil {
			return nil, err
		}

		return component, nil
	}

	// Finding modules to create
	kyma.Status.WaitingFor = []string{}
	for _, m := range kyma.Spec.Components {
		found := false
		for _, c := range components.Items {
			if c.Spec.ComponentName == m.Name {
				found = true
				if c.Status.Status != "success" {
					kyma.Status.WaitingFor = append(kyma.Status.WaitingFor, m.Name)
				}
				break
			}
		}
		if !found {
			kyma.Status.WaitingFor = append(kyma.Status.WaitingFor, m.Name)
			log.Info("Create module", "name", m.Name)
			component, err := constructComponentForKyma(&kyma, m)
			if err != nil {
				log.Error(err, "unable to construct component")
				// don't bother requeuing until we get a change to the spec
				return ctrl.Result{}, nil
			}

			if err := r.Create(ctx, component); err != nil {
				log.Error(err, "unable to create Helm component", "component", component)
				return ctrl.Result{RequeueAfter: 5 * time.Second}, IgnoreAlreadyExists(err)
			}
		}
	}

	oldStatus := kyma.Status.Status
	// Update status
	if len(kyma.Status.WaitingFor) == 0 {
		kyma.Status.Status = "success"
	} else {
		kyma.Status.Status = "reconciling"
	}

	// Do not update status if it was success before
	if oldStatus != kyma.Status.Status || kyma.Status.Status == "reconciling" {
		if err := r.Status().Update(ctx, &kyma); err != nil {
			return ctrl.Result{}, IgnoreStatusUpdateConflict(err)
		}
	}
	log.V(2).Info("Status update", "count", len(components.Items), "waiting for", len(kyma.Status.WaitingFor))

	// Delete orphan modules (removed from kyma.Spec)
	for _, c := range components.Items {
		found := false
		for _, m := range kyma.Spec.Components {
			if m.Name == c.Spec.ComponentName {
				found = true
				break
			}
		}
		if !found {
			log.Info("Delete module", "name", c.Name)
			if err := r.Delete(ctx, &c); err != nil {
				log.Error(err, "unable to delete component", "component", c.Name)
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

var (
	componentOwnerKey = ".metadata.controller"
	apiGVStr          = inventoryv1alpha1.GroupVersion.String()
)

// SetupWithManager sets up the controller with the Manager.
func (r *KymaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// set up a real clock, since we're not in a test

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &inventoryv1alpha1.HelmComponent{}, componentOwnerKey, func(rawObj client.Object) []string {
		// grab the job object, extract the owner...
		helmComponent := rawObj.(*inventoryv1alpha1.HelmComponent)
		owner := metav1.GetControllerOf(helmComponent)
		if owner == nil {
			return nil
		}
		// ...make sure it's a CronJob...
		if owner.APIVersion != apiGVStr || owner.Kind != "Kyma" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&inventoryv1alpha1.Kyma{}).
		Owns(&inventoryv1alpha1.HelmComponent{}).
		//		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10, RateLimiter: CustomRateLimiter()}).
		Complete(r)
}
