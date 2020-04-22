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
package aves

import (
	"os"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/maarek/aves/store"
)

const fill = 1000000

func BenchmarkBadger(b *testing.B) {
	b.ReportAllocs()
	// Setup
	go func() {
		if err := NewRespServer(":6379", "badger", "../tmp/badger.aves", false).Start(); err != nil {
			b.Errorf("server should start up without error %v", err.Error())
		}
	}()

	time.Sleep(time.Second / 4)

	conn, err := redis.Dial("tcp", ":6379", redis.DialConnectTimeout(time.Minute))
	if err != nil {
		b.Fatalf("%v", err.Error())
	}

	for i := 0; i < fill; i++ {
		stream := store.GenUlid().String()
		if _, err := conn.Do("PUBLISH", stream, i, "somepayload"); err != nil {
			b.Fatalf("%v", err.Error())
		}
	}

	// Benchmarks
	b.Run("BenchmarkBadgerSet", func(b *testing.B) {
		stream := store.GenUlid().String()
		for n := 0; n < b.N; n++ {
			if _, err := conn.Do("PUBLISH", stream, n, "somepayload"); err != nil {
				b.Fatalf("%v", err.Error())
			}
		}
	})
	b.Run("BenchmarkBadgerGet", func(b *testing.B) {
		stream := store.GenUlid().String()
		for n := 0; n < b.N; n++ {
			if _, err := conn.Do("ELIST", stream); err != nil {
				b.Fatalf("%v", err.Error())
			}
		}
	})

	// Cleanup
	if _, err := conn.Do("QUIT"); err != nil {
		b.Fatalf("%v", err.Error())
	}

	cleanupDir("tmp/badger.aves")
	conn.Close()
}

func BenchmarkPebble(b *testing.B) {
	b.ReportAllocs()
	// Setup
	go func() {
		if err := NewRespServer(":6379", "pebble", "../tmp/pebble.aves", false).Start(); err != nil {
			b.Errorf("server should start up without error %v", err.Error())
		}
	}()

	time.Sleep(time.Second / 4)

	conn, err := redis.Dial("tcp", ":6379", redis.DialConnectTimeout(time.Minute))
	if err != nil {
		b.Fatalf("%v", err.Error())
	}

	for i := 0; i < fill; i++ {
		stream := store.GenUlid().String()
		if _, err := conn.Do("PUBLISH", stream, i, "somepayload"); err != nil {
			b.Fatalf("%v", err.Error())
		}
	}

	// Benchmarks
	b.Run("BenchmarkPebbleSet", func(b *testing.B) {
		stream := store.GenUlid().String()
		for n := 0; n < b.N; n++ {
			if _, err := conn.Do("PUBLISH", stream, n, "somepayload"); err != nil {
				b.Errorf("%v", err.Error())
			}
		}
	})
	b.Run("BenchmarkPebbleGet", func(b *testing.B) {
		stream := store.GenUlid().String()
		for n := 0; n < b.N; n++ {
			if _, err := conn.Do("ELIST", stream); err != nil {
				b.Errorf("%v", err.Error())
			}
		}
	})

	// Cleanup
	if _, err := conn.Do("QUIT"); err != nil {
		b.Fatalf("%v", err.Error())
	}

	conn.Close()
}

func cleanupDir(dir string) { // The target directory.
	_ = os.RemoveAll(dir)
}
