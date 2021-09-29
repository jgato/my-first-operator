/*
Copyright 2021.

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
	"time"

	"fmt"
	myfirstoperatorv1alpha1 "github.com/jgato/my-first-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// MyOwnShellReconciler reconciles a MyOwnShell object
type MyOwnShellReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const myOwnShellFinalizer = "myownshell.my-first-operator.jgato.io/finalizer"

//+kubebuilder:rbac:groups=my-first-operator.jgato.io,resources=myownshells,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=my-first-operator.jgato.io,resources=myownshells/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=my-first-operator.jgato.io,resources=myownshells/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyOwnShell object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *MyOwnShellReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	// Fetch the  instance
	myownshell := &myfirstoperatorv1alpha1.MyOwnShell{}

	err := r.Get(ctx, req.NamespacedName, myownshell)
	log.Info("*Resource info*", "Resource type", myownshell.Kind, "Resource Name", myownshell.Name)
	log.Info("*Req info*", "string", req.String()) // which is the string combination of resource name and namespace

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("MyOwnShell resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get MyOwnShell")
		return ctrl.Result{}, err
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}

	err = r.Get(ctx, types.NamespacedName{Name: myownshell.Name, Namespace: myownshell.Namespace}, found)
	log.Info("*Deployment info*", "deployment name", found.Name, "deployment size", found.Spec.Replicas, "requested size", myownshell.Spec.Size)

	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForMyOwnShell(myownshell)
		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	foundCM := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: myownshell.Name, Namespace: myownshell.Namespace}, foundCM)

	if err != nil && errors.IsNotFound(err) {
		// Define a new configmap
		cm := r.configMapForMyOwnShell(myownshell)
		log.Info("Creating a new ConfigMap", "Deployment.Namespace", cm.Namespace, "Deployment.Name", cm.Name)
		err = r.Create(ctx, cm)
		if err != nil {
			log.Error(err, "Failed to create new ConfigMap", "Deployment.Namespace", cm.Namespace, "Deployment.Name", cm.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	}

	err = r.Get(ctx, types.NamespacedName{Name: myownshell.Name, Namespace: myownshell.Namespace}, found)

	// Ensure the deployment size is the same as the spec
	size := myownshell.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}

		foundCM.Data["replicas"] = fmt.Sprint(size)
		err = r.Update(ctx, foundCM)
		if err != nil {
			log.Error(err, "Failed to update ConfigMap", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Ask to requeue after 1 minute in order to give enough time for the
		// pods be created on the cluster side and the operand be able
		// to do the next update step accurately.
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Check if the Memcached instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isMyOwnShellMarkedToBeDeleted := myownshell.GetDeletionTimestamp() != nil
	isMyFoundMarkedToBeDeleted := found.GetDeletionTimestamp() != nil

	if isMyOwnShellMarkedToBeDeleted || isMyFoundMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(myownshell, myOwnShellFinalizer) {
			// We do some logic

			// We say everything goes well. But we should control
			// if something goes wrong, we dont delete finalizeer
			// in order to retry later
			if false {
				return ctrl.Result{}, err
			}

			// Remove myownshell and deployment finalizer. Once all finalizers have been
			// removed, the object will be deleted.
			if isMyOwnShellMarkedToBeDeleted {
				log.Info("Successfully finalized  MyOwnShell")
				r.Delete(ctx, foundCM)
				if err != nil {
					log.Error(err, "Failed to delete ConfirMap Created by MyOwnShell", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
					return ctrl.Result{}, err
				}
				// Ask
				controllerutil.RemoveFinalizer(myownshell, myOwnShellFinalizer)
				err := r.Update(ctx, myownshell)
				if err != nil {
					return ctrl.Result{}, err
				}
				// if we quit the finalizer of the MyOwnShell, we have to
				// put it away also in the Deployment
				// Otherwise the CR will be deleted, but not the deployment
				// that will keep waiting for the finalizer
				log.Info("Successfully finalized  Deployment")

				controllerutil.RemoveFinalizer(found, myOwnShellFinalizer)
				err = r.Update(ctx, found)
				if err != nil {
					return ctrl.Result{}, err
				}
			}
			if isMyFoundMarkedToBeDeleted {
				log.Info("Successfully finalized  Deployment")
				controllerutil.RemoveFinalizer(found, myOwnShellFinalizer)
				err := r.Update(ctx, found)
				if err != nil {
					return ctrl.Result{}, err
				}
			}

		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(myownshell, myOwnShellFinalizer) {
		controllerutil.AddFinalizer(myownshell, myOwnShellFinalizer)
		err = r.Update(ctx, myownshell)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Add the finalizer for the Deployment
	if !controllerutil.ContainsFinalizer(found, myOwnShellFinalizer) {
		controllerutil.AddFinalizer(found, myOwnShellFinalizer)
		err = r.Update(ctx, found)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil

}

// deploymentForMyOwnShell returns a myownshell Deployment object
func (r *MyOwnShellReconciler) deploymentForMyOwnShell(m *myfirstoperatorv1alpha1.MyOwnShell) *appsv1.Deployment {
	ls := labelsForMyOwnShell(m.Name)
	replicas := m.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:   "busybox:latest",
						Name:    "busybox",
						Command: []string{"sh", "-c", "echo The app is running, You can now ssh into for a while! && sleep 99999999 "},
					}},
				},
			},
		},
	}
	// Set MyOwnShell instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// configMapForMyOwnShell returns a myownshell Deployment object
func (r *MyOwnShellReconciler) configMapForMyOwnShell(m *myfirstoperatorv1alpha1.MyOwnShell) *corev1.ConfigMap {
	replicas := m.Spec.Size
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Data: map[string]string{"replicas": fmt.Sprint(replicas)},
	}

	return cm
}

// labelsForMyOwnShell returns the labels for selecting the resources
// belonging to the given MyOwnShell CR name.
func labelsForMyOwnShell(name string) map[string]string {
	return map[string]string{"app": "myownshell", "myownshell_cr": name}
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyOwnShellReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myfirstoperatorv1alpha1.MyOwnShell{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
