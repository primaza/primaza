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

package envtag_test

import (
	"testing"

	"github.com/primaza/primaza/pkg/envtag"
)

func Test_MatchEmptyContainsts(t *testing.T) {
	type test struct {
		environment string
		constraints []string
		want        bool
	}

	tt := []test{
		{environment: "", constraints: nil, want: true},
		{environment: "", constraints: []string{}, want: true},
		{environment: "env", constraints: nil, want: true},
		{environment: "env", constraints: []string{}, want: true},
		{environment: "env", constraints: []string{"env"}, want: true},
		{environment: "env", constraints: []string{"env", "!prod"}, want: true},
		{environment: "env", constraints: []string{"env", "!env"}, want: false},
		{environment: "env", constraints: []string{"!prod"}, want: true},
		{environment: "env", constraints: []string{"!env"}, want: false},
		{environment: "prod", constraints: []string{"!test", "stage"}, want: true},
		{environment: "prod", constraints: []string{"!test", "!stage"}, want: true},
		{environment: "prod", constraints: []string{"!test", "!prod"}, want: false},
	}

	for _, te := range tt {
		if got := envtag.Match(te.environment, te.constraints); got != te.want {
			t.Errorf("expected %v, got %v", te.want, got)
		}
	}
}
