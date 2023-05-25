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
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/primaza/primaza/api/v1alpha1"
	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"github.com/primaza/primaza/pkg/primaza/clustercontext"
)

func (r *ClusterEnvironmentReconciler) RunSvcInformers(ctx context.Context, cfg *rest.Config, ce v1alpha1.ClusterEnvironment, failedServiceNamespaces []string) error {
	r.svcInformersMux.Lock()
	defer r.svcInformersMux.Unlock()

	// calculate service namespaces to watch: "declared" minus "failed"
	snn := map[string]struct{}{}
	for _, n := range ce.Spec.ServiceNamespaces {
		snn[n] = struct{}{}
	}
	for _, n := range failedServiceNamespaces {
		delete(snn, n)
	}

	// calculate service namespaces to stop watching: "running" minus "to watch"
	d := map[string]struct{}{}
	for n := range r.svcInformers {
		d[n] = struct{}{}
	}
	for n := range snn {
		delete(d, n)
	}

	// stop informers
	for n := range d {
		r.svcInformers[n].cancelFunc()
		delete(r.svcInformers, n)
	}

	// run new informers
	errs := []error{}
	for n := range snn {
		if err := r.RunSvcInformer(ctx, cfg, ce.GetName(), ce.GetNamespace(), n); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (r *ClusterEnvironmentReconciler) RunSvcInformer(ctx context.Context, config *rest.Config, ceName string, ceNamespace, namespace string) error {
	l := log.FromContext(ctx)
	in := fmt.Sprintf("%s/%s", ceName, namespace)

	// check if informer already exists
	if _, ok := r.svcInformers[in]; ok {
		l.Info("Informer already exists", "clusterenvironment", ceName, "informer", in)
		return nil
	}

	clusterClient, err := dynamic.NewForConfig(config)
	if err != nil {
		l.Info("failed creating cluster client", "error", err)
		return err
	}

	gvk, err := apiutil.GVKForObject(&primazaiov1alpha1.RegisteredService{}, r.Scheme)
	if err != nil {
		l.Info("error getting RegisteredService GroupVersionKind", "error", err)
		return err
	}

	m, err := r.Client.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		l.Info("error getting registeredservice mapping", "error", err)
		return err
	}

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(clusterClient, time.Minute, namespace, nil)
	i := factory.ForResource(m.Resource).Informer()

	ictx, fc := context.WithCancel(ctx)
	li := log.FromContext(ictx)

	var synced atomic.Bool
	synced.Store(false)
	if _, err := i.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			li.Info("add registeredservice event fetched", "obj", obj)
			rs, err := r.parseRegisteredService(ictx, obj, "add")
			if err != nil {
				li.Info("error parsing registered service", "error", err)
				return
			}

			if !synced.Load() {
				li.Info("cache not synched, skipping registered service add event", "registeredservice", rs)
				return
			}

			li.Info("registered service add event", "registeredservice", rs)
			r.createUpdate(ictx, rs, ceName, ceNamespace, "create")
		},
		UpdateFunc: func(past, future interface{}) {
			li.Info("update registeredservice event fetched", "future", future)
			rs, err := r.parseRegisteredService(ictx, future, "update")
			if err != nil {
				li.Info("error parsing registered service", "error", err)
				return
			}

			if !synced.Load() {
				li.Info("cache not synched, skipping registered service update event", "registeredservice", rs)
				return
			}

			li.Info("registered service update event", "registeredservice", rs)
			r.createUpdate(ictx, rs, ceName, ceNamespace, "update")
		},
		DeleteFunc: func(obj interface{}) {
			li.Info("delete registeredservice event fetched", "obj", obj)
			rs, err := r.parseRegisteredService(ictx, obj, "delete")
			if err != nil {
				li.Info("error parsing registered service", "error", err)
				return
			}

			if !synced.Load() {
				li.Info("cache not synched, skipping registered service", "registeredservice", rs)
				return
			}

			li.Info("Deleting registered service in control plane", "registeredservice", rs)
			rs.Namespace = ceNamespace
			if err := r.Client.Delete(ictx, rs); err != nil {
				li.Info("error deleting registered service in Control Plane", "clusterenvironment", ceName, "registeredservice", rs, "error", err)
				return
			}
			li.Info("Deleted registered service in control plane", "registeredservice", rs)

			s := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: ceNamespace, Name: rs.Name + "-descriptor"}}
			if err := r.Client.Delete(ictx, s); err != nil && !apierrors.IsNotFound(err) {
				li.Info("error deleting registered service's secret in Control Plane", "clusterenvironment", ceName, "secret", s, "error", err)
				return
			}
		},
	}); err != nil {
		fc()
		return err
	}

	l.Info("run informer", "clusterenvironment", ceName, "service namespace", namespace)
	mi := informer{informer: i, ctx: ictx, cancelFunc: fc}
	r.svcInformers[in] = mi
	go mi.run()

	if !cache.WaitForCacheSync(ctx.Done(), i.HasSynced) {
		fc()
		delete(r.svcInformers, in)
		return fmt.Errorf("could not sync cache")
	}
	synced.Store(true)
	l.Info("informer synced", "clusterenvironment", ceName, "service namespace", namespace)

	return nil
}

