/*
 * Copyright 2019 Jeremy Lyman
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

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/alash3al/go-color"
	su "github.com/maarek/aves/server"
	_ "go.uber.org/automaxprocs/maxprocs"
)

func Ballast(size int) func() {
	ballast := make([]byte, size)
	return func() { runtime.KeepAlive(ballast) }
}

func main() {
	port := flag.Int("port", 6379, "port for resp api server")

	dbType := flag.String("type", "badger", "type of datastore (badger,bolt,pebble)")
	out := flag.String("out", "", "location of the database files")

	verbose := flag.Bool("verbose", false, "log level verbose")

	ballast := flag.Int("ballast", 2560, "ballast in MBs")

	flag.Parse()

	if *out == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Create a large heap allocation of nGiB
	// https://blog.twitch.tv/go-memory-ballast-how-i-learnt-to-stop-worrying-and-love-the-heap-26c2462549a2
	defer Ballast(*ballast << 20)()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Initialized Ballast %d\n", m.Alloc)

	fmt.Printf("starting server on port %d\n", *port)

	err := make(chan error)

	go (func() {
		err <- su.NewRespServer(fmt.Sprintf(":%d", *port), *dbType, *out, *verbose).Start()
	})()

	go func() {
		fmt.Println(http.ListenAndServe("localhost:6061", nil))
	}()

	if err := <-err; err != nil {
		color.Red(err.Error())
	}
}
