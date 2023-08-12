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

// Subtracts from `o` the elements in `s`
func Subtract[T ~[]E, E comparable](o T, s T) T {
	m := map[E]struct{}{}
	for _, e := range o {
		m[e] = struct{}{}
	}
	for _, e := range s {
		delete(m, e)
	}

	rs := make(T, len(m))
	c := 0
	for k := range m {
		rs[c] = k
		c++
	}
	return rs
}
