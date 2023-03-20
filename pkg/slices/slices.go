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

package slices

func SubtractStr(s1, s2 []string) []string {
	m := map[string]struct{}{}
	for _, s := range s1 {
		m[s] = struct{}{}
	}
	for _, s := range s2 {
		delete(m, s)
	}

	rs := make([]string, len(m))
	c := 0
	for k := range m {
		rs[c] = k
		c++
	}
	return rs
}

// ItemContains return true if the slice contains
// the given string
func ItemContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
