/*
Copyright 2023 The Primaza Authors.

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

	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
)

// ClusterEnvironmentReconciler reconciles a ClusterEnvironment object
type ClusterEnvironmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list
//+kubebuilder:rbac:groups=primaza.io,resources=clusterenvironments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primaza.io,resources=clusterenvironments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primaza.io,resources=clusterenvironments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterEnvironment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ClusterEnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	var ce primazaiov1alpha1.ClusterEnvironment
	if err := r.Client.Get(ctx, req.NamespacedName, &ce); err != nil {
		l.Info("error fetching ClusterEnvironment (deleted)", "error", err)
		return ctrl.Result{}, nil
	}

	res := r.testConnection(ctx, ce)
	if err := r.updateClusterEnvironmentStatus(ctx, ce, res); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

type connectionStatus struct {
	State   primazaiov1alpha1.ClusterEnvironmentState
	Reason  string
	Message string
}

func (r *ClusterEnvironmentReconciler) testConnection(ctx context.Context, ce primazaiov1alpha1.ClusterEnvironment) connectionStatus {
	l := log.FromContext(ctx)

	c, err := r.getClusterClient(ctx, ce)
	if err != nil {
		l.Error(err, "error creating the client", "clusterenvironment", ce)
		return connectionStatus{
			State:   primazaiov1alpha1.ClusterEnvironmentStateOffline,
			Reason:  "ClientCreationError",
			Message: fmt.Sprintf("error creating the client: %s", err),
		}
	}

	v, err := c.ServerVersion()
	if err != nil {
		l.Error(err, "error asking server version")
		return connectionStatus{
			State:   primazaiov1alpha1.ClusterEnvironmentStateOffline,
			Reason:  "ConnectionError",
			Message: fmt.Sprintf("error connecting to target cluster: %s", err),
		}
	}

	l.Info("server version", "version", v)
	return connectionStatus{
		State:   primazaiov1alpha1.ClusterEnvironmentStateOnline,
		Reason:  "ConnectionSuccessful",
		Message: fmt.Sprintf("successfully connected to target cluster: kubernetes version found %s", v),
	}
}

func (r *ClusterEnvironmentReconciler) updateClusterEnvironmentStatus(ctx context.Context, ce primazaiov1alpha1.ClusterEnvironment, cs connectionStatus) error {
	l := log.FromContext(ctx)

	l.Info("updating cluster environment status", "clusterenvironment", ce.GetName(), "connection status", cs)
	ce.Status.State = cs.State
	co := metav1.Condition{
		Type:    string(cs.State),
		Reason:  cs.Reason,
		Message: cs.Message,
		Status:  "True",
	}
	meta.SetStatusCondition(&ce.Status.Conditions, co)
	if err := r.Client.Status().Update(ctx, &ce); err != nil {
		l.Error(err, "error updating cluster environment status", "connection status", cs)
		return err
	}

	return nil
}

func (r *ClusterEnvironmentReconciler) getClusterClient(ctx context.Context, ce primazaiov1alpha1.ClusterEnvironment) (*kubernetes.Clientset, error) {
	kc, err := r.getKubeconfig(ctx, ce)
	if err != nil {
		return nil, err
	}

	cg, err := clientcmd.RESTConfigFromKubeConfig(kc)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(cg)
}

func (r *ClusterEnvironmentReconciler) getKubeconfig(ctx context.Context, ce primazaiov1alpha1.ClusterEnvironment) ([]byte, error) {
	sn := ce.Spec.ClusterContextSecret
	k := client.ObjectKey{Namespace: ce.Namespace, Name: sn}
	var s corev1.Secret
	if err := r.Client.Get(ctx, k, &s); err != nil {
		return nil, err
	}

	return s.Data["kubeconfig"], nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&primazaiov1alpha1.ClusterEnvironment{}).
		Complete(r)
}
