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
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/alash3al/go-color"
	"github.com/maarek/aves"
	cmds "github.com/maarek/aves/commands"
	"github.com/maarek/aves/oplog"
	"github.com/maarek/aves/store"
	"github.com/maarek/aves/store/badger"
	"github.com/maarek/aves/store/bolt"
	"github.com/maarek/aves/store/pebble"
	"github.com/tidwall/redcon"
)

// Server - metadata to load a server from
type Server struct {
	addr    string
	dbType  store.DBType
	path    string
	verbose bool
}

// NewRespServer - creates a server for running the data store
func NewRespServer(addr, dbt, out string, verbose bool) *Server {
	var dbType store.DBType
	switch dbt {
	case "bolt":
		dbType = store.BOLT
	case "pebble":
		dbType = store.PEBBLE
	default:
		dbType = store.BADGER
	}

	return &Server{
		addr:    addr,
		dbType:  dbType,
		path:    out,
		verbose: verbose,
	}
}

// Start the RESP Server
func (s *Server) Start() error {
	// initialize the data store
	db, err := loadDB(s.dbType, s.path)
	if err != nil {
		return fmt.Errorf("db error: %s", err.Error())
	}

	opl := oplog.NewBroadcaster()

	return redcon.ListenAndServe(
		s.addr,
		func(conn redcon.Conn, cmd redcon.Command) {
			// handles any panic
			defer (func() {
				if err := recover(); err != nil {
					conn.WriteError(fmt.Sprintf("fatal error: %s", (err.(error)).Error()))
				}
			})()

			// normalize the action "command"
			// normalize the command arguments
			action := strings.TrimSpace(strings.ToLower(string(cmd.Args[0])))
			argSlice := cmd.Args[1:]
			args := make([][]byte, len(argSlice))
			for i, v := range argSlice {
				v := bytes.TrimSpace(v)
				args[i] = v
			}

			if s.verbose {
				log.Println(color.YellowString(action), color.CyanString(string(bytes.Join(args, []byte(" ")))))
			}

			// internal ping-pong
			if action == "ping" {
				conn.WriteString("PONG")
				return
			}

			// close the connection
			if action == "quit" {
				conn.WriteString("OK")
				conn.Close()
				return
			}

			// match the command
			fn := aves.Commands[aves.Command(action)]
			if fn == nil {
				conn.WriteError(fmt.Sprintf("unknown commands [%s]", action))
				return
			}

			// dispatch the command and catch its errors
			fn(&cmds.Context{
				Conn:   conn,
				Action: action,
				Args:   args,
				DB:     db,
				OpLog:  opl,
			})
		},
		func(conn redcon.Conn) bool {
			conn.SetContext(map[string]interface{}{})
			return true
		},
		nil,
	)
}

// load/fetches the requested db
func loadDB(dbType store.DBType, out string) (db store.DB, err error) {
	switch dbType {
	case store.BADGER:
		db, err = badger.OpenDB(out)
	case store.BOLT:
		db, err = bolt.OpenDB(out)
	case store.PEBBLE:
		db, err = pebble.OpenDB(out)
	}
	if err != nil {
		return nil, err
	}

	return db, nil
}
