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

package clustercontext

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"

	primazaiov1alpha1 "github.com/primaza/primaza/api/v1alpha1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrSecretNotFound = fmt.Errorf("Cluster Context Secret not found")

func CreateClient(
	ctx context.Context,
	primazaCli client.Client,
	ce primazaiov1alpha1.ClusterEnvironment,
	scheme *runtime.Scheme,
	mapper meta.RESTMapper,
) (client.Client, error) {
	cfg, err := GetClusterRESTConfig(ctx, primazaCli, ce.Namespace, ce.Spec.ClusterContextSecret)
	if err != nil {
		return nil, err
	}

	oc := client.Options{
		Scheme: scheme,
		Mapper: mapper,
	}
	cli, err := client.New(cfg, oc)
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func GetClusterRESTConfig(ctx context.Context, cli client.Client, secretNamespace, secretName string) (*rest.Config, error) {
	s, err := getSecret(ctx, cli, secretNamespace, secretName)
	if err != nil {
		return nil, err
	}

	return clientcmd.RESTConfigFromKubeConfig(s.Data["kubeconfig"])
}

func getSecret(ctx context.Context, cli client.Client, secretNamespace, secretName string) (*corev1.Secret, error) {
	s := &corev1.Secret{}
	k := client.ObjectKey{Namespace: secretNamespace, Name: secretName}
	if err := cli.Get(ctx, k, s); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, errors.Join(ErrSecretNotFound, err)
		}
		return nil, err
	}

	return s, nil
}
