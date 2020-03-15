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
	"github.com/oklog/ulid/v2"
)

type DBType int

const (
	BADGER DBType = iota
	BOLT
	PEBBLE
)

// Stream ID
type StreamID []byte

// Key structure to hold both stream and version
type Key struct {
	// TimeSeries Identifier
	ID ulid.ULID // 16 byte array

	// Stream name
	Stream StreamID

	// Version offset
	Version []byte
}

// Create a new Event with defined Ulid
func NewEventKey(stream []byte, version []byte) Key {
	return Key{
		ID:      GenUlid(),
		Stream:  StreamID(stream),
		Version: version,
	}
}

// DB Interface
type DB interface {
	Set(k Key, v string) error
	Get(k Key) (string, error)
	Del(keys []string) error
	Scan(ScannerOpt ScannerOptions) error
	Size() int64
	GC() error
	Close()
}

type Handler func(k Key, v string) bool

// ScannerOptions - represents the options for a scanner
type ScannerOptions struct {
	// from where to start
	Offset []byte

	// whether to include the value of the offset in the result or not
	IncludeOffset bool

	// the prefix that must be exists in each key in the iteration
	Prefix []byte

	// fetch the values (true) or this is a key only iteration (false)
	FetchValues bool

	// fetch values from the time series index
	Index bool

	// the handler that handles the incoming data
	Handler Handler
}
