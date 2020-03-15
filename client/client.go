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
package client

import (
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/maarek/aves"
)

type Context struct {
	client redis.Conn
}

type CommandClient interface {
	// stream
	Delete(stream string) (bool, error)
	Exists(stream string) (bool, error)
	SList() ([]Stream, error)

	// events
	EList(stream string, offset string, index string) ([]SimpleEvent, error)

	// pubsub
	Publish(stream string, version string, event string) (bool, error)
	Subscribe(inc chan<- FullEvent, errc chan<- error, stream string, offset string)
	SubscribeAll(inc chan<- FullEvent, errc chan<- error, offset string)
}

func NewClient(addr string) (*Context, error) {
	conn, err := redis.Dial("tcp", addr, redis.DialConnectTimeout(time.Minute))
	if err != nil {
		return nil, fmt.Errorf("could not connect to the resp server: %v\n", err)
	}

	return &Context{
		client: conn,
	}, nil
}

func (c *Context) Delete(stream string) (bool, error) {
	v, err := redis.String(c.client.Do(string(aves.STREAM_DELETE), stream))
	if v == "OK" {
		return true, nil
	}
	return false, err
}

func (c *Context) Exists(stream string) (bool, error) {
	exists, err := redis.Bool(c.client.Do(string(aves.STREAM_EXISTS), stream))
	return exists, err
}

func (c *Context) SList() ([]Stream, error) {
	resp, err := redis.Values(c.client.Do(string(aves.STREAM_LIST)))
	if err != nil {
		return nil, err
	}
	return parseSListResp(resp)
}

func (c *Context) EList(stream string, offset string, index string) ([]SimpleEvent, error) {
	resp, err := redis.Values(c.client.Do(string(aves.EVENT_LIST), stream, offset, index))
	if err != nil {
		return nil, err
	}
	return parseSimpleEventListResp(resp)
}

func (c *Context) Publish(stream string, version string, event string) (bool, error) {
	v, err := redis.String(c.client.Do(string(aves.EVENT_PUBLISH), stream, version, event))
	if v == "OK" {
		return true, nil
	}
	return false, err
}

func (c *Context) Subscribe(inc chan<- FullEvent, errc chan<- error, stream string, offset string) {
	err := c.client.Send(string(aves.STREAM_SUBSCRIBE), stream, offset)
	if err != nil {
		errc <- err
		return
	}

	err = c.client.Flush()
	if err != nil {
		errc <- err
		return
	}

	for {
		resp, err := redis.Values(c.client.Receive())
		if err != nil {
			errc <- err
			return
		}
		// process pushed message
		parsed, err := parseFullEventListResp(resp)
		if err != nil {
			errc <- err
			return
		}

		for _, event := range parsed {
			inc <- event
		}
	}
}

func (c *Context) SubscribeAll(inc chan<- FullEvent, errc chan<- error, offset string) {
	err := c.client.Send(string(aves.SUBSCRIBE_ALL), offset)
	if err != nil {
		errc <- err
		return
	}

	err = c.client.Flush()
	if err != nil {
		errc <- err
		return
	}

	for {
		resp, err := redis.Values(c.client.Receive())
		if err != nil {
			errc <- err
			return
		}
		// process pushed message
		parsed, err := parseFullEventListResp(resp)
		if err != nil {
			errc <- err
			return
		}

		for _, event := range parsed {
			inc <- event
		}
	}
}
