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

package pubsub

import (
	"strconv"

	cmds "github.com/maarek/aves/commands"
	"github.com/maarek/aves/oplog"
	"github.com/maarek/aves/store"
	"github.com/tidwall/redcon"
)

// const defaultPubSubAllTopic = "*"

// PublishCommand - PUBLISH <stream> <version> <event-payload>
func PublishCommand(c *cmds.Context) {
	if len(c.Args) < 3 {
		c.WriteError("PUBLISH command must have at all required argument: PUBLISH <stream> <version> <event-payload>")
		return
	}

	// TODO: Check this another way?
	_, err := strconv.Atoi(string(c.Args[1]))
	if err != nil {
		c.WriteError("PUBLISH command must have an integer version string")
		return
	}

	key := store.NewEventKey(c.Args[0], c.Args[1])
	value := string(c.Args[2])

	err = c.DB.Set(key, value)
	if err != nil {
		c.WriteError("PUBLISH could not write event to the data store")
		return
	}

	// Publish to OpLog
	c.OpLog.Write(KeyValue{
		Key:   key,
		Value: value,
	})

	c.WriteString("OK")
}

// SubscribeCommand - SUBSCRIBE <stream> [<offset>]
func SubscribeCommand(c *cmds.Context) {
	if len(c.Args) < 1 {
		c.WriteError("SUBSCRIBE must has at least 1 argument, SUBSCRIBE <stream> [<offset>]")
		return
	}

	conn := c.Detach()

	var offset []byte
	prefix := c.Args[0]

	if len(c.Args) > 1 {
		offset = c.Args[1]
	}

	includeOffsetVals := false
	if len(offset) == 0 {
		includeOffsetVals = true
	}

	data := []KeyValue{}
	loaded := 0
	err := c.DB.Scan(store.ScannerOptions{
		IncludeOffset: includeOffsetVals,
		Offset:        offset,
		Prefix:        prefix,
		FetchValues:   true,
		Handler: func(k store.Key, v string) bool {
			data = append(data, KeyValue{
				Key:   k,
				Value: v,
			})
			loaded++
			return true
		},
	})

	if err != nil {
		c.WriteError(err.Error())
		return
	}

	for i, kv := range data {
		var d []byte
		if i%2 == 0 {
			continue
		}
		d = redcon.AppendArray(d, 4)
		d = redcon.AppendBulkString(d, string(kv.Key.Stream))
		d = redcon.AppendBulkString(d, kv.Key.ID.String())
		d = redcon.AppendBulkString(d, string(kv.Key.Version))
		d = redcon.AppendBulkString(d, kv.Value)
		if _, err := conn.NetConn().Write(d); err != nil {
			return
		}
	}

	// Stream from OpLog
	listener := c.OpLog.Listen()
	go listen(listener, conn)
}

// SubscribeAllCommand - SUBSCRIBEALL [<index>]
func SubscribeAllCommand(c *cmds.Context) {
	var prefix []byte
	conn := c.Detach()

	includeOffsetVals := true
	if len(c.Args) > 0 {
		prefix = c.Args[0]
		includeOffsetVals = false
	}

	data := []KeyValue{}
	loaded := 0
	err := c.DB.Scan(store.ScannerOptions{
		IncludeOffset: includeOffsetVals,
		Prefix:        prefix,
		Index:         true,
		FetchValues:   true,
		Handler: func(k store.Key, v string) bool {
			data = append(data, KeyValue{
				Key:   k,
				Value: v,
			})
			loaded++
			return true
		},
	})

	if err != nil {
		c.WriteError(err.Error())
		return
	}

	for i, kv := range data {
		var d []byte
		if i%2 == 0 {
			continue
		}
		d = redcon.AppendArray(d, 4)
		d = redcon.AppendBulkString(d, string(kv.Key.Stream))
		d = redcon.AppendBulkString(d, kv.Key.ID.String())
		d = redcon.AppendBulkString(d, string(kv.Key.Version))
		d = redcon.AppendBulkString(d, kv.Value)
		if _, err := conn.NetConn().Write(d); err != nil {
			return
		}
	}

	// Stream from OpLog
	listener := c.OpLog.Listen()
	go listen(listener, conn)
}

// KeyValue - key and value
type KeyValue struct {
	Key   store.Key
	Value string
}

func listen(r oplog.Receiver, conn redcon.DetachedConn) {
	for m := r.Read(); m != nil; m = r.Read() {
		kv := m.(KeyValue)
		var d []byte
		d = redcon.AppendArray(d, 4)
		d = redcon.AppendBulkString(d, string(kv.Key.Stream))
		d = redcon.AppendBulkString(d, kv.Key.ID.String())
		d = redcon.AppendBulkString(d, string(kv.Key.Version))
		d = redcon.AppendBulkString(d, kv.Value)
		if _, err := conn.NetConn().Write(d); err != nil {
			_ = conn.Close()
			return
		}
	}
}
