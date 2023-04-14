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
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type GetKubeconfigOptions struct {
	User      *string
	Namespace *string
}

func GetKubeconfig(token *ServiceAccountToken, host string, opts GetKubeconfigOptions) ([]byte, error) {
	cl := map[string]*clientcmdapi.Cluster{
		"default-cluster": {
			Server:                   host,
			CertificateAuthorityData: token.CACrt,
		},
	}

	un := "default-user"
	if opts.User != nil {
		un = *opts.User
	}
	ai := map[string]*clientcmdapi.AuthInfo{
		un: {
			Token: string(token.Token),
		},
	}

	ct := map[string]*clientcmdapi.Context{
		"default-context": {
			Cluster:  "default-cluster",
			AuthInfo: un,
		},
	}
	if opts.Namespace != nil {
		ct["default-context"].Namespace = *opts.Namespace
	}

	cc := clientcmdapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       cl,
		Contexts:       ct,
		AuthInfos:      ai,
		CurrentContext: "default-context",
	}

	return clientcmd.Write(cc)
}
