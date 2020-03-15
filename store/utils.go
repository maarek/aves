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
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var p *sync.Pool

func init() {
	p = &sync.Pool{
		New: func() interface{} {
			return &generator{r: ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)}
		},
	}
}

type generator struct {
	r io.Reader
}

func (g *generator) New() ulid.ULID {
	return ulid.MustNew(ulid.Timestamp(time.Now()), g.r)
}

func GenUlid() ulid.ULID {
	g := p.Get().(*generator)
	id := g.New()
	p.Put(g)
	return id
}
