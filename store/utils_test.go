/*
 * Copyright 2020 Jeremy Lyman
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package store

import (
	"io/ioutil"
	"sync"
	"testing"
)

func BenchmarkUlid(b *testing.B) {
	b.ReportAllocs()
	// Benchmarks
	b.Run("BenchmarkStringPool", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			testGen(b, func() string { return GenUlid().String() })
		}
	})
}

func testGen(b *testing.B, f func() string) {
	var (
		wg      sync.WaitGroup
		workers = 100
		count   = 1000
	)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < count; j++ {
				b.Log(ioutil.Discard, f())
			}
		}()
	}
	wg.Wait()
}
