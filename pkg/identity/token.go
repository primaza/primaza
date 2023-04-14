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
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

var ErrSecretMalformed = fmt.Errorf("service account's secret malformed")

type ServiceAccountToken struct {
	CACrt     []byte `json:"ca.crt"`
	Namespace []byte `json:"namespace"`
	Token     []byte `json:"token"`
}

func GetToken(secret *corev1.Secret) (*ServiceAccountToken, error) {
	c, err := getSecretDataField(secret, "ca.crt")
	if err != nil {
		return nil, err
	}

	n, err := getSecretDataField(secret, "namespace")
	if err != nil {
		return nil, err
	}

	t, err := getSecretDataField(secret, "token")
	if err != nil {
		return nil, err
	}

	return &ServiceAccountToken{
		CACrt:     c,
		Namespace: n,
		Token:     t,
	}, nil
}

func getSecretDataField(secret *corev1.Secret, field string) ([]byte, error) {
	d, ok := secret.Data[field]
	if !ok {
		return nil, fmt.Errorf("%w: can not find '%s' in secret '%s/%s'", ErrSecretMalformed, field, secret.Namespace, secret.Name)
	}

	return d, nil
}
