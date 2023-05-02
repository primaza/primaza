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

// Package sed contains logic for ServiceEndpointDefinition
package sed

import (
	"context"
	"fmt"

	"github.com/primaza/primaza/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/jsonpath"
)

// SEDResourceMapping
type SEDResourceMapping struct {
	resource unstructured.Unstructured

	key    string
	path   *jsonpath.JSONPath
	secret bool
}

func NewSEDResourceMapping(resource unstructured.Unstructured, mapping v1alpha1.ServiceClassResourceFieldMapping) (*SEDResourceMapping, error) {
	path := jsonpath.New("")
	err := path.Parse(fmt.Sprintf("{%s}", mapping.JsonPath))
	if err != nil {
		return nil, err
	}

	return &SEDResourceMapping{
		resource: resource,
		key:      mapping.Name,
		path:     path,
		secret:   mapping.Secret,
	}, nil
}

func (s *SEDResourceMapping) Key() string {
	return s.key
}

func (mapping *SEDResourceMapping) ReadKey(ctx context.Context) (*string, error) {
	results, err := mapping.path.FindResults(mapping.resource.Object)
	if err != nil {
		return nil, err
	}

	if len(results) != 1 || len(results[0]) != 1 {
		return nil, fmt.Errorf("jsonPath lookup into resource returned multiple results: %v", results)
	}

	value := fmt.Sprintf("%v", results[0][0])
	return &value, nil
}

func (s *SEDResourceMapping) InSecret() bool {
	return s.secret
}
