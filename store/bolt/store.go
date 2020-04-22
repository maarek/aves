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

package bolt

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	"github.com/boltdb/bolt"
	"github.com/maarek/aves/store"
	"github.com/oklog/ulid/v2"
)

var defaultConst = []byte("default")

// DB - represents a badger db implementation
type DB struct {
	bolt *bolt.DB
}

// OpenDB - Opens the specified path
func OpenDB(path string) (*DB, error) {
	bdb, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	if err := bdb.Update(func(txn *bolt.Tx) error {
		_, err := txn.CreateBucketIfNotExists(defaultConst)
		return err
	}); err != nil {
		return nil, err
	}

	db := new(DB)
	db.bolt = bdb

	return db, nil
}

// Close - closes the database
func (db *DB) Close() {
	db.bolt.Close()
}

// Size - returns the size of the database in bytes
// returns -1 on error
func (db *DB) Size() int64 {
	var size int64

	err := db.bolt.View(func(txn *bolt.Tx) error {
		size = txn.Size()
		return nil
	})
	if err != nil {
		return -1
	}

	return size
}

// GC - runs the garbage collector
func (db *DB) GC() error {
	return nil
}

// Set - sets a key with the specified value if the version doesn't exist
func (db *DB) Set(k store.Key, v string) error {
	return db.bolt.Update(func(txn *bolt.Tx) (err error) {
		d := txn.Bucket(defaultConst)

		b, err := d.CreateBucketIfNotExists(k.Stream[:])
		if err != nil {
			return err
		}

		item := b.Get(k.Version)
		if item != nil {
			return fmt.Errorf("event for key exists %v", k)
		}

		err = b.Put(k.Version, []byte(v))

		return err
	})
}

// Get - fetches the value of the specified key
func (db *DB) Get(k store.Key) (string, error) {
	var data string

	err := db.bolt.View(func(txn *bolt.Tx) error {
		d := txn.Bucket(defaultConst)

		b := d.Bucket(k.Stream[:])

		item := b.Get(k.Version)

		val := make([]byte, len(item))
		copy(val, item)

		data = string(val)

		return nil
	})

	return data, err
}

// Del - removes key(s) from the store
func (db *DB) Del(keys []string) error {
	return db.bolt.Update(func(txn *bolt.Tx) error {
		for _, key := range keys {
			d := txn.Bucket(defaultConst)
			err := d.DeleteBucket([]byte(key))
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// Scan - iterate over the whole store using the handler function
func (db *DB) Scan(scannerOpt store.ScannerOptions) error {
	return db.bolt.View(func(txn *bolt.Tx) error {
		var k, v []byte
		var it *bolt.Cursor
		var offset int
		var err error

		if len(scannerOpt.Offset) != 0 {
			offset, err = strconv.Atoi(string(scannerOpt.Offset))
			if err != nil {
				return errors.New("cannot convert offset to int")
			}
		}

		d := txn.Bucket(defaultConst)

		if len(scannerOpt.Prefix) != 0 {
			bucket := d.Bucket(scannerOpt.Prefix)
			if bucket == nil {
				return nil
			}
			it = bucket.Cursor()
		} else {
			it = d.Cursor()
		}

		start := func(it *bolt.Cursor) {
			if len(scannerOpt.Offset) == 0 {
				k, v = it.First()
			} else {
				bver := make([]byte, 4)
				binary.LittleEndian.PutUint32(bver, uint32(offset))
				k, v = it.Seek(bver)
				if !scannerOpt.IncludeOffset && k != nil {
					k, v = it.Next()
				}
			}
		}

		valid := func(_ *bolt.Cursor) bool {
			return k != nil
		}

		for start(it); valid(it); k, v = it.Next() {
			var streamID store.StreamID
			var ver []byte

			kCopy := make([]byte, len(k))
			copy(kCopy, k)

			if v != nil {
				copy(streamID[:], scannerOpt.Prefix)
				ver = kCopy
			} else {
				copy(streamID[:], kCopy)
				ver = []byte{}
			}

			key := store.Key{
				ID:      ulid.ULID{},
				Stream:  streamID,
				Version: ver,
			}

			vCopy := make([]byte, len(v))
			copy(vCopy, v)

			if !scannerOpt.Handler(key, string(vCopy)) {
				break
			}
		}

		return nil
	})
}
