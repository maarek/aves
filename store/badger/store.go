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
package badger

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
	"github.com/maarek/aves/store"
	"github.com/oklog/ulid/v2"
)

// BadgerDB - represents a badger db implementation
type BadgerDB struct {
	badger *badger.DB
}

// OpenDB - Opens the specified path
func OpenDB(path string) (*BadgerDB, error) {
	opts := badger.DefaultOptions(path)

	opts.Truncate = true
	opts.SyncWrites = false
	opts.TableLoadingMode = options.MemoryMap
	opts.ValueLogLoadingMode = options.FileIO
	opts.NumMemtables = 2
	opts.MaxTableSize = 10 << 20
	opts.NumLevelZeroTables = 2
	opts.ValueThreshold = 1

	bdb, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	db := new(BadgerDB)
	db.badger = bdb

	go (func() {
		for db.badger.RunValueLogGC(0.5) == nil {
			// cleaning ...
		}
	})()

	return db, nil
}

// Close
func (db *BadgerDB) Close() {
	db.badger.Close()
}

// Size - returns the size of the database (LSM + ValueLog) in bytes
func (db *BadgerDB) Size() int64 {
	lsm, vlog := db.badger.Size()
	return lsm + vlog
}

// GC - runs the garbage collector
func (db *BadgerDB) GC() error {
	var err error
	for {
		err = db.badger.RunValueLogGC(0.5)
		if err != nil {
			break
		}
	}
	return err
}

// Set - sets a key with the specified value if the version doesn't exist
func (db *BadgerDB) Set(k store.Key, v string) error {
	return db.badger.Update(func(txn *badger.Txn) (err error) {
		key, err := packStream(k)
		if err != nil {
			return err
		}

		item, err := txn.Get(key)
		if err != nil && err.Error() != "Key not found" {
			return err
		}
		if item != nil {
			return fmt.Errorf("event for key exists %v", k)
		}

		err = txn.Set(key, []byte(v))
		if err != nil {
			return err
		}

		id := [16]byte(store.GenUlid())
		key = packIndex(id[:], k.Stream[:], k.Version)
		err = txn.Set(key, []byte(v))

		return err
	})
}

// Get - fetches the value of the specified key
func (db *BadgerDB) Get(k store.Key) (string, error) {
	var data string

	err := db.badger.View(func(txn *badger.Txn) error {
		key, err := packStream(k)
		if err != nil {
			return err
		}
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		data = string(val)

		return nil
	})

	return data, err
}

// Del - removes key(s) from the store
func (db *BadgerDB) Del(keys []string) error {
	return db.badger.Update(func(txn1 *badger.Txn) error {
		for _, key := range keys {
			// scan for keys with prefix
			err := db.badger.View(func(txn2 *badger.Txn) error {
				it := txn2.NewIterator(badger.DefaultIteratorOptions)
				defer it.Close()

				// TODO: Move to Pack
				var sb strings.Builder
				sb.Grow(2 + len(key))
				sb.WriteString("s:")
				sb.WriteString(key)
				prefix := []byte(sb.String())

				for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
					item := it.Item()
					k := item.Key()
					// delete each key
					err := txn1.Delete(k)
					if err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// Scan - iterate over the whole store using the handler function
func (db *BadgerDB) Scan(scannerOpt store.ScannerOptions) error {
	var prefix []byte
	// Index scan for time
	if scannerOpt.Index {
		prefix = indexScanPrefix(scannerOpt.Prefix)
	} else {
		prefix = streamScanPrefix(scannerOpt.Prefix)
	}

	return db.badger.View(func(txn *badger.Txn) error {
		iteratorOpts := badger.DefaultIteratorOptions
		iteratorOpts.PrefetchValues = scannerOpt.FetchValues

		it := txn.NewIterator(iteratorOpts)
		defer it.Close()

		start := func(it *badger.Iterator) {
			if len(prefix) == 0 {
				it.Rewind()
			} else {
				it.Seek(prefix)
			}
		}

		valid := func(it *badger.Iterator) bool {
			if !it.Valid() {
				return false
			}

			if len(prefix) != 0 && !it.ValidForPrefix(prefix) {
				return false
			}

			return true
		}

		seen := false

		for start(it); valid(it); it.Next() {
			item := it.Item()

			hs := bytes.HasSuffix(item.Key(), scannerOpt.Offset)

			// ignore up to offset
			if !seen && !hs {
				continue
			}

			seen = true

			if hs && !scannerOpt.IncludeOffset {
				continue
			}

			var k, v []byte

			k = item.KeyCopy(nil)

			if scannerOpt.FetchValues {
				v, _ = item.ValueCopy(nil)
			}

			var key store.Key
			var err error

			if scannerOpt.Index {
				key, err = unpackIndex(k)
			} else {
				key, err = unpackStream(k)
			}
			if err != nil {
				return fmt.Errorf("invalid key format %s", string(k))
			}

			if !scannerOpt.Handler(key, string(v)) {
				break
			}
		}

		return nil
	})
}

func streamScanIdentifier() []byte {
	return []byte{'s', ':'}
}

func streamScanPrefix(prefix []byte) []byte {
	if len(prefix) == 0 {
		return streamScanIdentifier()
	}

	buf := make([]byte, len(prefix)+2)
	copy(buf, streamScanIdentifier()[:])
	copy(buf[2:], prefix)

	return buf[:]
}

func indexScanPrefix(ts []byte) []byte {
	var buf []byte
	buf = append(buf, 't', ':')
	if len(ts) == 0 {
		return buf
	}

	// Nab the first 6 bytes of the time index
	return append(buf, ts[:6]...)
}

func packStream(k store.Key) ([]byte, error) {
	if len(k.Stream) == 0 {
		return make([]byte, 0), fmt.Errorf("unable to pack key %v", k)
	}
	var buf []byte
	buf = append(buf, 's', ':')
	buf = append(buf, k.Stream[:]...)
	if len(k.Version) > 0 {
		buf = append(buf, ':')
		buf = append(buf, k.Version...)
	}
	return buf, nil
}

func unpackStream(key []byte) (store.Key, error) {
	k := store.Key{}
	v := bytes.Split(key, []byte{':'})

	if len(v) < 3 && (len(v[1]) == 0 || len(v[2]) == 0) {
		return k, fmt.Errorf("unable to unpack key %v", key)
	}

	k.ID = ulid.ULID{}
	k.Stream = store.StreamID(v[1])
	k.Version = v[2]

	return k, nil
}

func packIndex(ts, stream, offset []byte) []byte {
	var buf []byte
	buf = append(buf, 't', ':')
	buf = append(buf, ts...)
	buf = append(buf, ':')
	buf = append(buf, stream...)
	buf = append(buf, ':')
	buf = append(buf, offset...)
	return buf
}

func unpackIndex(key []byte) (store.Key, error) {
	k := store.Key{}
	v := bytes.Split(key, []byte{':'})

	if len(v) == 4 && (len(v[1]) != 16 || len(v[2]) == 0 || len(v[3]) == 0) {
		return k, fmt.Errorf("unable to unpack key %v", key)
	}

	var ulid ulid.ULID
	copy(ulid[:], v[1][:])

	k.ID = ulid
	k.Stream = store.StreamID(v[2])
	k.Version = v[3]

	return k, nil
}
