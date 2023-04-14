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

package identity

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	mv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Instance struct {
	Namespace      string   `json:"namespace"`
	ServiceAccount string   `json:"serviceAccount"`
	Secrets        []string `json:"secrets"`
}

func Create(ctx context.Context, cli kubernetes.Clientset, name string, namespace string) (*Instance, error) {
	sa, err := createServiceAccountIfNotExists(ctx, cli, name, namespace)
	if err != nil {
		return nil, err
	}

	sn := createSecretName(name)
	if _, err := createServiceAccountSecret(ctx, cli, sn, namespace, sa); err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, err
	}

	return &Instance{
		Namespace:      namespace,
		ServiceAccount: name,
		Secrets:        []string{sn},
	}, nil
}

func DeleteIfExists(ctx context.Context, cli kubernetes.Clientset, name string, namespace string) error {
	err := cli.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, mv1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func createSecretName(sa string) string {
	return fmt.Sprintf("tkn-%s", sa)
}

func createServiceAccountIfNotExists(ctx context.Context, cli kubernetes.Clientset, name string, namespace string) (*corev1.ServiceAccount, error) {
	c := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	o := mv1.CreateOptions{}
	sa, err := cli.CoreV1().ServiceAccounts(namespace).Create(ctx, c, o)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, err
		}

		return getServiceAccount(ctx, cli, name, namespace)
	}

	return sa, nil
}

func getServiceAccount(ctx context.Context, cli kubernetes.Clientset, name string, namespace string) (*corev1.ServiceAccount, error) {
	return cli.CoreV1().ServiceAccounts(namespace).Get(ctx, name, mv1.GetOptions{})
}

func createServiceAccountSecret(ctx context.Context, cli kubernetes.Clientset, name string, namespace string, sa *corev1.ServiceAccount) (*corev1.Secret, error) {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				corev1.ServiceAccountNameKey: sa.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ServiceAccount",
					Name:       sa.Name,
					UID:        sa.UID,
				},
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}

	return cli.CoreV1().Secrets(namespace).Create(ctx, s, mv1.CreateOptions{})
}
