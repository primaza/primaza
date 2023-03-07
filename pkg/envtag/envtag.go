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

package envtag

import "strings"

const NegativeConstraintSymbol = "!"

type matchResult byte

const (
	matched   matchResult = 0
	unmatched matchResult = 1
	forbidden matchResult = 2
)

// Match matches an environment against a list of constraints
func Match(environment string, constraints []string) bool {
	if len(constraints) == 0 {
		return true
	}

	v := false
	for _, m := range constraints {
		switch match(environment, m) {
		case matched:
			v = true
		case forbidden:
			return false
		}
	}
	return v
}

func match(environment, constraint string) matchResult {
	if strings.HasPrefix(constraint, NegativeConstraintSymbol) {
		pc := strings.TrimPrefix(constraint, NegativeConstraintSymbol)
		if environment == pc {
			return forbidden
		}
	}

	if environment == constraint {
		return matched
	}
	return unmatched
}
