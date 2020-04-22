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

package pebble

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/cockroachdb/pebble"
	"github.com/maarek/aves/store"
	"github.com/oklog/ulid/v2"
)

// DB - represents a pebble db implementation
type DB struct {
	pebble *pebble.DB
	wo     *pebble.WriteOptions
}

// OpenDB - Opens the specified path
func OpenDB(path string) (*DB, error) {
	c := *pebble.DefaultComparer
	// NB: this is named as such only to match the built-in RocksDB comparer.
	c.Name = "leveldb.BytewiseComparator"
	c.Split = func(a []byte) int {
		return len(a)
	}

	opts := &pebble.Options{
		Comparer: &c,
	}
	wo := pebble.NoSync

	pdb, err := pebble.Open(path, opts)
	if err != nil {
		return nil, err
	}

	db := new(DB)
	db.pebble = pdb
	db.wo = wo

	return db, nil
}

// Close - closes the database
func (db *DB) Close() {
	db.pebble.Close()
}

// Size - returns the size of the database in bytes
func (db *DB) Size() int64 {
	return -1
}

// GC - runs the garbage collector
func (db *DB) GC() error {
	return fmt.Errorf("unimplemented")
}

// Set - sets a key with the specified value if the version doesn't exist
func (db *DB) Set(k store.Key, v string) error {
	key, err := packStream(k)
	if err != nil {
		return err
	}

	item, closer, err := db.pebble.Get(key)
	if err != nil && err.Error() != "pebble: not found" {
		return err
	}
	if len(item) > 0 {
		closer.Close()
		return fmt.Errorf("event for key exists %v", k)
	}

	wb := db.pebble.NewBatch()

	err = wb.Set(key, []byte(v), db.wo)
	if err != nil {
		return err
	}

	id := [16]byte(store.GenUlid())
	key = packIndex(id[:], k.Stream[:], k.Version)
	err = wb.Set(key, []byte(v), db.wo)
	if err != nil {
		return err
	}

	err = wb.Commit(db.wo)

	return err
}

// Get - fetches the value of the specified key
func (db *DB) Get(k store.Key) (string, error) {
	key, err := packStream(k)
	if err != nil {
		return "", err
	}
	item, closer, err := db.pebble.Get(key)
	if err != nil {
		return "", err
	}

	defer closer.Close()

	var sb strings.Builder
	sb.Grow(len(item))
	sb.Write(item)

	return sb.String(), err
}

// Del - removes key(s) from the store
func (db *DB) Del(keys []string) error {
	wb := db.pebble.NewBatch()

	for _, key := range keys {
		k := store.Key{
			ID:      ulid.ULID{},
			Stream:  store.StreamID(key),
			Version: []byte{},
		}

		pattern, err := packStream(k)
		if err != nil {
			return err
		}

		io := &pebble.IterOptions{}
		it := db.pebble.NewIter(io)
		defer it.Close()
		it.SeekPrefixGE(pattern)

		for ; it.Valid(); it.Next() {
			keyToDel := it.Key()
			if !bytes.HasPrefix(keyToDel, pattern) {
				break
			}
			err = wb.Delete(keyToDel, db.wo)
			if err != nil {
				return err
			}
		}
	}

	err := wb.Commit(db.wo)

	return err
}

// Scan - iterate over the whole store using the handler function
func (db *DB) Scan(scannerOpt store.ScannerOptions) error {
	var prefix []byte
	// Index scan for time
	if scannerOpt.Index {
		prefix = indexScanPrefix(scannerOpt.Prefix)
	} else {
		prefix = streamScanPrefix(scannerOpt.Prefix)
	}

	// Create an upper bound by increasing the last value of the prefix (eg : to ;)
	prefixLen := len(prefix)
	upperBound := make([]byte, prefixLen)
	copy(upperBound, prefix)
	upperBound[prefixLen-1]++

	io := &pebble.IterOptions{
		UpperBound: upperBound,
	}
	it := db.pebble.NewIter(io)
	defer it.Close()

	start := func(it *pebble.Iterator) {
		if len(prefix) == 0 {
			it.First()
		} else {
			it.SeekGE(prefix)
		}
	}

	valid := func(it *pebble.Iterator) bool {
		if !it.Valid() {
			return false
		}

		key := it.Key()
		if len(prefix) != 0 && !bytes.HasPrefix(key, prefix) {
			return false
		}

		return true
	}

	seen := false

	for start(it); valid(it); it.Next() {
		hs := bytes.HasSuffix(it.Key(), scannerOpt.Offset)

		// ignore up to offset
		if !seen && !hs {
			continue
		}

		seen = true

		if hs && !scannerOpt.IncludeOffset {
			continue
		}

		// valid kv pair to pass to the handler
		k := make([]byte, len(it.Key()))
		copy(k, it.Key())

		v := make([]byte, len(it.Value()))
		copy(v, it.Value())

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
}

func streamScanIdentifier() []byte {
	return []byte{'s', ':'}
}

func streamScanPrefix(prefix []byte) []byte {
	if len(prefix) == 0 {
		return streamScanIdentifier()
	}

	buf := make([]byte, len(prefix)+2)
	copy(buf, streamScanIdentifier())
	copy(buf[2:], prefix)

	return buf
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

	var id ulid.ULID
	copy(id[:], v[1])

	k.ID = id
	k.Stream = store.StreamID(v[2])
	k.Version = v[3]

	return k, nil
}
