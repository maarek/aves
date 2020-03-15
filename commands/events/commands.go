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
package events

import (
	"strconv"

	cmds "github.com/maarek/aves/commands"
	"github.com/maarek/aves/store"
)

// RangeCommand - ELIST <stream> [<offset> <size>]
func RangeCommand(c cmds.Context) {
	var offset []byte
	var limit int

	if len(c.Args) < 1 {
		c.WriteError("ELIST must has at least 1 argument, ELIST <STREAM> [<offset> <size>]")
		return
	}

	prefix := c.Args[0]

	if len(c.Args) > 1 {
		offset = c.Args[1]
	}
	if len(c.Args) > 2 {
		limit, _ = strconv.Atoi(string(c.Args[2])) // TODO: Optimize?
	}

	includeOffsetVals := false
	if len(offset) == 0 {
		includeOffsetVals = true
	}

	data := []string{}
	loaded := 0
	err := c.DB.Scan(store.ScannerOptions{
		IncludeOffset: includeOffsetVals,
		Offset:        offset,
		Prefix:        prefix,
		FetchValues:   true,
		Handler: func(k store.Key, v string) bool {
			if limit > 0 && (loaded >= limit) {
				return false
			}
			data = append(data, string(k.Version), v)
			loaded++
			return true
		},
	})

	if err != nil {
		c.WriteError(err.Error())
		return
	}

	if len(data) == 0 {
		c.WriteNull()
		return
	}

	c.WriteArray(len(data))
	for k, v := range data {
		if k%2 == 0 {
			ver, err := strconv.Atoi(v)
			if err != nil {
				c.WriteNull()
			}
			c.WriteInt(ver)
		} else {
			c.WriteBulkString(v)
		}
	}
}
