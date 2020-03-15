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
package stream

import (
	cmds "github.com/maarek/aves/commands"
	"github.com/maarek/aves/store"
	"github.com/oklog/ulid/v2"
)

// DeleteCommand - DELETE <stream>
func DeleteCommand(c cmds.Context) {
	if len(c.Args) < 1 {
		c.WriteError("DELETE command must have at least 1 argument: DEL <stream> [<key2> ...]")
		return
	}

	keys := make([]string, len(c.Args))
	for i, arg := range c.Args {
		keys[i] = string(arg)
	}

	if err := c.DB.Del(keys); err != nil {
		c.WriteError(err.Error())
		return
	}

	c.WriteString("OK")
}

// ExistsCommand - EXISTS <stream>
func ExistsCommand(c cmds.Context) {
	if len(c.Args) < 1 {
		c.WriteError("EXISTS command must have at least 1 argument: EXISTS <stream>")
		return
	}

	key := store.Key{
		ID:      ulid.ULID{},
		Stream:  store.StreamID(c.Args[0]),
		Version: []byte{},
	}

	// TODO: This needs a Scan with Badger but a Get for Bucket is fine with Bolt
	_, err := c.DB.Get(key)
	if err != nil {
		c.WriteInt(0)
		return
	}

	c.WriteInt(1)
}

// ListCommand - SLIST
func ListCommand(c cmds.Context) {
	// TODO: Logic does not hold for all data stores
	data := make(map[string]int)

	err := c.DB.Scan(store.ScannerOptions{
		FetchValues:   false,
		IncludeOffset: true,
		Handler: func(k store.Key, _ string) bool {
			data[string(k.Stream)]++
			return true
		},
	})

	if err != nil {
		c.WriteError(err.Error())
		return
	}

	FIELDS := 2
	c.WriteArray(len(data) * FIELDS)
	for k, size := range data {
		c.WriteBulkString(k)
		c.WriteInt(size)
	}
}
