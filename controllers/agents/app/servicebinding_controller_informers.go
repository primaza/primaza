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
	"slices"
	"time"

	"github.com/primaza/primaza/api/v1alpha1"
	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"go.uber.org/atomic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type informer struct {
	informer   cache.SharedIndexInformer
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func (i *informer) run() {
	i.informer.Run(i.ctx.Done())
}

func (r *ServiceBindingReconciler) ensureInformerIsRunningForServiceBinding(ctx context.Context, serviceBinding v1alpha1.ServiceBinding) error {
	reconcileLog := log.FromContext(ctx)
	typemeta := metav1.TypeMeta{
		Kind:       serviceBinding.Spec.Application.Kind,
		APIVersion: serviceBinding.Spec.Application.APIVersion,
	}
	gvk := typemeta.GroupVersionKind()
	mapping, err := r.Client.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		reconcileLog.Error(err, "error on creating mapping")
		return err
	}
	reconcileLog.Info("resource to be watched", "resource", mapping.Resource)
	if err = r.ensureInformerIsRunningForResource(ctx, mapping.Resource, serviceBinding); err != nil {
		reconcileLog.Error(err, "error running informer")
		return err
	}
	return nil
}

func (r *ServiceBindingReconciler) ensureInformerIsRunningForResource(ctx context.Context, resource schema.GroupVersionResource, serviceBinding v1alpha1.ServiceBinding) error {
	l := log.FromContext(ctx)

	// check if informer already exists
	if _, ok := r.informers[serviceBinding.GetName()]; ok {
		l.Info("Informer already exists")
		return nil
	}

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		l.Info("failed creating cluster config")
		return err
	}
	clusterClient, err := dynamic.NewForConfig(clusterConfig)
	if err != nil {
		l.Info("failed creating cluster config")
		return err
	}
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(clusterClient, time.Minute, serviceBinding.Namespace, nil)
	i := factory.ForResource(resource).Informer()

	var synced atomic.Bool
	synced.Store(false)

	l.Info("run informer", "GroupVersionResource", resource)
	c, fc := context.WithCancel(ctx)

	if err := r.addEventHandler(c, i, &synced, serviceBinding); err != nil {
		fc()
		return err
	}

	li := informer{informer: i, ctx: c, cancelFunc: fc}
	r.informers[serviceBinding.GetName()] = li
	go li.run()

	if !cache.WaitForCacheSync(ctx.Done(), i.HasSynced) {
		fc()
		delete(r.informers, serviceBinding.GetName())
		return fmt.Errorf("could not sync cache")
	}

	synced.Store(true)

	return nil
}

func (r *ServiceBindingReconciler) addEventHandler(ctx context.Context, index cache.SharedIndexInformer, synced *atomic.Bool, serviceBinding v1alpha1.ServiceBinding) error {
	_, err := index.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    r.prepareAddFunc(ctx, synced, serviceBinding),
		UpdateFunc: r.prepareUpdateFunc(ctx, synced, serviceBinding),
		DeleteFunc: r.prepareDeleteFunc(ctx, synced, serviceBinding),
	})
	return err
}

func (r *ServiceBindingReconciler) prepareAddFunc(ctx context.Context, synced *atomic.Bool, serviceBinding primazaiov1alpha1.ServiceBinding) func(obj interface{}) {
	return func(obj interface{}) {
		if !synced.Load() {
			return
		}

		l := log.FromContext(ctx).WithValues("service binding", serviceBinding.Name)

		applicationResource := obj.(*unstructured.Unstructured)
		if !verifyApplicationSatisfiesServiceBindingSpec(applicationResource, serviceBinding) {
			return
		}
		l.Info("application resource", "application", applicationResource.GetName())

		psSecret, err := r.GetSecret(ctx, serviceBinding, *applicationResource)
		if err != nil {
			l.Error(err, "Informer AddEventHandler: Error retrieving secret")
			return
		}
		var sb primazaiov1alpha1.ServiceBinding
		k := types.NamespacedName{Namespace: serviceBinding.Namespace, Name: serviceBinding.Name}
		if err := r.Get(ctx, k, &sb, &client.GetOptions{}); err != nil {
			l.Error(err, "Informer AddEventHandler: retrieving ServiceBinding", "service binding", k)
			return
		}

		if err := r.PrepareBinding(ctx, &sb, psSecret, *applicationResource); err != nil {
			l.Error(err, "Informer AddEventHandler: Error preparing binding")
			return
		}
	}
}

func (r *ServiceBindingReconciler) prepareUpdateFunc(ctx context.Context, synced *atomic.Bool, serviceBinding primazaiov1alpha1.ServiceBinding) func(past, future interface{}) {
	return func(past, future interface{}) {
		if !synced.Load() {
			return
		}

		l := log.FromContext(ctx).WithValues("service binding", serviceBinding.Name)
		l.Info("watched resource updated")

		applicationResource := future.(*unstructured.Unstructured)

		l.Info("application resource", "application", applicationResource.GetName())
		if !verifyApplicationSatisfiesServiceBindingSpec(applicationResource, serviceBinding) {
			return
		}
		psSecret, err := r.GetSecret(ctx, serviceBinding, *applicationResource)
		if err != nil {
			l.Error(err, "Informer AddEventHandler: Error retrieving secret")
			return
		}

		var sb primazaiov1alpha1.ServiceBinding
		k := types.NamespacedName{Namespace: serviceBinding.Namespace, Name: serviceBinding.Name}
		if err := r.Get(ctx, k, &sb, &client.GetOptions{}); err != nil {
			l.Error(err, "Informer AddEventHandler: retrieving ServiceBinding", "service binding", k)
			return
		}

		if err := r.PrepareBinding(ctx, &sb, psSecret, *applicationResource); err != nil {
			l.Error(err, "Informer AddEventHandler: Error preparing binding")
			return
		}
	}
}

func (r *ServiceBindingReconciler) prepareDeleteFunc(ctx context.Context, synced *atomic.Bool, serviceBinding primazaiov1alpha1.ServiceBinding) func(obj interface{}) {
	return func(obj interface{}) {
		if !synced.Load() {
			return
		}

		l := log.FromContext(ctx).WithValues("service binding", serviceBinding.Name)
		l.Info("watched resource deleted")

		sb := primazaiov1alpha1.ServiceBinding{}
		k := types.NamespacedName{Namespace: serviceBinding.Namespace, Name: serviceBinding.Name}
		if err := r.Get(ctx, k, &sb); err != nil {
			l.Error(err, "error retrieving latest revision of ServiceBinding")
			return
		}

		applicationResource := obj.(*unstructured.Unstructured)
		if !slices.ContainsFunc(
			sb.Status.Connections,
			func(w primazaiov1alpha1.BoundWorkload) bool { return w.Name == applicationResource.GetName() }) {
			return
		}

		// update ServiceBinding's Connections
		cc := []primazaiov1alpha1.BoundWorkload{}
		for _, b := range sb.Status.Connections {
			if b.Name != applicationResource.GetName() {
				cc = append(cc, b)
			}
		}
		sb.Status.Connections = cc

		// applications are deleted, so setting the service binding status to false and reconcile
		s := primazaiov1alpha1.ServiceBindingStateReady
		c := metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               primazaiov1alpha1.ServiceBindingBoundCondition,
			Status:             metav1.ConditionFalse,
			Reason:             conditionGetAppsFailureReason,
			Message:            "application was bound with the secret but the application got deleted",
		}
		if err := r.updateServiceBindingStatus(ctx, &sb, c, s); err != nil {
			l.Error(err, "error on updating status")
		}
	}
}
