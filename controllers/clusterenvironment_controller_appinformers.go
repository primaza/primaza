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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/primaza/primaza/api/v1alpha1"
	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
)

func (r *ClusterEnvironmentReconciler) RunAppInformers(ctx context.Context, cfg *rest.Config, ce v1alpha1.ClusterEnvironment, failedApplicationNamespaces []string) error {
	r.appInformersMux.Lock()
	defer r.appInformersMux.Unlock()

	// calculate service namespaces to watch: "declared" minus "failed"
	snn := map[string]struct{}{}
	for _, n := range ce.Spec.ApplicationNamespaces {
		snn[n] = struct{}{}
	}
	for _, n := range failedApplicationNamespaces {
		delete(snn, n)
	}

	// calculate service namespaces to stop watching: "running" minus "to watch"
	d := map[string]struct{}{}
	for n := range r.appInformers {
		d[n] = struct{}{}
	}
	for n := range snn {
		delete(d, n)
	}

	// stop informers
	for n := range d {
		r.appInformers[n].cancelFunc()
		delete(r.appInformers, n)
	}

	// run new informers
	errs := []error{}
	for n := range snn {
		if err := r.RunAppInformer(ctx, cfg, &ce, n); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (r *ClusterEnvironmentReconciler) RunAppInformer(ctx context.Context, config *rest.Config, ce *v1alpha1.ClusterEnvironment, namespace string) error {
	l := log.FromContext(ctx).WithValues("cluster-environment", ce.Name)
	in := fmt.Sprintf("%s/%s", ce.Name, namespace)

	l.Info("Running informer for service claims")
	// check if informer already exists
	if _, ok := r.appInformers[in]; ok {
		l.Info("Informer already exists", "informer", in)
		return nil
	}

	clusterClient, err := dynamic.NewForConfig(config)
	if err != nil {
		l.Info("failed creating cluster client", "error", err)
		return err
	}

	gvk, err := apiutil.GVKForObject(&primazaiov1alpha1.ControlPlaneServiceClaim{}, r.Scheme)
	if err != nil {
		l.Info("error getting ServiceClaim GroupVersionKind", "error", err)
		return err
	}

	m, err := r.Client.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		l.Info("error getting ServiceClaim mapping", "error", err)
		return err
	}

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(clusterClient, time.Minute, namespace, nil)
	i := factory.ForResource(m.Resource).Informer()

	ictx, fc := context.WithCancel(ctx)
	li := log.FromContext(ictx).WithValues("cluster-environment", ce.Name)

	parseServiceClaim := func(obj interface{}, action string) (*primazaiov1alpha1.ControlPlaneServiceClaim, error) {
		u, ok := obj.(*unstructured.Unstructured)
		if !ok {
			li.Info("'obj' is not an instance of unstructured.Unstructured", "event", action, "obj", obj)
			return nil, fmt.Errorf("obj is not an instance of unstructured.Unstructured")
		}

		rs := primazaiov1alpha1.ControlPlaneServiceClaim{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), &rs); err != nil {
			li.Info("'obj' is not a registered service", "event", action, "unstructured", u, "error", err)
			return nil, err
		}

		return &rs, nil
	}

	var synced atomic.Bool
	synced.Store(false)
	if _, err := i.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			li.Info("add serviceclaim event fetched", "obj", obj)
			if !synced.Load() {
				return
			}
			sc, err := parseServiceClaim(obj, "add")
			if err != nil {
				li.Info("error parsing serviceclaim", "error", err)
				return
			}

			// TODO: we may have some collision or loops here,
			// e.g. when primaza is creating a ServiceClaim that matches above constraints.
			// My suggestion is to create an ApplicationServiceClaim CRD
			// for the Claim from an Application namespace workflow
			if sc.Spec.Target != nil && sc.Spec.Target.EnvironmentTag != "" {
				li.Info("error serviceclaim is not cluster scoped", "serviceclaim", sc)
				return
			}

			csc := &primazaiov1alpha1.ControlPlaneServiceClaim{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ce.Namespace,
					Name:      sc.Name,
				},
			}
			if _, err := controllerutil.CreateOrUpdate(ictx, r.Client, csc, func() error {
				csc.Spec = sc.Spec
				csc.Spec.Target = &primazaiov1alpha1.ControlPlaneServiceClaimTarget{
					ApplicationClusterContext: &primazaiov1alpha1.ServiceClaimApplicationClusterContext{
						ClusterEnvironmentName: ce.Name,
						Namespace:              namespace,
					},
				}
				return nil
			}); err != nil {
				li.Info(
					"error creating the serviceclaim in control plane",
					"clusterenvironment", ce.Name,
					"serviceclaim", sc,
					"error", err)
				return
			}
		},
		UpdateFunc: func(past, future interface{}) {},
		DeleteFunc: func(obj interface{}) {
			li.Info("delete serviceclaim event fetched", "obj", obj)
			if !synced.Load() {
				return
			}

			sc, err := parseServiceClaim(obj, "delete")
			if err != nil {
				li.Info("error parsing serviceclaim", "error", err)
				return
			}

			csc := &primazaiov1alpha1.ControlPlaneServiceClaim{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ce.Namespace,
					Name:      sc.Name,
				},
			}
			if err := r.Client.Delete(ictx, csc); err != nil {
				li.Info(
					"error deleting serviceclaim in control plane",
					"clusterenvironment", ce.Name,
					"serviceclaim", sc,
					"error", err)
				return
			}
		},
	}); err != nil {
		fc()
		return err
	}

	l.Info("run informer", "clusterenvironment", ce.Name, "service namespace", namespace)

	mi := informer{informer: i, ctx: ictx, cancelFunc: fc}
	r.appInformers[in] = mi
	go mi.run()

	if !cache.WaitForCacheSync(ctx.Done(), i.HasSynced) {
		fc()
		delete(r.appInformers, in)
		return fmt.Errorf("could not sync cache")
	}
	synced.Store(true)
	l.Info("informer synced", "clusterenvironment", ce.Name, "application namespace", namespace)

	return nil
}
