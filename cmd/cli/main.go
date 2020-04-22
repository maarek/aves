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
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/maarek/aves"
	"github.com/maarek/aves/client"
)

func streamDelete(c *client.Context, args []string) error {
	var stream string
	if len(args) > 2 {
		stream = args[2]
	}
	if ok, err := c.Delete(stream); !ok || err != nil {
		return err
	}
	fmt.Println("success")
	return nil
}

func streamExists(c *client.Context, args []string) error {
	var stream string
	if len(args) > 2 {
		stream = args[2]
	}
	exists, err := c.Exists(stream)
	if err != nil {
		return err
	}
	if exists {
		fmt.Println("true")
	} else {
		fmt.Println("false")
	}
	return nil
}

func streamList(c *client.Context) error {
	streams, err := c.SList()
	if err != nil {
		return err
	}
	for _, stream := range streams {
		fmt.Printf("%s: %d\n", stream.StreamID, stream.EventCount)
	}
	return nil
}

func eventList(c *client.Context, args []string) error {
	var offset, limit string
	if len(args) > 3 {
		offset = args[3]
	}
	if len(args) > 4 {
		limit = args[4]
	}
	events, err := c.EList(args[2], offset, limit)
	if err != nil {
		return err
	}
	for _, event := range events {
		fmt.Printf("%d: %s\n", event.Version, event.Data)
	}
	return nil
}

func eventPublish(c *client.Context, args []string) error {
	var stream, version, data string
	if len(args) > 2 {
		stream = args[2]
	}
	if len(args) > 3 {
		version = args[3]
	}
	if len(args) > 4 {
		version = args[4]
	}
	if ok, err := c.Publish(stream, version, data); !ok || err != nil {
		return err
	}
	fmt.Println("success")
	return nil
}

func streamSubscribe(c *client.Context, args []string) error {
	var stream, offset string
	if len(args) > 2 {
		stream = args[2]
	}
	if len(args) > 3 {
		offset = args[3]
	}
	inc := make(chan client.FullEvent, 10)
	errc := make(chan error)
	go c.Subscribe(inc, errc, stream, offset)
	for {
		select {
		case err := <-errc:
			return err
		case event := <-inc:
			fmt.Printf("%s:%s:%d: %s\n", event.StreamID, event.EventID, event.Version, event.Data)
		}
	}
}

func subscribeAll(c *client.Context, args []string) error {
	var offset string
	if len(args) > 2 {
		offset = args[2]
	}
	inc := make(chan client.FullEvent, 10)
	errc := make(chan error)
	go c.SubscribeAll(inc, errc, offset)
	for {
		select {
		case err := <-errc:
			return err
		case event := <-inc:
			fmt.Printf("%s:%s:%d: %s\n", event.StreamID, event.EventID, event.Version, event.Data)
		}
	}
}

func main() {
	addr := flag.String("addr", ":6379", "host:port for resp api server")
	flag.Parse()

	// Verify that the Command has beeen provided.
	// os.Arg[0] the command
	if len(os.Args) < 2 {
		fmt.Println("command is required")
		os.Exit(1)
	}

	c, err := client.NewClient(*addr)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// match commands
	switch aves.Command(strings.ToLower(os.Args[1])) {
	// stream
	case aves.StreamDelete:
		err = streamDelete(c, os.Args)
	case aves.StreamExists:
		err = streamExists(c, os.Args)
	case aves.StreamList:
		err = streamList(c)
	// events
	case aves.EventList:
		err = eventList(c, os.Args)
	// pubsub
	case aves.EventPublish:
		err = eventPublish(c, os.Args)
	case aves.StreamSubscribe:
		err = streamSubscribe(c, os.Args)
	case aves.SubscribeAll:
		err = subscribeAll(c, os.Args)
	default:
		err = errors.New("unknown command")
	}

	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}

	os.Exit(0)
}