func (r *ClusterEnvironmentReconciler) parseRegisteredService(ctx context.Context, obj interface{}, action string) (*primazaiov1alpha1.RegisteredService, error) {
	li := log.FromContext(ctx)
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		li.Info("'obj' is not an instance of unstructured.Unstructured", "event", action, "obj", obj)
		return nil, fmt.Errorf("obj is not an instance of unstructured.Unstructured")
	}

	rs := primazaiov1alpha1.RegisteredService{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), &rs); err != nil {
		li.Info("'obj' is not a registered service", "event", action, "unstructured", u, "error", err)
		return nil, err
	}

	return &rs, nil
}

func (r *ClusterEnvironmentReconciler) bakeClient(ctx context.Context, ceName, ceNamespace string) (client.Client, error) {
	li := log.FromContext(ctx)
	ce := primazaiov1alpha1.ClusterEnvironment{}
	k := types.NamespacedName{Namespace: ceNamespace, Name: ceName}
	if err := r.Client.Get(ctx, k, &ce); err != nil {
		li.Info("error retrieving clusterenvironment for service namespace sync", "clusterenvironment", ceName, "error", err)
		return nil, err
	}

	cli, err := clustercontext.CreateClient(ctx, r.Client, ce, r.Scheme, r.RESTMapper())
	if err != nil {
		li.Info("error building client for service namespace sync", "clusterenvironment", ceName, "error", err)
		return nil, err
	}
	return cli, err
}

func (r *ClusterEnvironmentReconciler) createUpdate(ctx context.Context, rs *primazaiov1alpha1.RegisteredService, ceName, ceNamespace, action string) {
	li := log.FromContext(ctx)
	curs := &primazaiov1alpha1.RegisteredService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ceNamespace,
			Name:      rs.Name,
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, curs, func() error {
		curs.Spec = rs.Spec
		return nil
	}); err != nil {
		li.Info(
			"error creating or updating the registeredservice",
			"action", action,
			"clusterenvironment", ceName,
			"error", err)
		return
	}

	// fetch registered service's secret
	cli, err := r.bakeClient(ctx, ceName, ceNamespace)
	if err != nil {
		li.Info(
			"error creating the client for fetching the registeredservice secret",
			"action", action,
			"clusterenvironment", ceName,
			"error", err)
		return
	}

	s := corev1.Secret{}
	sk := types.NamespacedName{Namespace: rs.Namespace, Name: rs.Name + "-descriptor"}
	if err := cli.Get(ctx, sk, &s); err != nil {
		li.Info(
			"error fetching registeredservice's secret",
			"error", err,
			"registeredservice", rs,
			"secret", sk)
		return
	}

	cs := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: ceNamespace, Name: s.Name}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, &cs, func() error {
		cs.Data = s.Data
		return nil
	}); err != nil {
		li.Info(
			"error creating/updating a registeredservice's secret",
			"error", err,
			"registeredservice", rs,
			"secret", cs)
		return
	}
}
