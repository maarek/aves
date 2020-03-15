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
	"os"
	"strings"

	"github.com/maarek/aves"
	c "github.com/maarek/aves/client"
)

func main() {
	addr := flag.String("addr", ":6379", "host:port for resp api server")
	flag.Parse()

	// Verify that the Command has beeen provided.
	// os.Arg[0] the command
	if len(os.Args) < 2 {
		fmt.Println("command is required")
		os.Exit(1)
	}

	client, err := c.NewClient(*addr)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// match commands
	switch aves.Command(strings.ToLower(os.Args[1])) {
	// stream
	case aves.STREAM_DELETE:
		var stream string
		if len(os.Args) > 2 {
			stream = os.Args[2]
		}
		if ok, err := client.Delete(stream); !ok || err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		fmt.Println("success")
	case aves.STREAM_EXISTS:
		var stream string
		if len(os.Args) > 2 {
			stream = os.Args[2]
		}
		exists, err := client.Exists(stream)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		if exists {
			fmt.Println("true")
		} else {
			fmt.Println("false")
		}
	case aves.STREAM_LIST:
		streams, err := client.SList()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		for _, stream := range streams {
			fmt.Printf("%s: %d\n", stream.StreamID, stream.EventCount)
		}
	// events
	case aves.EVENT_LIST:
		var offset, limit string
		if len(os.Args) > 3 {
			offset = os.Args[3]
		}
		if len(os.Args) > 4 {
			limit = os.Args[4]
		}
		events, err := client.EList(os.Args[2], offset, limit)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		for _, event := range events {
			fmt.Printf("%d: %s\n", event.Version, event.Data)
		}
	// pubsub
	case aves.EVENT_PUBLISH:
		var stream, version, data string
		if len(os.Args) > 2 {
			stream = os.Args[2]
		}
		if len(os.Args) > 3 {
			version = os.Args[3]
		}
		if len(os.Args) > 4 {
			version = os.Args[4]
		}
		if ok, err := client.Publish(stream, version, data); !ok || err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		fmt.Println("success")
	case aves.STREAM_SUBSCRIBE:
		var stream, offset string
		if len(os.Args) > 2 {
			stream = os.Args[2]
		}
		if len(os.Args) > 3 {
			offset = os.Args[3]
		}
		inc := make(chan c.FullEvent, 10)
		errc := make(chan error)
		go client.Subscribe(inc, errc, stream, offset)
		for {
			select {
			case err := <-errc:
				fmt.Println(err.Error())
				os.Exit(1)
			case event := <-inc:
				fmt.Printf("%s:%s:%d: %s\n", event.StreamID, event.EventID, event.Version, event.Data)
			}
		}
	case aves.SUBSCRIBE_ALL:
		var offset string
		if len(os.Args) > 2 {
			offset = os.Args[2]
		}
		inc := make(chan c.FullEvent, 10)
		errc := make(chan error)
		go client.SubscribeAll(inc, errc, offset)
		for {
			select {
			case err := <-errc:
				fmt.Println(err.Error())
				os.Exit(1)
			case event := <-inc:
				fmt.Printf("%s:%s:%d: %s\n", event.StreamID, event.EventID, event.Version, event.Data)
			}
		}
	default:
		fmt.Println("unknown command")
		os.Exit(1)
	}

	os.Exit(0)
}
